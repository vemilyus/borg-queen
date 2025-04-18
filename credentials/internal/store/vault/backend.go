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
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Backend interface {
	Init() error

	ListFiles(string) ([]string, error)
	ReadFile(string) ([]byte, error)
	WriteFile(string, []byte) error
	DeleteFile(string) (bool, error)
}

type localStorageBackend struct {
	path string
}

func NewLocalStorageBackend(path string) Backend {
	return &localStorageBackend{
		path: path,
	}
}

func (b *localStorageBackend) Init() error {
	absPath, err := filepath.Abs(b.path)
	if err != nil {
		return err
	}

	b.path = absPath

	err = os.MkdirAll(b.path, 0700)
	if err != nil {
		return fmt.Errorf("error creating storage path: %s (%v)", b.path, err)
	}

	return nil
}

func (b *localStorageBackend) ListFiles(path string) ([]string, error) {
	listPath := b.cleanPath(path)

	listing, err := os.ReadDir(listPath)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("error reading path: %s (%v)", path, err)
	}

	var items []string

	for _, item := range listing {
		if !item.IsDir() {
			items = append(items, filepath.Join(path, item.Name()))
		}
	}

	return items, nil
}

func (b *localStorageBackend) ReadFile(path string) ([]byte, error) {
	readPath := b.cleanPath(path)

	data, err := os.ReadFile(readPath)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("error reading file: %s (%v)", path, err)
	}

	return data, nil
}

func (b *localStorageBackend) WriteFile(path string, data []byte) error {
	writePath := b.cleanPath(path)

	parentDir := filepath.Dir(writePath)
	err := os.MkdirAll(parentDir, 0700)
	if err != nil {
		return fmt.Errorf("error creating path: %s (%v)", path, err)
	}

	err = os.WriteFile(writePath, data, 0600)
	if err != nil {
		return fmt.Errorf("error writing file: %s (%v)", path, err)
	}

	return nil
}

func (b *localStorageBackend) DeleteFile(path string) (bool, error) {
	deletePath := b.cleanPath(path)

	stat, err := os.Stat(deletePath)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("error reading path: %s (%v)", path, err)
	}

	if stat.IsDir() {
		return false, fmt.Errorf("cannot delete a directory: %s", path)
	}

	err = os.RemoveAll(deletePath)
	if err != nil {
		return false, fmt.Errorf("error deleting file: %s (%v)", path, err)
	}

	return true, nil
}

func (b *localStorageBackend) cleanPath(path string) string {
	cleanPath := filepath.Clean(filepath.Join(b.path, path))
	if !strings.HasPrefix(strings.ToLower(cleanPath), strings.ToLower(b.path)) {
		panic("path tried to escape: " + path)
	}

	return cleanPath
}
