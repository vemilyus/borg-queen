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
	"context"
	"crypto/tls"
	"github.com/rs/zerolog/log"
	"github.com/vemilyus/borg-queen/credentials/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func checkIfProd(config *Config) (bool, error) {
	var opts []grpc.DialOption
	if config.UseTls {
		skipVerify := false
		if config.StoreHost == "::1" || config.StoreHost == "localhost" || config.StoreHost == "127.0.0.1" {
			skipVerify = true
		}

		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: skipVerify})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(config.HostString(), opts...)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create gRPC client")
	}

	client := proto.NewCredStoreClient(conn)
	defer func() { _ = conn.Close() }()

	info, err := client.GetInfo(context.Background(), &proto.Unit{})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to retrieve store info")
	}

	return info.GetIsProduction(), nil
}
