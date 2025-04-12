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

package store

import (
	"filippo.io/age"
	"github.com/integrii/flaggy"
	"github.com/rs/zerolog/log"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/config"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/httpclient"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/utils"
	"github.com/vemilyus/borg-queen/credentials/internal/model"
)

type setRecoveryRecipientCmd struct {
	*flaggy.Subcommand
}

func newSetRecoveryRecipientCmd(parent *flaggy.Subcommand) *setRecoveryRecipientCmd {
	setCmd := &setRecoveryRecipientCmd{}

	cmd := flaggy.NewSubcommand("recovery-recipient")
	cmd.Description = "Sets a new recovery recipient"

	parent.AttachSubcommand(cmd, 1)

	setCmd.Subcommand = cmd

	return setCmd
}

func (cmd *setRecoveryRecipientCmd) run(state *config.State) {
	log.Warn().Msg("Setting a new recovery recipient is a destructive action and cannot be undone!")
	doSet, err := utils.PromptConfirm("Confirm setting a new recovery recipient", false)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to confirm")
	}

	if !doSet {
		log.Info().Msg("Not setting a new recovery recipient, user aborted")
		return
	}

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to generate identity")
	}

	log.Info().Msg("New recovery identity generated...")
	log.Warn().Msg("Save the new identity now, it will never be shown again!")
	log.Info().Send()

	println(identity.String())

	log.Info().Send()

	doConfirm, err := utils.PromptConfirm("Do you want to set the recovery recipient?", false)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to confirm")
	}

	if !doConfirm {
		log.Warn().Msg("Not setting a new recovery recipient, user aborted")
	}

	passphrase := utils.AskForPassphrase()
	defer passphrase.Destroy()

	httpClient := httpclient.New(state.Config())

	err = httpClient.Post(model.PathPostVaultRecoveryRecipient, model.SetRecoveryRecipientRequest{
		PassphraseRequest: model.PassphraseRequest{
			Passphrase: passphrase.String(),
		},
		Recipient: identity.Recipient().String(),
	}, nil)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to set recovery recipient")
	}

	log.Info().Msg("Successfully set the recovery recipient")
}
