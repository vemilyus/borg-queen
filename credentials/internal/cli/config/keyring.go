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
	goerrors "errors"
	"github.com/99designs/keyring"
	"github.com/awnumar/memguard"
	"github.com/pkg/errors"
	"sync"
)

var (
	config = keyring.Config{
		AllowedBackends: []keyring.BackendType{
			keyring.SecretServiceBackend,
			keyring.KeychainBackend,
			keyring.KeyCtlBackend,
			keyring.WinCredBackend,
		},
		ServiceName:                    "borg-queen-credentials",
		KeychainName:                   "borg-queen-credentials",
		KeychainTrustApplication:       true,
		KeychainSynchronizable:         false,
		KeychainAccessibleWhenUnlocked: false,
		KWalletAppID:                   "borg-queen-credentials",
		KWalletFolder:                  "borg-queen-credentials",
		WinCredPrefix:                  "borg-queen-credentials",
	}

	create    sync.Once
	ring      keyring.Keyring
	createErr error
)

func initKeyring() error {
	create.Do(func() {
		ring, createErr = keyring.Open(config)

		if createErr != nil {
			createErr = errors.Wrap(createErr, "error opening keyring")
		}
	})

	return createErr
}

func setInKeyring(name string, value *memguard.LockedBuffer) error {
	if err := initKeyring(); err != nil {
		return err
	}

	var err error
	if value == nil {
		err = ring.Remove(name)
	} else {
		err = ring.Set(keyring.Item{
			Key:  name,
			Data: value.Bytes(),
		})
	}

	if err != nil {
		err = errors.Wrap(err, "error storing value in keyring: "+name)
	}

	return err
}

func getFromKeyring(name string) (*memguard.LockedBuffer, error) {
	if err := initKeyring(); err != nil {
		return nil, err
	}

	item, err := ring.Get(name)
	if goerrors.Is(err, keyring.ErrKeyNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "error retrieving value from keyring: "+name)
	}

	return memguard.NewBufferFromBytes(item.Data), nil
}
