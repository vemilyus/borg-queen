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

package config

import (
	"github.com/integrii/flaggy"
	"github.com/rs/zerolog/log"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/conn"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/utils"
	"strconv"
)

type Cmd struct {
	*flaggy.Subcommand
}

func NewCmd() *Cmd {
	configureCmd := &Cmd{}

	cmd := flaggy.NewSubcommand("configure")
	cmd.Description = "Configure the connection to a credential store"

	flaggy.AttachSubcommand(cmd, 1)

	configureCmd.Subcommand = cmd

	return configureCmd
}

func (cmd *Cmd) Run(state *State) {
	hostname, err := utils.Prompt("Enter the hostname", state.Config().StoreHost)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	state.config.StoreHost = hostname

	currentPort := ""
	if state.config.StorePort != nil {
		currentPort = strconv.Itoa(int(*state.config.StorePort))
	}

	port, err := utils.Prompt("Enter the port", currentPort)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	newPort, err := strconv.Atoi(port)
	if err != nil || newPort < 0 || newPort > 65535 {
		log.Fatal().Err(err).Msg("Invalid port number")
	}

	var storePort *uint16
	if state.config.StorePort != nil {
		tmp := uint16(newPort)
		storePort = &tmp
	}

	isTls, err := conn.CheckIfTls(hostname, storePort)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to check whether store uses TLS")
	}

	if !isTls {
		doProceed, err := utils.PromptConfirm("Store isn't using TLS, do you want to proceed?", false)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		if !doProceed {
			log.Info().Msg("Store isn't using TLS, user aborted")
			return
		}
	}

	state.config.UseTls = isTls

	if storePort != nil {
		if (isTls && *storePort != 443) || (!isTls && *storePort != 80) {
			state.config.StorePort = storePort
		}
	}

	err = Store(&state.configDir, *state.config)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to store configuration")
	}
}

type LoginCmd struct {
	*flaggy.Subcommand
	plainText      bool
	passphraseMode bool
}

func NewLoginCmd() *LoginCmd {
	loginCmd := &LoginCmd{
		plainText:      false,
		passphraseMode: false,
	}

	cmd := flaggy.NewSubcommand("login")
	cmd.Description = "Log in to a credential store"

	cmd.Bool(&loginCmd.plainText, "t", "plain-text", "Store credentials in plain text (except passphrase)")
	cmd.Bool(&loginCmd.passphraseMode, "p", "passphrase", "login using passphrase")

	flaggy.AttachSubcommand(cmd, 1)

	loginCmd.Subcommand = cmd

	return loginCmd
}

func (cmd *LoginCmd) Run(state *State) {
	state.config.VerifyConnectionConfig()

	if cmd.passphraseMode {
		state.config.StorePassphraseInKeyring = !cmd.plainText

		if !cmd.plainText {
			passphrase, err := utils.PromptSecure("Enter passphrase")
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to read passphrase")
			}

			defer passphrase.Destroy()

			state.config.Passphrase = passphrase
		} else {
			log.Warn().Msg("Refusing to store passphrase in plain text, will ask for it when needed")
		}
	} else {
		clientId, err := utils.PromptSecure("Enter Client ID")
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to read client ID")
		}

		defer clientId.Destroy()

		clientSecret, err := utils.PromptSecure("Enter Client Secret")
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to read client secret")
		}

		defer clientSecret.Destroy()

		if cmd.plainText {
			state.config.Credentials = &Credentials{
				Id:     clientId.String(),
				Secret: clientSecret.String(),
			}
		} else {
			state.config.Credentials = nil
			state.config.SecureCredentials = &SecureCredentials{
				Id:     clientId,
				Secret: clientSecret,
			}
		}
	}

	err := Store(&state.configDir, *state.config)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to store configuration")
	}
}
