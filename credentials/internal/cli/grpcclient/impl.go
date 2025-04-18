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
	"errors"
	"github.com/vemilyus/borg-collective/credentials/internal/proto"
	"google.golang.org/grpc/status"
	"io"
)

type grpcClientImpl struct {
	client proto.CredStoreClient
	ctx    context.Context
}

func (g *grpcClientImpl) GetInfo() (*proto.StoreInfo, error) {
	info, err := g.client.GetInfo(g.ctx, &proto.Unit{})
	if err != nil {
		return nil, unpackError(err)
	}

	return info, nil
}

func (g *grpcClientImpl) UnlockVault(credentials *proto.AdminCredentials) error {
	if _, err := g.client.UnlockVault(g.ctx, credentials); err != nil {
		return unpackError(err)
	}

	return nil
}

func (g *grpcClientImpl) LockVault() error {
	if _, err := g.client.LockVault(g.ctx, &proto.Unit{}); err != nil {
		return unpackError(err)
	}

	return nil
}

func (g *grpcClientImpl) SetRecoveryRecipient(recipient *proto.RecoveryRecipient) error {
	if _, err := g.client.SetRecoveryRecipient(g.ctx, recipient); err != nil {
		return unpackError(err)
	}

	return nil
}

func (g *grpcClientImpl) CreateVaultItem(creation *proto.ItemCreation) (*proto.Item, error) {
	item, err := g.client.CreateVaultItem(g.ctx, creation)
	if err != nil {
		return nil, unpackError(err)
	}

	return item, nil
}

func (g *grpcClientImpl) ListVaultItems(search *proto.ItemSearch) ([]*proto.Item, error) {
	stream, err := g.client.ListVaultItems(g.ctx, search)
	if err != nil {
		return nil, unpackError(err)
	}

	var items []*proto.Item
	for {
		item, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, unpackError(err)
		}

		items = append(items, item)
	}

	return items, nil
}

func (g *grpcClientImpl) DeleteVaultItems(deletion *proto.ItemDeletion) ([]string, error) {
	stream, err := g.client.DeleteVaultItems(g.ctx, deletion)
	if err != nil {
		return nil, unpackError(err)
	}

	var items []string
	for {
		item, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, unpackError(err)
		}

		items = append(items, item.GetId())
	}

	return items, nil
}

func (g *grpcClientImpl) ReadVaultItem(request *proto.ItemRequest) (*proto.ItemValue, error) {
	value, err := g.client.ReadVaultItem(g.ctx, request)
	if err != nil {
		return nil, unpackError(err)
	}

	return value, nil
}

func (g *grpcClientImpl) CreateClientCredentials(creation *proto.ClientCreation) (*proto.ClientCredentials, error) {
	creds, err := g.client.CreateClientCredentials(g.ctx, creation)
	if err != nil {
		return nil, unpackError(err)
	}

	return creds, nil
}

func unpackError(err error) error {
	s, ok := status.FromError(err)
	if !ok {
		return errors.New(err.Error())
	}

	return errors.New(s.Message())
}
