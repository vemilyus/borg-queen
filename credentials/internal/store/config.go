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

package store

import (
	"errors"
	"github.com/BurntSushi/toml"
	"os"
)

type Config struct {
	StoragePath   string
	ListenAddress string
	Tls           *TlsConfig
}

type TlsConfig struct {
	CertFile string
	KeyFile  string
}

func LoadConfig(path string) (*Config, error) {
	var conf Config
	if _, err := toml.DecodeFile(path, &conf); err != nil {
		return nil, err
	}

	return &conf, nil
}

func InitStoragePath(config *Config) error {
	if config.StoragePath == "" {
		return errors.New("no storage path configured")
	}

	return os.MkdirAll(config.StoragePath, 0o700)
}
