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

package config

import (
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
)

type Config struct {
	StoreHost                string
	StorePort                *uint16
	UseTls                   bool
	Credentials              *Credentials
	SecureCredentials        *SecureCredentials `toml:"-"`
	StorePassphraseInKeyring bool
	Passphrase               *memguard.LockedBuffer `toml:"-"`
}

func (config *Config) HostString() string {
	var storePort uint16
	if config.StorePort != nil {
		storePort = *config.StorePort
	} else if config.UseTls {
		storePort = 443
	} else {
		storePort = 80
	}

	return fmt.Sprintf("%s:%d", config.StoreHost, storePort)
}

func (config *Config) Destroy() {
	if config.Passphrase != nil {
		config.Passphrase.Destroy()
	}

	if config.SecureCredentials != nil {
		config.SecureCredentials.Id.Destroy()
		config.SecureCredentials.Secret.Destroy()
	}
}

type Credentials struct {
	Id     string
	Secret string
}

type SecureCredentials struct {
	Id     *memguard.LockedBuffer
	Secret *memguard.LockedBuffer
}

func Store(parentPath *string, config Config) error {
	path, err := EnsureConfigPath(parentPath)
	if err != nil {
		return err
	}

	configWriter, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	defer func() { _ = configWriter.Close() }()

	encoder := toml.NewEncoder(configWriter)
	if err = encoder.Encode(config); err != nil {
		return err
	}

	config.storeSecureValues()

	return encoder.Encode(config)
}

func (config *Config) storeSecureValues() {
	var err error

	var storePort int32
	if config.StorePort != nil {
		storePort = (int32)(*config.StorePort)
	} else {
		storePort = -1
	}

	if config.SecureCredentials != nil {
		func() {
			idKey := fmt.Sprintf("%s:%d-cred-id", config.StoreHost, storePort)
			if err = setInKeyring(idKey, config.SecureCredentials.Id); err != nil {
				log.Warn().Err(err).Msgf("Failed to store credentials for %s:%d", config.StoreHost, storePort)
				return
			}

			secretKey := fmt.Sprintf("%s:%d-cred-secret", config.StoreHost, storePort)
			if err = setInKeyring(secretKey, config.SecureCredentials.Secret); err != nil {
				log.Warn().Err(err).Msgf("Failed to store credentials for %s:%d", config.StoreHost, storePort)
				_ = setInKeyring(idKey, nil)
			}
		}()
	}

	if config.StorePassphraseInKeyring && config.Passphrase != nil {
		passphraseKey := fmt.Sprintf("%s:%d-passphrase", config.StoreHost, config.StorePort)
		if err = setInKeyring(passphraseKey, config.Passphrase); err != nil {
			log.Warn().Err(err).Msgf("Failed to store passphrase for %s:%d", config.StoreHost, storePort)
		}
	}
}

func Load(path string) (*Config, error) {
	configReader, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}

		return nil, err
	}

	defer func() {
		_ = configReader.Close()
	}()

	decoder := toml.NewDecoder(configReader)

	var conf Config
	if err = decoder.Decode(&conf); err != nil {
		return nil, err
	}

	conf.loadSecureValues()

	return &conf, nil
}

func (config *Config) loadSecureValues() {
	var err error

	var storePort int32
	if config.StorePort != nil {
		storePort = (int32)(*config.StorePort)
	} else {
		storePort = -1
	}

	idKey := fmt.Sprintf("%s:%d-cred-id", config.StoreHost, storePort)
	secretKey := fmt.Sprintf("%s:%d-cred-secret", config.StoreHost, storePort)

	var credId *memguard.LockedBuffer
	var credSecret *memguard.LockedBuffer

	if credId, err = getFromKeyring(idKey); err != nil {
		log.Warn().Err(err).Msgf("Failed to load credentials for %s:%d", config.StoreHost, storePort)
	}

	if credSecret, err = getFromKeyring(secretKey); credId != nil && err != nil {
		log.Warn().Err(err).Msgf("Failed to load credentials for %s:%d", config.StoreHost, storePort)
	}

	if credId != nil && credSecret != nil {
		config.SecureCredentials = &SecureCredentials{
			Id:     credId,
			Secret: credSecret,
		}
	}

	if config.StorePassphraseInKeyring {
		passphraseId := fmt.Sprintf("%s:%d-passphrase", config.StoreHost, storePort)
		var passphrase *memguard.LockedBuffer

		if passphrase, err = getFromKeyring(passphraseId); err != nil {
			log.Warn().Err(err).Msgf("Failed to load passphrase for %s:%d", config.StoreHost, storePort)
		}

		if passphrase != nil {
			config.Passphrase = passphrase
		}
	}
}

func EnsureConfigPath(parentPath *string) (string, error) {
	var parentDir string
	if parentPath != nil {
		parentDir = *parentPath
	} else {
		var err error
		parentDir, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}

	err := os.MkdirAll(parentDir, 0700)
	if err != nil {
		return "", err
	}

	return filepath.Join(parentDir, "config.toml"), nil
}

func (config *Config) VerifyConnectionConfig() {
	if config.StoreHost == "" {
		log.Fatal().Msg("Store host is not configured")
	}

	log.Info().Msgf("Connecting to store at %s", config.HostString())
}
