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
	"crypto/rand"
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/vemilyus/borg-queen/credentials/internal/model"
	"github.com/vemilyus/borg-queen/credentials/internal/store/vault"
	"unsafe"
)

func (s *State) CreateClientCredentials(request model.CreateClientCredentialsRequest) (*model.CreateClientCredentialsResponse, *model.ErrorResponse) {
	err := s.vault.VerifyPassphrase(request.Passphrase)
	if err != nil {
		return nil, &model.ErrorResponse{Message: err.Error()}
	}

	randStr := rand.Text()

	secretBuffer := memguard.NewBufferFromBytes(*(*[]byte)(unsafe.Pointer(&randStr)))
	defer secretBuffer.Destroy()

	item, err := s.vault.CreateItem("CC[" + request.Description + "]")
	if err != nil {
		return nil, &model.ErrorResponse{Message: err.Error()}
	}

	err = s.vault.SetItemValue(item.Id, secretBuffer)
	if err != nil {
		return nil, &model.ErrorResponse{Message: err.Error()}
	}

	secret, err := s.vault.GetItem(item.Id)
	if err != nil {
		return nil, &model.ErrorResponse{Message: err.Error()}
	}
	defer secret.Destroy()

	return &model.CreateClientCredentialsResponse{
		ClientCredentialsRequest: model.ClientCredentialsRequest{
			Id:     item.Id,
			Secret: string(secret.Bytes()),
		},
	}, nil
}

func (s *State) verifyClientCredentials(credentials model.ClientCredentialsRequest) *model.ErrorResponse {
	defer memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(&credentials.Secret)))

	item, err := s.vault.GetItem(credentials.Id)
	if err != nil {
		return &model.ErrorResponse{Message: "client credentials mismatch"}
	}

	defer item.Destroy()

	if credentials.Secret != item.String() {
		return &model.ErrorResponse{Message: "client credentials mismatch"}
	}

	return nil
}

func (s *State) verifyClientRemoteHost(itemId uuid.UUID, verificationId *uuid.UUID, remoteHost string) (uuid.UUID, *model.ErrorResponse) {
	vaultItems := s.vault.Items()
	var verificationItemId uuid.UUID
	for _, item := range vaultItems {
		if item.Description == "VI["+itemId.String()+"]" {
			verificationItemId = item.Id
		}
	}

	var err error
	if verificationItemId == uuid.Nil {
		var item *vault.Item
		item, err = s.vault.CreateItem("VI[" + itemId.String() + "]")
		if err != nil {
			return uuid.Nil, &model.ErrorResponse{Message: "failed to verify client remote host"}
		}

		err = s.vault.SetItemValue(item.Id, memguard.NewBufferFromBytes([]byte(remoteHost)))
		if err != nil {
			return uuid.Nil, &model.ErrorResponse{Message: "failed to verify client remote host"}
		}

		verificationItemId = item.Id
	} else {
		if verificationId == nil {
			log.Warn().Msg(fmt.Sprintf("No verification ID specified by %s while attempting to access %s", remoteHost, itemId))
			return uuid.Nil, &model.ErrorResponse{Message: "failed to verify client remote host"}
		}

		if *verificationId != verificationItemId {
			log.Warn().Msg(fmt.Sprintf("Mismatched verification ID specified by %s while attempting to access %s", remoteHost, itemId))
			return uuid.Nil, &model.ErrorResponse{Message: "failed to verify client remote host"}
		}

		var checkRemoteHost *memguard.LockedBuffer
		checkRemoteHost, err = s.vault.GetItem(*verificationId)
		if err != nil {
			log.Warn().Msg(fmt.Sprintf("Invalid verification ID specified by %s while attempting to access %s", remoteHost, itemId))
			return uuid.Nil, &model.ErrorResponse{Message: "failed to verify client remote host"}
		}

		defer checkRemoteHost.Destroy()

		if remoteHost != checkRemoteHost.String() {
			log.Error().Msg(fmt.Sprintf("CRITICAL: %s attempted to use credentials of %s", remoteHost, checkRemoteHost.String()))
			return uuid.Nil, &model.ErrorResponse{Message: "failed to verify client remote host"}
		}

		verificationItemId = *verificationId
	}

	return verificationItemId, nil
}

func (s *State) ClientReadVaultItem(request model.ClientReadVaultItemRequest, remoteHost string) (*model.ReadVaultItemResponse, *model.ErrorResponse) {
	err := s.verifyClientCredentials(request.ClientCredentialsRequest)
	if err != nil {
		return nil, err
	}

	verificationId, err := s.verifyClientRemoteHost(request.ItemId, request.VerificationId, remoteHost)
	if err != nil {
		return nil, err
	}

	itemValue, rawErr := s.vault.GetItem(request.ItemId)
	if rawErr != nil {
		return nil, &model.ErrorResponse{Message: "failed to read item"}
	}

	defer itemValue.Destroy()

	return &model.ReadVaultItemResponse{
		Value:          itemValue.Bytes()[:],
		VerificationId: &verificationId,
	}, nil
}
