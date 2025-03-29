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

package vault

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"filippo.io/age"
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
	"unsafe"
)

type Options struct {
	StoragePath string
}

type Item struct {
	Id          uuid.UUID `json:"id"`
	Description string    `json:"description"`
	Checksum    string    `json:"checksum"`
	ModifiedAt  time.Time `json:"modified_at"`
}

type Vault struct {
	lock               sync.RWMutex
	options            *Options
	identityPath       string
	identityKey        *memguard.Enclave
	metadataHmacSecret *memguard.Enclave
	primaryRecipient   *age.X25519Recipient
	recoveryRecipient  *age.X25519Recipient
	items              map[uuid.UUID]Item
}

func (v *Vault) Options() *Options {
	return v.options
}

func (v *Vault) IsLocked() bool {
	return v.identityKey == nil
}

func NewVault(options *Options) (*Vault, error) {
	err := os.MkdirAll(options.StoragePath, 0700)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage path (%s): %v", options.StoragePath, err)
	}

	backupPath := filepath.Join(options.StoragePath, ".bak")
	err = os.MkdirAll(backupPath, 0700)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup path (%s): %v", backupPath, err)
	}

	recoveryRecipientPath := filepath.Join(options.StoragePath, ".recovery")
	recoveryRecipient, err := loadRecoveryRecipient(recoveryRecipientPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load recovery recipient (%s): %v", recoveryRecipientPath, err)
	}

	return &Vault{
		lock:               sync.RWMutex{},
		options:            options,
		identityPath:       filepath.Join(options.StoragePath, ".identity"),
		identityKey:        nil,
		metadataHmacSecret: nil,
		primaryRecipient:   nil,
		recoveryRecipient:  recoveryRecipient,
		items:              nil,
	}, nil
}

func (v *Vault) Unlock(passphrase string) error {
	v.lock.Lock()
	defer v.lock.Unlock()

	if !v.IsLocked() {
		return nil
	}

	passphraseBytes := *(*[]byte)(unsafe.Pointer(&passphrase))
	rawSum := sha256.Sum256(passphraseBytes)
	memguard.WipeBytes(passphraseBytes)

	v.identityKey = func() *memguard.Enclave {
		defer wipeSum(rawSum)
		return memguard.NewEnclave(rawSum[:])
	}()

	var err error
	identityFile := filepath.Join(v.options.StoragePath, ".identity")
	if _, err = os.Stat(identityFile); err == nil {
		identityKey, _ := v.identityKey.Open()
		defer identityKey.Destroy()

		identity, err := readIdentity(identityFile, identityKey)
		if err != nil {
			v.identityKey = nil

			log.Error().Err(err).Str("source", identityFile).Msg("failed to read identity file")
			return errors.New("failed to verify passphrase")
		}

		v.metadataHmacSecret = deriveMetadataHmacSecret(*identity)
		v.primaryRecipient = identity.Recipient()
	} else if os.IsNotExist(err) {
		identity, err := age.GenerateX25519Identity()
		if err != nil {
			v.identityKey = nil

			log.Error().Err(err).Msg("failed to generate primary identity")
			return errors.New("failed to verify passphrase")
		}

		identityKey, _ := v.identityKey.Open()
		defer identityKey.Destroy()

		err = writeIdentity(identityFile, identityKey, identity)
		if err != nil {
			log.Err(err).Str("target", identityFile).Msg("failed to write identity")
			return errors.New("failed to verify passphrase")
		}

		v.metadataHmacSecret = deriveMetadataHmacSecret(*identity)
		v.primaryRecipient = identity.Recipient()
	} else {
		v.identityKey = nil

		log.Error().Err(err).Str("source", identityFile).Msg("failed to stat identity file")
		return errors.New("failed to verify passphrase")
	}

	metadataHmacSecret, err := v.metadataHmacSecret.Open()
	if err != nil {
		v.identityKey = nil
		v.metadataHmacSecret = nil
		v.primaryRecipient = nil

		log.Error().Err(err).Msg("failed to access metadata HMAC secret")
		return errors.New("failed to verify passphrase")
	}

	defer metadataHmacSecret.Destroy()

	v.items, err = readAllMetadataUnsafe(v.options.StoragePath, metadataHmacSecret)
	if err != nil {
		v.identityKey = nil
		v.metadataHmacSecret = nil
		v.primaryRecipient = nil
		v.items = nil

		log.Error().Err(err).Msg("failed to read all item metadata")
		return errors.New("failed to verify passphrase")
	}

	return nil
}

func (v *Vault) Lock() error {
	v.lock.Lock()
	defer v.lock.Unlock()

	if v.IsLocked() {
		return errors.New("vault is locked")
	}

	v.identityKey = nil
	v.metadataHmacSecret = nil
	v.primaryRecipient = nil
	v.items = nil

	return nil
}

