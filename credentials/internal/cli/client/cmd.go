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

package client

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/integrii/flaggy"
	"github.com/rs/zerolog/log"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/config"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/httpclient"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/utils"
	"github.com/vemilyus/borg-queen/credentials/internal/model"
	"slices"
	"strings"
)

type Cmd struct {
	*flaggy.Subcommand
	*createClientCredentialsCmd
	*deleteClientCredentialsCmd
}

func NewCmd() *Cmd {
	clientCmd := &Cmd{}

	cmd := flaggy.NewSubcommand("client")
	cmd.Description = "Create and delete client credentials"

	flaggy.AttachSubcommand(cmd, 1)

	clientCmd.Subcommand = cmd
	clientCmd.createClientCredentialsCmd = newCreateClientCredentialsCmd(cmd)
	clientCmd.deleteClientCredentialsCmd = newDeleteClientCredentialsCmd(cmd)

	return clientCmd
}

func (cmd *Cmd) Run(state *config.State) {
	state.Config().VerifyConnectionConfig()

	if cmd.createClientCredentialsCmd.Used {
		cmd.createClientCredentialsCmd.run(state)
	} else if cmd.deleteClientCredentialsCmd.Used {
		cmd.deleteClientCredentialsCmd.run(state)
	} else {
		flaggy.ShowHelpAndExit("")
	}
}

type createClientCredentialsCmd struct {
	*flaggy.Subcommand
	description string
}

func newCreateClientCredentialsCmd(parent *flaggy.Subcommand) *createClientCredentialsCmd {
	createCmd := &createClientCredentialsCmd{}

	cmd := flaggy.NewSubcommand("create")
	cmd.Description = "Creates a new set of client credentials"

	cmd.String(&createCmd.description, "d", "description", "Description for the client credentials")

	parent.AttachSubcommand(cmd, 1)

	createCmd.Subcommand = cmd

	return createCmd
}

func (cmd *createClientCredentialsCmd) run(state *config.State) {
	actualDescription := strings.TrimSpace(cmd.description)
	if actualDescription == "" {
		log.Fatal().Msg("No description provided")
	}

	passphrase := state.Config().Passphrase
	if passphrase == nil {
		passphrase = utils.AskForPassphrase()
		defer passphrase.Destroy()
	}

	httpClient := httpclient.New(state.Config())

	var clientCredentials model.CreateClientCredentialsResponse
	err := httpClient.Post(model.PathPostClient, model.CreateClientCredentialsRequest{
		PassphraseRequest: model.PassphraseRequest{
			Passphrase: passphrase.String(),
		},
		Description: actualDescription,
	}, &clientCredentials)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create client credentials")
	}

	defer clientCredentials.Wipe()

	log.Info().Msgf("Created client credentials: %s", actualDescription)
	log.Info().Msgf("Client ID:     %s", clientCredentials.Id.String())
	log.Info().Msgf("Client Secret: %s", clientCredentials.Secret)
}

type deleteClientCredentialsCmd struct {
	*flaggy.Subcommand
	clientId string
}

func newDeleteClientCredentialsCmd(parent *flaggy.Subcommand) *deleteClientCredentialsCmd {
	deleteCmd := &deleteClientCredentialsCmd{}

	cmd := flaggy.NewSubcommand("delete")
	cmd.Description = "Deletes credentials for a client"

	cmd.AddPositionalValue(&deleteCmd.clientId, "CLIENT-ID", 1, true, "The ID of the client to delete")

	parent.AttachSubcommand(cmd, 1)

	deleteCmd.Subcommand = cmd

	return deleteCmd
}

func (cmd *deleteClientCredentialsCmd) run(state *config.State) {
	clientId, err := uuid.Parse(cmd.clientId)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse client ID")
	}

	doDelete, err := utils.PromptConfirm(fmt.Sprintf("Confirm deletion of client credentials %s", clientId.String()), true)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	if !doDelete {
		log.Info().Msg("Not deleting client credentials, user aborted")
		return
	}

	passphrase := state.Config().Passphrase
	if passphrase == nil {
		passphrase = utils.AskForPassphrase()
		defer passphrase.Destroy()
	}

	httpClient := httpclient.New(state.Config())

	var response model.DeleteVaultItemsResponse
	err = httpClient.Delete(model.PathDeleteItem, model.DeleteVaultItemsRequest{
		PassphraseRequest: model.PassphraseRequest{
			Passphrase: passphrase.String(),
		},
		ItemIds: []uuid.UUID{clientId},
	}, &response)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to delete client credentials")
	}

	if slices.Contains(response.DeletedItemIds, clientId) {
		log.Info().Msg("Client credentials deleted")
	} else {
		log.Info().Msg("Client credentials not found")
	}
}
