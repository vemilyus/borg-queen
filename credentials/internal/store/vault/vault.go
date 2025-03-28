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
	"bufio"
	"bytes"
	"crypto/sha256"
	"errors"
	"filippo.io/age"
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
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
	options           *Options
	identityPath      string
	identityKey       *memguard.Enclave
	primaryRecipient  *age.X25519Recipient
	recoveryRecipient *age.X25519Recipient
	lock              sync.RWMutex
	items             map[uuid.UUID]Item
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
		options:           options,
		identityPath:      filepath.Join(options.StoragePath, ".identity"),
		identityKey:       nil,
		primaryRecipient:  nil,
		recoveryRecipient: recoveryRecipient,
		lock:              sync.RWMutex{},
		items:             map[uuid.UUID]Item{},
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

			log.Errorf("failed to read identity file (%s): %v", identityFile, err)
			return errors.New("failed to verify passphrase")
		}

		v.primaryRecipient = identity.Recipient()
	} else if os.IsNotExist(err) {
		identity, err := age.GenerateX25519Identity()
		if err != nil {
			log.Errorf("failed to generate primary identity: %v", err)
			return errors.New("failed to verify passphrase")
		}

		v.primaryRecipient = identity.Recipient()

		identityKey, _ := v.identityKey.Open()
		defer identityKey.Destroy()

		err = writeIdentity(identityFile, identityKey, identity)
		if err != nil {
			log.Errorf("failed to write identity (%s): %v", identityFile, err)
			return errors.New("failed to verify passphrase")
		}
	} else {
		log.Errorf("failed to stat identity file: %v", err)
		return errors.New("failed to verify passphrase")
	}

	items, err := readAllMetadataUnsafe(v.options.StoragePath)
	if err != nil {
		return fmt.Errorf("failed to load item metadata: %v", err)
	}

	v.items = items

	return nil
}

func (v *Vault) Lock() error {
	v.lock.Lock()
	defer v.lock.Unlock()

	if v.IsLocked() {
		return errors.New("vault is locked")
	}

	v.identityKey = nil
	v.primaryRecipient = nil

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

	recoveryRecipientPath := filepath.Join(v.options.StoragePath, ".recovery")
	if err := writeRecoveryRecipient(recoveryRecipientPath, recipient); err != nil {
		log.Errorf("failed to write recovery recipient: %v", err)

		if v.recoveryRecipient != nil {
			for i := 0; i < 3; i++ {
				time.Sleep(time.Second)

				err = writeRecoveryRecipient(recoveryRecipientPath, *v.recoveryRecipient)
				if err == nil {
					break
				}
			}

			if err != nil {
				log.Fatalf("failed to restore previous recovery recipient: %v", err)
			}
		}

		return errors.New("failed to set recovery recipient")
	}

	v.recoveryRecipient = &recipient

	items, err := readAllMetadataUnsafe(v.options.StoragePath)
	if err != nil {
		log.Errorf("failed to read item metadata: %v", err)
		return errors.New("failed to set recovery recipient")
	}

	for _, item := range items {
		func() {
			value, err := v.readItemValueUnsafe(item)
			if err != nil {
				log.Errorf("failed to read item value (%s): %v", item.Id, err)
				return
			}

			defer value.Destroy()

			err = v.writeItemValueUnsafe(item, value)
			if err != nil {
				log.Errorf("failed to write item value (%s): %v", item.Id, err)
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

	if err := writeItemMetadataUnsafe(v.metadataPath(item), item); err != nil {
		log.Errorf("failed to write item metadata: %v", err)
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
		log.Warnf("no such item: %s", id)
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

func (v *Vault) WriteItemValue(id uuid.UUID, value []byte) error {
	return v.SetItemValue(id, memguard.NewBufferFromBytes(value))
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

	checksum := sum(value.Bytes())
	item.Checksum = checksum
	item.ModifiedAt = time.Now()

	backupPath := v.backupPath(item)
	if err := copyFile(valuePath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup of previous value (%s): %v", item.Id, err)
	}

	err = os.WriteFile(valuePath, ageBytes, 0600)
	if err != nil {
		return fmt.Errorf("failed to write item value (%s): %v", item.Id, err)
	}

	err = writeItemMetadataUnsafe(v.metadataPath(item), item)
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
			log.Debugf("error deleting item metadata file (%s): %v", id.String(), err)
		} else {
			removed = true
		}
	}

	valuePath := v.valuePath(item)
	if _, err := os.Stat(valuePath); err == nil {
		err = os.Remove(valuePath)
		if err != nil {
			log.Debugf("error deleting item value file (%s): %v", id.String(), err)
		} else {
			removed = true
		}
	}

	if removed {
		log.Infof("removed files for item: %s", id.String())
	}

	return true
}

func (v *Vault) decryptFromRestUnsafe(data []byte) (*memguard.LockedBuffer, error) {
	identityKey, _ := v.identityKey.Open()
	defer identityKey.Destroy()

	identity, err := readIdentity(v.identityPath, identityKey)
	if err != nil {
		log.Fatalf("error reading identity: %v", err)
		return nil, errors.New("failed to decrypt data")
	}

	reader, err := age.Decrypt(bytes.NewReader(data), identity)
	if err != nil {
		log.Fatalf("error decrypting data: %v", err)
		return nil, errors.New("failed to decrypt data")
	}

	out := bytes.Buffer{}
	if _, err := io.Copy(&out, reader); err != nil {
		log.Fatalf("error decrypting data: %v", err)
		return nil, errors.New("failed to decrypt data")
	}

	result := out.Bytes()[:]
	wipeBuffer(out, len(result))

	return memguard.NewBufferFromBytes(result), nil
}

func (v *Vault) encryptForRestUnsafe(data *memguard.LockedBuffer) ([]byte, error) {
	var recipients []age.Recipient
	recipients = append(recipients, v.primaryRecipient)
	if v.recoveryRecipient != nil {
		recipients = append(recipients, v.recoveryRecipient)
	}

	out := bytes.Buffer{}
	outWriter := bufio.NewWriter(&out)

	wc, err := age.Encrypt(outWriter, recipients...)
	if err != nil {
		log.Fatalf("error encrypting data: %v", err)
		return nil, errors.New("failed to encrypt data")
	}

	_, err = wc.Write(data.Bytes())
	if err != nil {
		log.Fatalf("error writing data: %v", err)
		return nil, errors.New("failed to encrypt data")
	}

	err = wc.Close()
	if err != nil {
		log.Fatalf("error closing writer: %v", err)
		return nil, errors.New("failed to encrypt data")
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
