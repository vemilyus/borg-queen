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
	"errors"
	"github.com/awnumar/memguard"
	"github.com/google/uuid"
	"github.com/vemilyus/borg-collective/credentials/internal/proto"
	"unsafe"
)

func (s *State) CreateClientCredentials(request *proto.ClientCreation) (*proto.ClientCredentials, error) {
	err := s.vault.VerifyPassphrase(request.GetCredentials().Passphrase)
	if err != nil {
		return nil, err
	}

	randStr := rand.Text()

	secretBuffer := memguard.NewBufferFromBytes(*(*[]byte)(unsafe.Pointer(&randStr)))
	defer secretBuffer.Destroy()

	item, err := s.vault.CreateItem("CC[" + request.Description + "]")
	if err != nil {
		return nil, err
	}

	err = s.vault.SetItemValue(item.Id, secretBuffer)
	if err != nil {
		return nil, err
	}

	secret, err := s.vault.GetItem(item.Id)
	if err != nil {
		return nil, err
	}
	defer secret.Destroy()

	return &proto.ClientCredentials{
		Id:     item.Id.String(),
		Secret: string(secret.Bytes()),
	}, nil
}

func (s *State) verifyClientCredentials(credentials *proto.ClientCredentials) error {
	defer memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(&credentials.Secret)))

	itemId, err := uuid.Parse(credentials.Id)
	if err != nil {
		return errors.New("client credentials mismatch")
	}

	item, err := s.vault.GetItem(itemId)
	if err != nil {
		return errors.New("client credentials mismatch")
	}

	defer item.Destroy()

	if credentials.Secret != item.String() {
		return errors.New("client credentials mismatch")
	}

	return nil
}
