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
	"bytes"
	"github.com/google/uuid"
	"github.com/integrii/flaggy"
	"github.com/rs/zerolog/log"
	"github.com/vemilyus/borg-collective/credentials/internal/cli/config"
	"github.com/vemilyus/borg-collective/credentials/internal/cli/grpcclient"
	"github.com/vemilyus/borg-collective/credentials/internal/cli/utils"
	"github.com/vemilyus/borg-collective/credentials/internal/proto"
	"slices"
	"strings"
)

type createVaultItemCmd struct {
	*flaggy.Subcommand
	description string
}

func newCreateVaultItemCmd(parent *flaggy.Subcommand) *createVaultItemCmd {
	createCmd := &createVaultItemCmd{
		description: "",
	}

	cmd := flaggy.NewSubcommand("create")
	cmd.ShortName = "c"
	cmd.Description = "Creates a new vault item to securely store a secret value"

	cmd.String(&createCmd.description, "d", "description", "Description of the vault item")

	parent.AttachSubcommand(cmd, 1)

	createCmd.Subcommand = cmd

	return createCmd
}

func (cmd *createVaultItemCmd) run(state *config.State) {
	var err error

	cmd.description = strings.TrimSpace(cmd.description)
	if cmd.description == "" {
		cmd.description, err = utils.Prompt("Enter a description", "")
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to prompt for description")
		}
	}

	secret, err := utils.PromptSecure("Enter the secret value")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to prompt for secret value")
	}

	defer secret.Destroy()

	secretVerify, err := utils.PromptSecure("Confirm secret value")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to prompt for secret value confirmation")
	}

	defer secretVerify.Destroy()

	if !bytes.Equal(secret.Bytes(), secretVerify.Bytes()) {
		log.Fatal().Msg("Secret value mismatch")
	}

	passphrase := state.Config().Passphrase
	if passphrase == nil {
		passphrase = utils.AskForPassphrase()
		defer passphrase.Destroy()
	}

	item, err := grpcclient.Run(
		state.Config(),
		func(c grpcclient.GrpcClient) (*proto.Item, error) {
			return c.CreateVaultItem(&proto.ItemCreation{
				Credentials: &proto.AdminCredentials{Passphrase: passphrase.String()},
				Description: cmd.description,
				Value:       secret.Bytes(),
			})
		},
	)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create vault item")
	}

	log.Info().Msgf("Created vault item with ID: %s", item.Id)
}

type deleteVaultItemsCmd struct {
	*flaggy.Subcommand
	firstItemId string
}

func newDeleteVaultItemsCmd(parent *flaggy.Subcommand) *deleteVaultItemsCmd {
	deleteCmd := &deleteVaultItemsCmd{}

	cmd := flaggy.NewSubcommand("delete")
	cmd.Description = "Deletes the specified items"

	cmd.AddPositionalValue(&deleteCmd.firstItemId, "ITEM-IDS", 1, true, "IDs of items to delete")

	parent.AttachSubcommand(cmd, 1)

	deleteCmd.Subcommand = cmd

	return deleteCmd
}

func (cmd *deleteVaultItemsCmd) run(state *config.State) {
	var itemIds []string

	rawItemIds := append([]string{}, cmd.firstItemId)
	rawItemIds = append(rawItemIds, flaggy.TrailingArguments...)

	for _, item := range rawItemIds {
		parsed, err := uuid.Parse(item)
		if err != nil {
			log.Warn().Err(err).Msgf("Failed to parse item id: %s", item)
			continue
		}

		itemIds = append(itemIds, parsed.String())
	}

	if len(itemIds) == 0 {
		log.Info().Msg("No items to delete")
		return
	}

	log.Info().Msgf("Preparing to delete %d item(s)", len(itemIds))
	for i, id := range itemIds {
		log.Info().Msgf("[%d] %s", i+1, id)
	}

	doDelete, err := utils.PromptConfirm("Confirm deletion of above items", false)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to confirm deletion")
	}

	if !doDelete {
		log.Info().Msg("Not deleting items, user aborted")
		return
	}

	passphrase := state.Config().Passphrase
	if passphrase == nil {
		passphrase = utils.AskForPassphrase()
		defer passphrase.Destroy()
	}

	deletedItemIds, err := grpcclient.Run(
		state.Config(),
		func(c grpcclient.GrpcClient) ([]string, error) {
			return c.DeleteVaultItems(&proto.ItemDeletion{
				Credentials: &proto.AdminCredentials{Passphrase: passphrase.String()},
				Id:          itemIds,
			})
		})

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to delete items")
	}

	for i, id := range itemIds {
		if slices.Contains(deletedItemIds, id) {
			log.Info().Msgf("[%d] %s DELETED", i+1, id)
		} else {
			log.Warn().Msgf("[%d] %s NOT FOUND", i+1, id)
		}
	}
}