func (v *Vault) Items() []Item {
	v.lock.RLock()
	defer v.lock.RUnlock()

	return slices.Collect(maps.Values(v.items))
}

func (v *Vault) SetRecoveryRecipient(recipient age.X25519Recipient) error {
	v.lock.Lock()
	defer v.lock.Unlock()

	if v.IsLocked() {
		return errors.New("vault is locked")
	}

	metadataHmacSecret, err := v.metadataHmacSecret.Open()
	if err != nil {
		log.Error().Err(err).Msg("failed to access metadata HMAC secret")
		return errors.New("failed to set recovery recipient")
	}

	defer metadataHmacSecret.Destroy()

	recoveryRecipientPath := filepath.Join(v.options.StoragePath, ".recovery")
	if err := writeRecoveryRecipient(recoveryRecipientPath, recipient); err != nil {
		log.Error().Err(err).Str("target", recoveryRecipientPath).Msg("failed to write recovery recipient")

		if v.recoveryRecipient != nil {
			for i := 0; i < 3; i++ {
				time.Sleep(time.Second)

				err = writeRecoveryRecipient(recoveryRecipientPath, *v.recoveryRecipient)
				if err == nil {
					break
				}
			}

			if err != nil {
				log.Fatal().Err(err).Msg("failed to restore previous recovery recipient")
			}
		}

		return errors.New("failed to set recovery recipient")
	}

	v.recoveryRecipient = &recipient

	items, err := readAllMetadataUnsafe(v.options.StoragePath, metadataHmacSecret)
	metadataHmacSecret.Destroy()

	if err != nil {
		log.Error().Err(err).Msg("failed to read all item metadata")
		return errors.New("failed to set recovery recipient")
	}

	for _, item := range items {
		func() {
			value, err := v.readItemValueUnsafe(item)
			if err != nil {
				log.Error().Err(err).Str("item", item.Id.String()).Msg("failed to read item value")
				return
			}

			defer value.Destroy()

			err = v.writeItemValueUnsafe(item, value)
			if err != nil {
				log.Error().Err(err).Str("item", item.Id.String()).Msg("failed to write item value")
			}
		}()
	}

	return nil
}

func (v *Vault) CreateItem(description string) (*Item, error) {
	v.lock.Lock()
	defer v.lock.Unlock()

	if v.IsLocked() {
		return nil, errors.New("vault is locked")
	}

	id := uuid.New()
	item := Item{
		Id:          id,
		Description: description,
		Checksum:    "",
		ModifiedAt:  time.Now(),
	}

	metadataHmacSecret, err := v.metadataHmacSecret.Open()
	if err != nil {
		log.Error().Err(err).Msg("failed to access metadata HMAC secret")
		return nil, errors.New("failed to create item")
	}

	defer metadataHmacSecret.Destroy()

	metadataPath := v.metadataPath(item)
	if err := writeItemMetadataUnsafe(metadataPath, item, metadataHmacSecret); err != nil {
		log.Error().Err(err).Str("target", metadataPath).Msg("failed to write item metadata")
		return nil, errors.New("failed to create item")
	}

	v.items[id] = item

	return &item, nil
}

func (v *Vault) DeleteItem(id uuid.UUID) error {
	v.lock.Lock()
	defer v.lock.Unlock()

	if v.IsLocked() {
		return errors.New("vault is locked")
	}

	ok := v.deleteItemUnsafe(id)
	if !ok {
		log.Warn().Str("item", id.String()).Msg("no such item")
	}

	return nil
}

func (v *Vault) GetItem(id uuid.UUID) (*memguard.LockedBuffer, error) {
	v.lock.RLock()
	defer v.lock.RUnlock()

	if v.IsLocked() {
		return nil, errors.New("vault is locked")
	}

	item, ok := v.items[id]
	if !ok {
		return nil, errors.New("item not found")
	}

	if item.Checksum == "" {
		return nil, nil
	}

	return v.readItemValueUnsafe(item)
}

func (v *Vault) SetItemValue(id uuid.UUID, value *memguard.LockedBuffer) error {
	if len(value.Bytes()) == 0 {
		return errors.New("value is empty")
	}
	defer value.Destroy()

	v.lock.Lock()
	defer v.lock.Unlock()

	if v.IsLocked() {
		return errors.New("vault is locked")
	}

	item, ok := v.items[id]
	if !ok {
		return errors.New("item not found")
	}

	return v.writeItemValueUnsafe(item, value)
}

func (v *Vault) WriteItemValue(id uuid.UUID, r io.Reader) error {
	buf, err := memguard.NewBufferFromEntireReader(r)
	if err != nil {
		return err
	}

	return v.SetItemValue(id, buf)
}

