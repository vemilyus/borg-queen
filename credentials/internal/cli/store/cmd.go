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
	"github.com/vemilyus/borg-collective/credentials/internal/cli/config"
	"github.com/vemilyus/borg-collective/credentials/internal/cli/grpcclient"
	"github.com/vemilyus/borg-collective/credentials/internal/cli/utils"
	"github.com/vemilyus/borg-collective/credentials/internal/proto"
	"golang.org/x/crypto/ssh/terminal"
	"os"
)

type Cmd struct {
	*flaggy.Subcommand
	*infoCmd
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
	storeCmd.infoCmd = newInfoCmd(cmd)
	storeCmd.unlockCmd = newUnlockCmd(cmd)
	storeCmd.lockCmd = newLockCmd(cmd)
	storeCmd.setRecoveryRecipientCmd = newSetRecoveryRecipientCmd(cmd)
	storeCmd.exportCmd = newExportCmd(cmd)

	return storeCmd
}

func (cmd *Cmd) Run(state *config.State) {
	state.Config().VerifyConnectionConfig()

	if cmd.infoCmd.Used {
		cmd.infoCmd.run(state)
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

type infoCmd struct {
	*flaggy.Subcommand
	quiet bool
}

func newInfoCmd(parent *flaggy.Subcommand) *infoCmd {
	vCmd := &infoCmd{
		quiet: false,
	}

	cmd := flaggy.NewSubcommand("info")
	cmd.ShortName = "i"
	cmd.Description = "Shows information about the remote store"

	cmd.Bool(&vCmd.quiet, "q", "quiet", "Only print the version number")

	parent.AttachSubcommand(cmd, 1)

	vCmd.Subcommand = cmd

	return vCmd
}

func (cmd *infoCmd) run(state *config.State) {
	storeInfo, err := grpcclient.Run(
		state.Config(),
		func(c grpcclient.GrpcClient) (*proto.StoreInfo, error) {
			return c.GetInfo()
		},
	)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get store info")
	}

	if cmd.quiet {
		_, _ = os.Stdout.WriteString(storeInfo.GetVersion())

		if terminal.IsTerminal(int(os.Stdin.Fd())) {
			println()
		}
	} else {
		log.Info().Msg("Remote store info")
		log.Info().Msgf("    Version: %s", storeInfo.GetVersion())
		log.Info().Msgf("    Locked: %v", storeInfo.GetIsVaultLocked())
		log.Info().Msgf("    Production mode: %v", storeInfo.GetIsProduction())
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

	_, err := grpcclient.Run(
		state.Config(),
		func(c grpcclient.GrpcClient) (any, error) {
			return nil, c.UnlockVault(&proto.AdminCredentials{Passphrase: passphrase.String()})
		},
	)

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
	_, err := grpcclient.Run(
		state.Config(),
		func(c grpcclient.GrpcClient) (any, error) {
			return nil, c.LockVault()
		},
	)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to lock remote store")
	}

	log.Info().Msgf("Locked remote store at %s", state.Config().HostString())
}
