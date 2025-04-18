// Copyright (C) 2025 Alex Katlein
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package item

import (
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/google/uuid"
	"github.com/integrii/flaggy"
	"github.com/rs/zerolog/log"
	"github.com/vemilyus/borg-collective/credentials/internal/cli/config"
	"github.com/vemilyus/borg-collective/credentials/internal/cli/grpcclient"
	"github.com/vemilyus/borg-collective/credentials/internal/cli/utils"
	"github.com/vemilyus/borg-collective/credentials/internal/proto"
	"golang.org/x/term"
	"os"
	"strings"
	"time"
)

type listVaultItemsCmd struct {
	*flaggy.Subcommand
	search string
	idOnly bool
}

func newListVaultItemsCmd(parent *flaggy.Subcommand) *listVaultItemsCmd {
	listCmd := &listVaultItemsCmd{
		search: "",
		idOnly: false,
	}

	cmd := flaggy.NewSubcommand("list")
	cmd.ShortName = "ls"
	cmd.Description = "Lists all items available in the store"

	cmd.AddPositionalValue(&listCmd.search, "SEARCH", 1, false, "Filter by description content")
	cmd.Bool(&listCmd.idOnly, "q", "quiet", "Only display item IDs")

	parent.AttachSubcommand(cmd, 1)

	listCmd.Subcommand = cmd

	return listCmd
}

func (cmd *listVaultItemsCmd) run(state *config.State) {
	cmd.search = strings.TrimSpace(cmd.search)
	actualSearch := &cmd.search
	if cmd.search == "" {
		actualSearch = nil
	}

	passphrase := state.Config().Passphrase
	if passphrase == nil {
		passphrase = utils.AskForPassphrase()
		defer passphrase.Destroy()
	}

	items, err := grpcclient.Run(
		state.Config(),
		func(c grpcclient.GrpcClient) ([]*proto.Item, error) {
			search := &proto.ItemSearch{
				Credentials: &proto.AdminCredentials{Passphrase: passphrase.String()},
			}

			if actualSearch != nil {
				search.Query = *actualSearch
			}

			return c.ListVaultItems(search)
		},
	)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to retrieve list of items")
	}

	log.Info().Msgf("Retrieved %d items", len(items))
	if actualSearch != nil {
		log.Info().Msgf("Used search: %s", *actualSearch)
	}

	if cmd.idOnly {
		for _, item := range items {
			fmt.Println(item.GetId())
		}
	} else {
		for _, item := range items {
			fmt.Printf("%s\t%s\t%s\n", item.GetId(), item.GetDescription(), time.UnixMilli(item.GetCreatedAt()).Format(time.RFC3339))
		}
	}
}

type readVaultItemCmd struct {
	*flaggy.Subcommand
	itemId string
}

func newReadVaultItemCmd(parent *flaggy.Subcommand) *readVaultItemCmd {
	readCmd := &readVaultItemCmd{}

	cmd := flaggy.NewSubcommand("read")
	cmd.ShortName = "r"
	cmd.Description = "Reads an item value"

	cmd.AddPositionalValue(&readCmd.itemId, "ITEM-ID", 1, true, "The ID of the item to read")

	parent.AttachSubcommand(cmd, 1)

	readCmd.Subcommand = cmd

	return readCmd
}

func (cmd *readVaultItemCmd) run(state *config.State) {
	itemId, err := uuid.Parse(cmd.itemId)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse item ID")
	}

	itemRequest := &proto.ItemRequest{
		ItemId: itemId.String(),
	}

	if state.Config().SecureCredentials != nil {
		clientId, err := uuid.Parse(state.Config().SecureCredentials.Id.String())
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to parse client ID")
		}

		defer memguard.WipeBytes(clientId[:])

		itemRequest.Credentials = &proto.ItemRequest_Client{Client: &proto.ClientCredentials{
			Id:     clientId.String(),
			Secret: state.Config().SecureCredentials.Secret.String(),
		}}
	} else if state.Config().Credentials != nil {
		clientId, err := uuid.Parse(state.Config().Credentials.Id)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to parse client ID")
		}

		defer memguard.WipeBytes(clientId[:])

		itemRequest.Credentials = &proto.ItemRequest_Client{Client: &proto.ClientCredentials{
			Id:     clientId.String(),
			Secret: state.Config().Credentials.Secret,
		}}
	} else {
		log.Info().Msg("Reading vault item using passphrase")

		passphrase := state.Config().Passphrase
		if passphrase == nil {
			passphrase = utils.AskForPassphrase()
			defer passphrase.Destroy()
		}

		itemRequest.Credentials = &proto.ItemRequest_Admin{Admin: &proto.AdminCredentials{Passphrase: passphrase.String()}}
	}

	itemValue, err := grpcclient.Run(
		state.Config(),
		func(c grpcclient.GrpcClient) (*proto.ItemValue, error) {
			return c.ReadVaultItem(itemRequest)
		},
	)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read item")
	}

	secret := memguard.NewBufferFromBytes(itemValue.GetValue())
	defer secret.Destroy()

	_, _ = os.Stdout.Write(secret.Bytes())

	if term.IsTerminal(int(os.Stdin.Fd())) {
		println()
	}
}
