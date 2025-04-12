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

package model

import (
	"github.com/awnumar/memguard"
	"github.com/google/uuid"
	"github.com/vemilyus/borg-queen/credentials/internal/store/vault"
	"unsafe"
)

type PassphraseRequest struct {
	Passphrase string `json:"passphrase"`
}

func (p *PassphraseRequest) Wipe() {
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(&p.Passphrase)))
}

type ClientCredentialsRequest struct {
	Id     uuid.UUID `json:"id"`
	Secret string    `json:"secret"`
}

func (c *ClientCredentialsRequest) Wipe() {
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(&c.Secret)))
}

type CreateClientCredentialsRequest struct {
	PassphraseRequest
	Description string `json:"description"`
}

func (c *CreateClientCredentialsRequest) Wipe() {
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(&c.Passphrase)))
}

type CreateClientCredentialsResponse struct {
	ClientCredentialsRequest
}

func (c *CreateClientCredentialsResponse) Wipe() {
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(&c.Secret)))
}

type SetRecoveryRecipientRequest struct {
	PassphraseRequest
	Recipient string `json:"recipient"`
}

func (s *SetRecoveryRecipientRequest) Wipe() {
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(&s.Passphrase)))
}

type ListVaultItemsRequest struct {
	PassphraseRequest
	DescriptionContains *string `json:"descriptionContains"`
}

func (l *ListVaultItemsRequest) Wipe() {
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(&l.Passphrase)))
}

type ListVaultItemsResponse struct {
	Items []vault.Item `json:"items"`
}

type CreateVaultItemRequest struct {
	PassphraseRequest
	Description string `json:"description"`
	Data        []byte `json:"data"`
}

func (c *CreateVaultItemRequest) Wipe() {
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(&c.Passphrase)))
}

type CreateVaultItemResponse struct {
	ItemId uuid.UUID `json:"itemId"`
}

type ReadVaultItemRequest struct {
	PassphraseRequest
	ItemId uuid.UUID `json:"itemId"`
}

func (r *ReadVaultItemRequest) Wipe() {
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(&r.Passphrase)))
}

type ClientReadVaultItemRequest struct {
	ClientCredentialsRequest
	ItemId         uuid.UUID  `json:"itemId"`
	VerificationId *uuid.UUID `json:"verificationId"`
}

func (c *ClientReadVaultItemRequest) Wipe() {
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(&c.Secret)))
}

type ReadVaultItemResponse struct {
	Value          []byte     `json:"value"`
	VerificationId *uuid.UUID `json:"verificationId"`
}

func (c *ReadVaultItemResponse) Wipe() {
	memguard.WipeBytes(c.Value)
}

type DeleteVaultItemsRequest struct {
	PassphraseRequest
	ItemIds []uuid.UUID `json:"itemIds"`
}

func (d *DeleteVaultItemsRequest) Wipe() {
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(&d.Passphrase)))
}

type DeleteVaultItemsResponse struct {
	DeletedItemIds []uuid.UUID `json:"deletedItemIds"`
}

type VersionResponse struct {
	Version      string `json:"version"`
	IsProduction bool   `json:"isProduction"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
