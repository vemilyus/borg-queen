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
	"github.com/vemilyus/borg-queen/credentials/internal/model"
	"github.com/vemilyus/borg-queen/credentials/internal/store"
	"github.com/vemilyus/borg-queen/credentials/internal/store/vault"
)

type State struct {
	config       *store.Config
	vault        *vault.Vault
	version      string
	isProduction bool
}

func NewState(config *store.Config, vault *vault.Vault, version string, prod bool) *State {
	return &State{
		config:       config,
		vault:        vault,
		version:      version,
		isProduction: prod,
	}
}

func (s *State) Config() *store.Config {
	return s.config
}

func (s *State) IsProduction() bool {
	return s.isProduction
}

func (s *State) GetVersion() model.VersionResponse {
	return model.VersionResponse{
		Version:      s.version,
		IsProduction: s.isProduction,
	}
}
