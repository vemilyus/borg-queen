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
	"maps"
	"strings"
)

type inMemoryBackend struct {
	files map[string][]byte
}

func (i *inMemoryBackend) Init() error {
	i.files = map[string][]byte{}

	return nil
}

func (i *inMemoryBackend) ListFiles(path string) ([]string, error) {
	var finalList []string
	for key := range maps.Keys(i.files) {
		if path == "" && !strings.Contains(key, "/") {
			finalList = append(finalList, key)
		} else if path != "" && strings.HasSuffix(key, path) {
			finalList = append(finalList, key)
		}
	}

	if path != "" && len(finalList) == 0 {
		return nil, nil
	}

	return finalList, nil
}

func (i *inMemoryBackend) ReadFile(path string) ([]byte, error) {
	data, ok := i.files[path]
	if !ok {
		return nil, nil
	}

	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	return dataCopy, nil
}

func (i *inMemoryBackend) WriteFile(path string, bytes []byte) error {
	byteCopy := make([]byte, len(bytes))
	copy(byteCopy, bytes)

	i.files[path] = byteCopy

	return nil
}

func (i *inMemoryBackend) DeleteFile(path string) (bool, error) {
	_, ok := i.files[path]
	if !ok {
		return false, nil
	}

	delete(i.files, path)

	return true, nil
}
