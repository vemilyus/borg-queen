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

package utils

import (
	"bufio"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"strings"

	"github.com/awnumar/memguard"
	"golang.org/x/term"
)

func Prompt(prompt string, currentValue string) (string, error) {
	finalPrompt := prompt
	if currentValue != "" {
		finalPrompt += " [" + currentValue + "]"
	}

	finalPrompt += ": "

	reader := bufio.NewReader(os.Stdin)

	for {
		print(finalPrompt)

		if !term.IsTerminal(int(os.Stdin.Fd())) {
			println("Not a TTY")
			os.Exit(2)
		}

		v, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		v = strings.TrimSpace(v)
		if v != "" {
			return v, nil
		} else if currentValue != "" {
			return currentValue, nil
		}
	}
}

func PromptSecure(prompt string) (*memguard.LockedBuffer, error) {
	fmt.Printf("%s: ", prompt)

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		println("Not a TTY")
		os.Exit(2)
	}

	s, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}

	println()

	return memguard.NewBufferFromBytes(s), nil
}

func PromptConfirm(prompt string, defaultYes bool) (bool, error) {
	preview := ""
	if defaultYes {
		preview += "Y"
	} else {
		preview += "y"
	}
	preview += "/"
	if defaultYes {
		preview += "n"
	} else {
		preview += "N"
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [%s]: ", prompt, preview)

		if !term.IsTerminal(int(os.Stdout.Fd())) {
			println("Not a TTY")
			os.Exit(2)
		}

		v, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}

		v = strings.TrimSpace(v)

		if v == "" {
			return defaultYes, nil
		}

		if strings.HasPrefix("yes", strings.ToLower(v)) {
			return true, nil
		} else if strings.HasPrefix("no", strings.ToLower(v)) {
			return false, nil
		}
	}
}

func AskForPassphrase() *memguard.LockedBuffer {
	value, err := PromptSecure("Enter passphrase")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to enter passphrase")
	}

	return value
}
