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
	"github.com/google/uuid"
	"github.com/vemilyus/borg-queen/credentials/internal/store/vault"
)

type PassphraseRequest struct {
	Passphrase string `json:"passphrase"`
}

type ClientCredentialsRequest struct {
	Id     uuid.UUID `json:"id"`
	Secret string    `json:"secret"`
}

type CreateClientCredentialsRequest struct {
	PassphraseRequest
	Description string `json:"description"`
}

type CreateClientCredentialsResponse struct {
	ClientCredentialsRequest
}

type ListVaultItemsRequest struct {
	PassphraseRequest
	DescriptionContains *string `json:"descriptionContains"`
}

type ListVaultItemsResponse struct {
	Items []vault.Item `json:"items"`
}

type ReadVaultItemRequest struct {
	PassphraseRequest
	ItemId uuid.UUID `json:"itemId"`
}

type ClientReadVaultItemRequest struct {
	ClientCredentialsRequest
	ItemId         uuid.UUID  `json:"itemId"`
	VerificationId *uuid.UUID `json:"verificationId"`
}

type ReadVaultItemResponse struct {
	Value          []byte     `json:"value"`
	VerificationId *uuid.UUID `json:"verificationId"`
}

type DeleteVaultItemsRequest struct {
	PassphraseRequest
	ItemIds []uuid.UUID `json:"itemIds"`
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
