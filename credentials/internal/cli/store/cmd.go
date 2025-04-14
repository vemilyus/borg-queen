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
	"github.com/integrii/flaggy"
	"github.com/rs/zerolog/log"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/config"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/httpclient"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/utils"
	"github.com/vemilyus/borg-queen/credentials/internal/model"
	"golang.org/x/crypto/ssh/terminal"
	"os"
)

type Cmd struct {
	*flaggy.Subcommand
	*versionCmd
	*unlockCmd
	*lockCmd
	*setRecoveryRecipientCmd
	*exportCmd
}

func NewCmd() *Cmd {
	storeCmd := &Cmd{}

	cmd := flaggy.NewSubcommand("store")
	cmd.Description = "Foundational interactions with a remote store"

	flaggy.AttachSubcommand(cmd, 1)

	storeCmd.Subcommand = cmd
	storeCmd.versionCmd = newVersionCmd(cmd)
	storeCmd.unlockCmd = newUnlockCmd(cmd)
	storeCmd.lockCmd = newLockCmd(cmd)
	storeCmd.setRecoveryRecipientCmd = newSetRecoveryRecipientCmd(cmd)
	storeCmd.exportCmd = newExportCmd(cmd)

	return storeCmd
}

func (cmd *Cmd) Run(state *config.State) {
	state.Config().VerifyConnectionConfig()

	if cmd.versionCmd.Used {
		cmd.versionCmd.run(state)
	} else if cmd.unlockCmd.Used {
		cmd.unlockCmd.run(state)
	} else if cmd.lockCmd.Used {
		cmd.lockCmd.run(state)
	} else if cmd.setRecoveryRecipientCmd.Used {
		cmd.setRecoveryRecipientCmd.run(state)
	} else if cmd.exportCmd.Used {
		cmd.exportCmd.run(state)
	} else {
		flaggy.ShowHelpAndExit("")
	}
}

type versionCmd struct {
	*flaggy.Subcommand
	quiet bool
}

func newVersionCmd(parent *flaggy.Subcommand) *versionCmd {
	vCmd := &versionCmd{
		quiet: false,
	}

	cmd := flaggy.NewSubcommand("version")
	cmd.ShortName = "v"
	cmd.Description = "Shows the current version of the remote store"

	cmd.Bool(&vCmd.quiet, "q", "quiet", "Only print the version number")

	parent.AttachSubcommand(cmd, 1)

	vCmd.Subcommand = cmd

	return vCmd
}

func (cmd *versionCmd) run(state *config.State) {
	httpClient := httpclient.New(state.Config())

	var response model.VersionResponse
	err := httpClient.Get(model.PathGetVersion, &response)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get version")
	}

	if cmd.quiet {
		_, _ = os.Stdout.WriteString(response.Version)

		if terminal.IsTerminal(int(os.Stdin.Fd())) {
			println()
		}
	} else {
		log.Info().Msg("Remote store info")
		log.Info().Msgf("    Version: %s", response.Version)
		log.Info().Msgf("    Locked: %v", response.IsVaultLocked)
		log.Info().Msgf("    Production mode: %v", response.IsProduction)
	}
}

type unlockCmd struct {
	*flaggy.Subcommand
}

func newUnlockCmd(parent *flaggy.Subcommand) *unlockCmd {
	uCmd := &unlockCmd{}

	cmd := flaggy.NewSubcommand("unlock")
	cmd.Description = "Unlocks the remote store for subsequent access"

	parent.AttachSubcommand(cmd, 1)

	uCmd.Subcommand = cmd

	return uCmd
}

func (cmd *unlockCmd) run(state *config.State) {
	log.Info().Msgf("Unlocking remote store at %s", state.Config().HostString())

	passphrase := utils.AskForPassphrase()
	defer passphrase.Destroy()

	httpClient := httpclient.New(state.Config())

	err := httpClient.Post(model.PathPostVaultUnlock, model.PassphraseRequest{Passphrase: passphrase.String()}, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to unlock remote store")
	}

	log.Info().Msgf("Unlocked remote store at %s", state.Config().HostString())
}

type lockCmd struct {
	*flaggy.Subcommand
}

func newLockCmd(parent *flaggy.Subcommand) *lockCmd {
	lCmd := &lockCmd{}

	cmd := flaggy.NewSubcommand("lock")
	cmd.Description = "Locks the remote store to prevent any access"

	parent.AttachSubcommand(cmd, 1)

	lCmd.Subcommand = cmd

	return lCmd
}

func (cmd *lockCmd) run(state *config.State) {
	httpClient := httpclient.New(state.Config())

	err := httpClient.Delete(model.PathDeleteVaultLock, "", nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to lock remote store")
	}

	log.Info().Msgf("Locked remote store at %s", state.Config().HostString())
}
