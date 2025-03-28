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
	"github.com/gofiber/fiber/v2"
	"github.com/integrii/flaggy"
	"github.com/vemilyus/borg-queen/credentials/internal/store"
	"github.com/vemilyus/borg-queen/credentials/internal/store/handlers"
	"github.com/vemilyus/borg-queen/credentials/internal/store/state"
	"github.com/vemilyus/borg-queen/credentials/internal/store/vault"
	"log"
)

var version = "unknown"

var prod bool
var configPath string

func main() {
	memguard.CatchInterrupt()
	defer memguard.Purge()

	parseArgs()
	config := loadConfig(configPath)

	vaultInstance, err := vault.NewVault(&vault.Options{
		StoragePath: config.StoragePath,
	})

	if err != nil {
		log.Fatal(err)
	}

	stateInstance := state.NewState(
		config,
		vaultInstance,
		version,
		prod,
	)

	startServer(stateInstance)
}

func parseArgs() {
	flaggy.SetName("Credential Store")
	flaggy.SetDescription("Securely stores and provides credentials over the network")
	flaggy.SetVersion(version)

	flaggy.Bool(&prod, "p", "production", "Indicates whether to run in production mode (requires TLS config)")
	flaggy.AddPositionalValue(&configPath, "CONFIG-PATH", 0, true, "Path to the configuration file")

	flaggy.Parse()
}

func loadConfig(configPath string) *store.Config {
	config, err := store.LoadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	err = store.InitStoragePath(config)
	if err != nil {
		log.Fatal(err)
	}

	return config
}

func startServer(state *state.State) {
	app := fiber.New(fiber.Config{
		AppName:            "Credential Host",
		CaseSensitive:      true,
		Concurrency:        4,
		DisableKeepalive:   true,
		EnableIPValidation: true,
	})

	handlers.Setup(app, state)

	config := state.Config()

	var err error
	if prod {
		if config.Tls == nil {
			log.Fatal("TLS configuration is not set")
		}

		err = app.ListenTLS(config.ListenAddress, config.Tls.CertFile, config.Tls.KeyFile)
	} else {
		err = app.Listen(config.ListenAddress)
	}

	log.Fatal(err)
}
