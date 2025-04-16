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

package grpcclient

import (
	"context"
	"crypto/tls"
	"github.com/rs/zerolog/log"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/config"
	"github.com/vemilyus/borg-queen/credentials/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type GrpcClient interface {
	GetInfo() (*proto.StoreInfo, error)
	UnlockVault(credentials *proto.AdminCredentials) error
	LockVault() error
	SetRecoveryRecipient(recipient *proto.RecoveryRecipient) error
	CreateVaultItem(creation *proto.ItemCreation) (*proto.Item, error)
	ListVaultItems(search *proto.ItemSearch) ([]*proto.Item, error)
	DeleteVaultItems(deletion *proto.ItemDeletion) ([]string, error)
	ReadVaultItem(request *proto.ItemRequest) (*proto.ItemValue, error)
	CreateClientCredentials(creation *proto.ClientCreation) (*proto.ClientCredentials, error)
}

func Run[T any](config *config.Config, action func(client GrpcClient) (T, error)) (T, error) {
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

	grpcClient := grpcClientImpl{
		client: client,
		ctx:    context.Background(),
	}

	return action(&grpcClient)
}
