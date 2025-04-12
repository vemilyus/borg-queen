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
	"github.com/integrii/flaggy"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/config"
)

type Cmd struct {
	*flaggy.Subcommand
	*listVaultItemsCmd
	*readVaultItemCmd
	*createVaultItemCmd
	*deleteVaultItemsCmd
}

func NewCmd() *Cmd {
	itemCmd := &Cmd{}

	cmd := flaggy.NewSubcommand("item")
	cmd.Description = "Operations to interact with vault items"

	flaggy.AttachSubcommand(cmd, 1)

	itemCmd.Subcommand = cmd
	itemCmd.listVaultItemsCmd = newListVaultItemsCmd(cmd)
	itemCmd.readVaultItemCmd = newReadVaultItemCmd(cmd)
	itemCmd.createVaultItemCmd = newCreateVaultItemCmd(cmd)
	itemCmd.deleteVaultItemsCmd = newDeleteVaultItemsCmd(cmd)

	return itemCmd
}

func (cmd *Cmd) Run(state *config.State) {
	state.Config().VerifyConnectionConfig()

	if cmd.listVaultItemsCmd.Used {
		cmd.listVaultItemsCmd.run(state)
	} else if cmd.readVaultItemCmd.Used {
		cmd.readVaultItemCmd.run(state)
	} else if cmd.createVaultItemCmd.Used {
		cmd.createVaultItemCmd.run(state)
	} else if cmd.deleteVaultItemsCmd.Used {
		cmd.deleteVaultItemsCmd.run(state)
	} else {
		flaggy.ShowHelpAndExit("")
	}
}
