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

package service

import (
	"filippo.io/age"
	"github.com/awnumar/memguard"
	"github.com/google/uuid"
	"github.com/vemilyus/borg-queen/credentials/internal/model"
)

func (s *State) SetRecoveryRecipient(request model.SetRecoveryRecipientRequest) *model.ErrorResponse {
	err := s.vault.VerifyPassphrase(request.Passphrase)
	if err != nil {
		return &model.ErrorResponse{Message: err.Error()}
	}

	var recipient *age.X25519Recipient
	recipient, err = age.ParseX25519Recipient(request.Recipient)
	if err != nil {
		return &model.ErrorResponse{Message: "invalid recipient: " + err.Error()}
	}

	err = s.vault.SetRecoveryRecipient(*recipient)
	if err != nil {
		return &model.ErrorResponse{Message: err.Error()}
	}

	return nil
}

func (s *State) CreateVaultItem(request model.CreateVaultItemRequest) (*model.CreateVaultItemResponse, *model.ErrorResponse) {
	err := s.vault.VerifyPassphrase(request.Passphrase)
	if err != nil {
		return nil, &model.ErrorResponse{Message: err.Error()}
	}

	item, err := s.vault.CreateItem(request.Description)
	if err != nil {
		return nil, &model.ErrorResponse{Message: err.Error()}
	}

	itemValue := memguard.NewBufferFromBytes(request.Data)

	err = s.vault.SetItemValue(item.Id, itemValue)
	if err != nil {
		_ = s.vault.DeleteItem(item.Id)
		return nil, &model.ErrorResponse{Message: err.Error()}
	}

	return &model.CreateVaultItemResponse{ItemId: item.Id}, nil
}

func (s *State) ListVaultItems(request model.ListVaultItemsRequest) (*model.ListVaultItemsResponse, *model.ErrorResponse) {
	err := s.vault.VerifyPassphrase(request.Passphrase)
	if err != nil {
		return nil, &model.ErrorResponse{Message: err.Error()}
	}

	items := s.vault.Items()

	return &model.ListVaultItemsResponse{Items: items}, nil
}

func (s *State) ReadVaultItem(request model.ReadVaultItemRequest) (*model.ReadVaultItemResponse, *model.ErrorResponse) {
	err := s.vault.VerifyPassphrase(request.Passphrase)
	if err != nil {
		return nil, &model.ErrorResponse{Message: err.Error()}
	}

	value, err := s.vault.GetItem(request.ItemId)
	if err != nil {
		return nil, &model.ErrorResponse{Message: err.Error()}
	}

	defer value.Destroy()

	return &model.ReadVaultItemResponse{Value: value.Bytes()[:]}, nil
}

func (s *State) DeleteVaultItems(request model.DeleteVaultItemsRequest) (*model.DeleteVaultItemsResponse, *model.ErrorResponse) {
	err := s.vault.VerifyPassphrase(request.Passphrase)
	if err != nil {
		return nil, &model.ErrorResponse{Message: err.Error()}
	}

	deletedItemIds := make([]uuid.UUID, 0, len(request.ItemIds))

	for _, id := range request.ItemIds {
		err = s.vault.DeleteItem(id)
		if err != nil {
			return nil, &model.ErrorResponse{Message: err.Error()}
		}

		deletedItemIds = append(deletedItemIds, id)
	}

	return &model.DeleteVaultItemsResponse{DeletedItemIds: deletedItemIds}, nil
}
