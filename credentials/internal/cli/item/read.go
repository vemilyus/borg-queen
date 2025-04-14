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
	"github.com/vemilyus/borg-queen/credentials/internal/cli/config"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/httpclient"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/utils"
	"github.com/vemilyus/borg-queen/credentials/internal/model"
	"golang.org/x/term"
	"os"
	"path"
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

	httpClient := httpclient.New(state.Config())

	var response model.ListVaultItemsResponse
	err := httpClient.Post(model.PathGetItemList, model.ListVaultItemsRequest{
		PassphraseRequest:   model.PassphraseRequest{Passphrase: passphrase.String()},
		DescriptionContains: actualSearch,
	}, &response)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to retrieve list of items")
	}

	log.Info().Msgf("Retrieved %d items", len(response.Items))
	if actualSearch != nil {
		log.Info().Msgf("Used search: %s", *actualSearch)
	}

	if cmd.idOnly {
		for _, item := range response.Items {
			fmt.Println(item.Id.String())
		}
	} else {
		for _, item := range response.Items {
			fmt.Printf("%s\t%s\t%s\n", item.Id.String(), item.Description, item.ModifiedAt.Format(time.RFC3339))
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

	verificationId, err := loadVerificationId(state.ConfigDir(), itemId)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to load verification ID for item %s", itemId.String())
	}

	var clientRequest *model.ClientReadVaultItemRequest
	var adminRequest *model.ReadVaultItemRequest
	if state.Config().SecureCredentials != nil {
		clientId, err := uuid.Parse(state.Config().SecureCredentials.Id.String())
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to parse client ID")
		}

		defer memguard.WipeBytes(clientId[:])

		clientRequest = &model.ClientReadVaultItemRequest{
			ClientCredentialsRequest: model.ClientCredentialsRequest{
				Id:     clientId,
				Secret: state.Config().SecureCredentials.Secret.String(),
			},
			ItemId:         itemId,
			VerificationId: verificationId,
		}
	} else if state.Config().Credentials != nil {
		clientId, err := uuid.Parse(state.Config().Credentials.Id)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to parse client ID")
		}

		defer memguard.WipeBytes(clientId[:])

		clientRequest = &model.ClientReadVaultItemRequest{
			ClientCredentialsRequest: model.ClientCredentialsRequest{
				Id:     clientId,
				Secret: state.Config().Credentials.Secret,
			},
			ItemId:         itemId,
			VerificationId: verificationId,
		}
	} else {
		log.Info().Msg("Reading vault item using passphrase")

		passphrase := state.Config().Passphrase
		if passphrase == nil {
			passphrase = utils.AskForPassphrase()
			defer passphrase.Destroy()
		}

		adminRequest = &model.ReadVaultItemRequest{
			PassphraseRequest: model.PassphraseRequest{
				Passphrase: passphrase.String(),
			},
			ItemId: itemId,
		}
	}

	httpClient := httpclient.New(state.Config())

	var readVaultItemResponse model.ReadVaultItemResponse
	if clientRequest != nil {
		log.Info().Msgf("Reading vault item %s using client credentials", itemId.String())

		err = httpClient.Post(model.PathGetReadItem, clientRequest, &readVaultItemResponse)
	} else {
		err = httpClient.Post(model.PathGetItem, adminRequest, &readVaultItemResponse)
	}

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read item")
	}

	if clientRequest != nil && readVaultItemResponse.VerificationId != nil {
		err = storeVerificationId(state.ConfigDir(), itemId, readVaultItemResponse.VerificationId.String())
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to store verification ID")
		}
	}

	secret := memguard.NewBufferFromBytes(readVaultItemResponse.Value)
	defer secret.Destroy()

	_, _ = os.Stdout.Write(secret.Bytes())

	if term.IsTerminal(int(os.Stdin.Fd())) {
		println()
	}
}

func loadVerificationId(parentDir string, itemId uuid.UUID) (*uuid.UUID, error) {
	finalPath := path.Join(parentDir, itemId.String()+".vid")
	var err error
	if _, err = os.Stat(finalPath); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	verificationIdBytes, err := os.ReadFile(finalPath)
	if err != nil {
		return nil, err
	}

	idStr := strings.TrimSpace(string(verificationIdBytes))
	verificationId, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}

	return &verificationId, nil
}

func storeVerificationId(parentDir string, itemId uuid.UUID, verificationId string) error {
	finalPath := path.Join(parentDir, itemId.String()+".vid")
	return os.WriteFile(finalPath, []byte(verificationId), 0600)
}
