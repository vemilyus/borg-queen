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

type SetupRequest struct {
	Passphrase string `json:"passphrase"`
}

type UnlockRequest struct {
	Passphrase string `json:"passphrase"`
}

type LoginRequest struct {
	Passphrase  string   `json:"passphrase"`
	Credentials []string `json:"credentials"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type VersionResponse struct {
	Version      string `json:"version"`
	IsProduction bool   `json:"isProduction"`
}
