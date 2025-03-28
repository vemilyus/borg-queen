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

import "time"

type Metadata struct {
	TokensCredentials map[string][]string `json:"tokens_credentials"`
	Credentials       []string            `json:"credentials"`
}

type Vault struct {
	Credentials map[string]Credential `json:"credentials"`
}

type Credential struct {
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}