func (v *Vault) readItemValueUnsafe(item Item) (*memguard.LockedBuffer, error) {
	ageBytes, err := os.ReadFile(v.valuePath(item))
	if err != nil {
		return nil, fmt.Errorf("failed to read item value (%s): %v", item.Id, err)
	}

	value, err := v.decryptFromRestUnsafe(ageBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to read item value (%s): %v", item.Id, err)
	}

	decryptSum := sum(value.Bytes())
	if decryptSum != item.Checksum {
		value.Destroy()
		return nil, fmt.Errorf("failed to read item value (%s): checksum mismatch", item.Id)
	}

	return value, nil
}

func (v *Vault) writeItemValueUnsafe(item Item, value *memguard.LockedBuffer) error {
	ageBytes, err := v.encryptForRestUnsafe(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt item value (%s): %v", item.Id, err)
	}

	valuePath := v.valuePath(item)

	if item.Checksum != "" {
		backupPath := v.backupPath(item)
		if err := copyFile(valuePath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup of previous value (%s): %v", item.Id, err)
		}
	}

	checksum := sum(value.Bytes())
	item.Checksum = checksum
	item.ModifiedAt = time.Now()

	err = os.WriteFile(valuePath, ageBytes, 0600)
	if err != nil {
		return fmt.Errorf("failed to write item value (%s): %v", item.Id, err)
	}

	metadataHmacSecret, err := v.metadataHmacSecret.Open()
	if err != nil {
		log.Error().Err(err).Msg("failed to access metadata HMAC secret")
		return fmt.Errorf("failed to write item value (%s): %v", item.Id, err)
	}

	defer metadataHmacSecret.Destroy()

	err = writeItemMetadataUnsafe(v.metadataPath(item), item, metadataHmacSecret)
	if err != nil {
		return fmt.Errorf("failed to write item metadata (%s): %v", item.Id, err)
	}

	v.items[item.Id] = item

	return nil
}

func (v *Vault) deleteItemUnsafe(id uuid.UUID) bool {
	item, ok := v.items[id]
	if !ok {
		return false
	}

	delete(v.items, id)

	removed := false

	metadataPath := v.metadataPath(item)
	if _, err := os.Stat(metadataPath); err == nil {
		err = os.Remove(metadataPath)
		if err != nil {
			log.Debug().
				Err(err).
				Str("item", item.Id.String()).
				Str("target", metadataPath).
				Msg("failed to delete item metadata file")
		} else {
			removed = true
		}
	}

	valuePath := v.valuePath(item)
	if _, err := os.Stat(valuePath); err == nil {
		err = os.Remove(valuePath)
		if err != nil {
			log.Debug().
				Err(err).
				Str("item", item.Id.String()).
				Str("target", valuePath).
				Msg("failed to delete item value file")
		} else {
			removed = true
		}
	}

	if removed {
		log.Info().Str("item", item.Id.String()).Msg("removed files for item")
	}

	return true
}

func (v *Vault) decryptFromRestUnsafe(data []byte) (*memguard.LockedBuffer, error) {
	identityKey, _ := v.identityKey.Open()
	defer identityKey.Destroy()

	identity, err := readIdentity(v.identityPath, identityKey)
	if err != nil {
		log.Fatal().Err(err).Msg("error reading identity")
	}

	reader, err := age.Decrypt(bytes.NewReader(data), identity)
	if err != nil {
		log.Fatal().Err(err).Msg("error decrypting data")
	}

	out := bytes.Buffer{}
	defer wipeBuffer(out, out.Len())

	if _, err := io.Copy(&out, reader); err != nil {
		log.Fatal().Err(err).Msg("error decrypting data")
	}

	result := make([]byte, out.Len())
	copy(result, out.Bytes())

	return memguard.NewBufferFromBytes(result), nil
}

func (v *Vault) encryptForRestUnsafe(data *memguard.LockedBuffer) ([]byte, error) {
	var recipients []age.Recipient
	recipients = append(recipients, v.primaryRecipient)
	if v.recoveryRecipient != nil {
		recipients = append(recipients, v.recoveryRecipient)
	}

	out := &bytes.Buffer{}
	wc, err := age.Encrypt(out, recipients...)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to encrypt data")
	}

	_, err = io.Copy(wc, bytes.NewReader(data.Bytes()))
	if err != nil {
		log.Fatal().Err(err).Msg("error writing data")
	}

	err = wc.Close()
	if err != nil {
		log.Fatal().Err(err).Msg("error closing writer")
	}

	return out.Bytes(), nil
}

func (v *Vault) backupPath(item Item) string {
	return filepath.Join(v.options.StoragePath, ".bak", fmt.Sprintf("%s.%d.json", item.Id.String(), time.Now().UnixMilli()))
}

func (v *Vault) metadataPath(item Item) string {
	return filepath.Join(v.options.StoragePath, item.Id.String()+".json")
}

func (v *Vault) valuePath(item Item) string {
	return filepath.Join(v.options.StoragePath, item.Id.String()+".age")
}
