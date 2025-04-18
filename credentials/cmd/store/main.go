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
	"crypto/tls"
	"github.com/awnumar/memguard"
	"github.com/integrii/flaggy"
	"github.com/rs/zerolog/log"
	"github.com/vemilyus/borg-queen/credentials/internal/logging"
	"github.com/vemilyus/borg-queen/credentials/internal/store"
	"github.com/vemilyus/borg-queen/credentials/internal/store/cert"
	"github.com/vemilyus/borg-queen/credentials/internal/store/server"
	"github.com/vemilyus/borg-queen/credentials/internal/store/service"
	"github.com/vemilyus/borg-queen/credentials/internal/store/vault"
	"net"
)

var (
	version = "unknown"

	prod       bool
	configPath string
)

func main() {
	memguard.CatchInterrupt()
	defer memguard.Purge()

	parseArgs()
	logging.InitLogging(prod)

	config := loadConfig(configPath)

	vaultInstance, err := vault.NewVault(&vault.Options{
		Backend: vault.NewLocalStorageBackend(config.StoragePath),
	})

	if err != nil {
		log.Fatal().Err(err).Send()
	}

	stateInstance := service.NewState(
		config,
		vaultInstance,
		version,
		prod,
	)

	startServer(stateInstance)
}

func parseArgs() {
	flaggy.SetName("credstore")
	flaggy.SetDescription("Securely stores and provides credentials over the network")
	flaggy.SetVersion(version)

	flaggy.Bool(&prod, "p", "production", "Indicates whether to run in production mode (requires TLS config)")
	flaggy.AddPositionalValue(&configPath, "CONFIG-PATH", 1, true, "Path to the configuration file")

	flaggy.Parse()
}

func loadConfig(configPath string) *store.Config {
	config, err := store.LoadConfig(configPath)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	err = store.InitStoragePath(config)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	return config
}

func startServer(state *service.State) {
	grpcServer := server.NewGrpcServer(state)

	config := state.Config()

	var listener net.Listener
	var err error

	if prod {
		if config.Tls == nil {
			log.Fatal().Msg("TLS configuration is not set")
		}

		var certReloader *cert.X509KeyPairReloader
		certReloader, err = cert.NewX509KeyPairReloader(config.Tls.CertFile, config.Tls.KeyFile)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to load TLS certificate")
		}

		tlsConfig := &tls.Config{
			GetCertificate: certReloader.GetCertificate,
			NextProtos:     []string{"h2"},
		}

		listener, err = tls.Listen("tcp", config.ListenAddress, tlsConfig)
	} else {
		listener, err = net.Listen("tcp", config.ListenAddress)
	}

	if err != nil {
		log.Fatal().Err(err).Send()
	}

	if prod {
		log.Info().Msg("Running in production mode")
	}

	log.Info().Msgf("Listening on %s", config.ListenAddress)

	err = grpcServer.Serve(listener)

	log.Fatal().Err(err).Send()
}
