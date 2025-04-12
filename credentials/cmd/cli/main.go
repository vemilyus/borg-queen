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

package main

import (
	"github.com/awnumar/memguard"
	"github.com/integrii/flaggy"
	"github.com/rs/zerolog/log"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/client"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/config"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/item"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/store"
	"github.com/vemilyus/borg-queen/credentials/internal/logging"
	"path/filepath"
)

var (
	version = "unknown"

	configDir = ""

	configureCmd = config.NewCmd()
	loginCmd     = config.NewLoginCmd()
	storeCmd     = store.NewCmd()
	clientCmd    = client.NewCmd()
	itemCmd      = item.NewCmd()
)

func main() {
	memguard.CatchInterrupt()
	defer memguard.Purge()

	logging.InitSimpleLogging()

	parseArgs()

	state := createState()
	defer state.Config().Destroy()

	if configureCmd.Used {
		configureCmd.Run(state)
	} else if loginCmd.Used {
		loginCmd.Run(state)
	} else if storeCmd.Used {
		storeCmd.Run(state)
	} else if itemCmd.Used {
		itemCmd.Run(state)
	} else if clientCmd.Used {
		clientCmd.Run(state)
	} else {
		flaggy.ShowHelpAndExit("")
	}
}

func parseArgs() {
	flaggy.ShowHelpOnUnexpectedEnable()

	flaggy.SetName("cred")
	flaggy.SetDescription("Securely interacts with a remote credential store")
	flaggy.SetVersion(version)

	flaggy.String(&configDir, "", "config-dir", "the location where the configuration file is stored")

	flaggy.Parse()
}

func createState() *config.State {
	var actualCfgDir *string
	if configDir != "" {
		actualCfgDir = &configDir
	}

	path, err := config.EnsureConfigPath(actualCfgDir)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	cfg, err := config.Load(path)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	return config.NewState(filepath.Dir(path), cfg)
}
