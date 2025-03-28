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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVault(t *testing.T) {
	// Define a temporary storage path for testing
	tempDir := t.TempDir()
	// Test successful creation of a new vault
	vault, err := NewVault(&Options{
		StoragePath: tempDir,
	})
	assert.NoError(t, err)
	assert.NotNil(t, vault)
	assert.True(t, vault.IsLocked()) // Should be locked initially

	// Check if the necessary directories were created
	identityPath := filepath.Join(tempDir, ".identity")
	backupPath := filepath.Join(tempDir, ".bak")

	_, err = os.Stat(identityPath)
	assert.True(t, os.IsNotExist(err), "Identity file should not exist yet")

	_, err = os.Stat(backupPath)
	assert.NoError(t, err, "Backup directory should be created")
}

func TestNewVault_ErrorCreatingStoragePath(t *testing.T) {
	// Attempt to create a vault with an invalid storage path
	vault, err := NewVault(&Options{
		StoragePath: "/invalid/path/to/storage", // Invalid path
	})
	assert.Error(t, err)
	assert.Nil(t, vault)
}

func TestUnlock(t *testing.T) {
	// Create a new vault
	vault, err := NewVault(&Options{
		StoragePath: t.TempDir(),
	})
	assert.NoError(t, err)

	// Define a valid passphrase
	validPassphrase := "correct_passphrase"

	// Test unlocking with the correct passphrase
	err = vault.Unlock(validPassphrase)
	assert.NoError(t, err)
	assert.False(t, vault.IsLocked())        // Vault should be unlocked
	assert.NotNil(t, vault.primaryRecipient) // Primary recipient should be set

	// Lock the vault again
	err = vault.Lock()
	assert.NoError(t, err)
	assert.True(t, vault.IsLocked()) // Vault should be locked

	// Test repeated unlocking of the locked vault
	err = vault.Unlock(validPassphrase)
	assert.NoError(t, err)                   // Should succeed
	assert.False(t, vault.IsLocked())        // Vault should be unlocked again
	assert.NotNil(t, vault.primaryRecipient) // Primary recipient should be set

	// Test repeated calls to unlocked vault
	err = vault.Unlock(validPassphrase)
	assert.NoError(t, err)                   // Should succeed
	assert.False(t, vault.IsLocked())        // Vault should still be unlocked
	assert.NotNil(t, vault.primaryRecipient) // Primary recipient should be set

	// Lock the vault again
	err = vault.Lock()
	assert.NoError(t, err)
	assert.True(t, vault.IsLocked()) // Vault should be locked

	// Test unlocking with a wrong passphrase
	invalidPassphrase := "wrong_passphrase"
	err = vault.Unlock(invalidPassphrase)
	assert.Error(t, err)                  // Should return an error for invalid passphrase
	assert.True(t, vault.IsLocked())      // Vault should still be locked
	assert.Nil(t, vault.primaryRecipient) // Primary recipient should be nil
}
