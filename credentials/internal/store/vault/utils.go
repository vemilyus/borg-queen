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
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"filippo.io/age"
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"
)

func loadRecoveryRecipient(path string) (*age.X25519Recipient, error) {
	recBytes, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return age.ParseX25519Recipient(string(recBytes))
}

func writeRecoveryRecipient(path string, recipient age.X25519Recipient) error {
	recBytes := []byte(recipient.String())

	return os.WriteFile(path, recBytes, 0700)
}

func readIdentity(identityFile string, identityKey *memguard.LockedBuffer) (*age.X25519Identity, error) {
	cryptBytes, err := os.ReadFile(identityFile)
	if err != nil {
		return nil, err
	}

	c, err := aes.NewCipher(identityKey.Bytes())
	identityKey.Destroy()

	if err != nil {
		panic(err.Error())
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		panic(err.Error())
	}

	nonce := cryptBytes[:12]
	cryptBytes = cryptBytes[12:]

	rawIdentity, err := gcm.Open(nil, nonce, cryptBytes, nil)
	defer memguard.WipeBytes(rawIdentity)

	if err != nil {
		return nil, err
	}

	return age.ParseX25519Identity(*(*string)(unsafe.Pointer(&rawIdentity)))
}

func deriveMetadataHmacSecret(identity age.X25519Identity) *memguard.Enclave {
	identityString := identity.String()
	identityBytes := []byte(identityString)
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(&identityString)))
	rawHmacSecret := sha256.Sum256(identityBytes)

	return memguard.NewEnclave(rawHmacSecret[:])
}

func writeIdentity(identityFile string, identityKey *memguard.LockedBuffer, identity *age.X25519Identity) error {
	identityString := identity.String()
	identityBytes := *(*[]byte)(unsafe.Pointer(&identityString))
	defer memguard.WipeBytes(identityBytes)

	c, err := aes.NewCipher(identityKey.Bytes())
	if err != nil {
		return err
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		panic(err.Error())
	}

	cryptBytes := gcm.Seal(nil, nonce, identityBytes, nil)

	result := make([]byte, 0)
	result = append(result, nonce...)
	result = append(result, cryptBytes...)

	return os.WriteFile(identityFile, result, 0600)
}

func readAllMetadataUnsafe(storagePath string, hmacSecret *memguard.LockedBuffer) (map[uuid.UUID]Item, error) {
	listing, err := os.ReadDir(storagePath)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %w", err)
	}

	items := make(map[uuid.UUID]Item)

	for _, entry := range listing {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			metadataFile := filepath.Join(storagePath, entry.Name())
			metadata, err := readItemMetadataUnsafe(metadataFile, hmacSecret)
			if err != nil {
				log.Warn().Err(err).Str("source", metadataFile).Msg("error reading item metadata")
				continue
			}

			items[metadata.Id] = *metadata
		}
	}

	return items, nil
}

func readItemMetadataUnsafe(metadataPath string, hmacSecret *memguard.LockedBuffer) (*Item, error) {
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, err
	}

	h := hmac.New(sha256.New, hmacSecret.Bytes())
	h.Write(metadataBytes[:len(metadataBytes)-32])
	checkHmac := h.Sum(nil)
	if !bytes.Equal(checkHmac, metadataBytes[len(metadataBytes)-32:]) {
		return nil, errors.New("invalid metadata: checksum mismatch")
	}

	var metadata Item
	err = json.Unmarshal(metadataBytes[:len(metadataBytes)-32], &metadata)
	if err != nil {
		return nil, err
	}

	if filepath.Base(metadataPath) != metadata.Id.String()+".json" {
		return nil, errors.New("metadata path doesn't match item id: " + metadata.Id.String())
	}

	return &metadata, nil
}

func writeItemMetadataUnsafe(metadataPath string, item Item, hmacSecret *memguard.LockedBuffer) error {
	metadataBytes, err := json.Marshal(item)
	if err != nil {
		return err
	}

	h := hmac.New(sha256.New, hmacSecret.Bytes())
	h.Write(metadataBytes)

	result := make([]byte, 32+len(metadataBytes))
	copy(result, metadataBytes)
	copy(result[len(metadataBytes):], h.Sum(nil))

	return os.WriteFile(metadataPath, result, 0600)
}

func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}

	defer func() { _ = srcFile.Close() }()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer func() { _ = destFile.Close() }()

	_, err = io.Copy(destFile, srcFile)
	return err
}

func sum(data []byte) string {
	raw := sha256.Sum256(data)
	return hex.EncodeToString(raw[:])
}

func wipeSum(sum [32]byte) {
	for i := range sum {
		sum[i] = 0
	}

	runtime.KeepAlive(sum)
}

func wipeBuffer(buf *bytes.Buffer, length int) {
	// NOTE: Yes it may miss some data if the buffer was forced to allocate a bigger byte slice,
	//       but any left-over secret value in memory will only be a partial value, so it's
	//       not as bad as it seems.

	buf.Truncate(0)
	for range length {
		buf.WriteByte(0)
	}

	runtime.KeepAlive(buf)
}
