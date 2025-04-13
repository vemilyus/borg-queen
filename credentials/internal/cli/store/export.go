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
	"encoding/json"
	"github.com/awnumar/memguard"
	"github.com/google/uuid"
	"github.com/integrii/flaggy"
	"github.com/rs/zerolog/log"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/config"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/httpclient"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/utils"
	"github.com/vemilyus/borg-queen/credentials/internal/model"
	"os"
	"unsafe"
)

type exportCmd struct {
	*flaggy.Subcommand
	pretty bool
}

func newExportCmd(parent *flaggy.Subcommand) *exportCmd {
	eCmd := &exportCmd{
		pretty: false,
	}

	cmd := flaggy.NewSubcommand("export")
	cmd.Description = "Exports the entire contents of the vault as JSON (DANGEROUS)"

	cmd.Bool(&eCmd.pretty, "p", "pretty", "Pretty print the output")

	parent.AttachSubcommand(cmd, 1)

	eCmd.Subcommand = cmd

	return eCmd
}

func (cmd *exportCmd) run(state *config.State) {
	log.Warn().Msg("Exporting the entire contents of the vault may potentially compromise\n the security of your data.")
	doExport, err := utils.PromptConfirm("Confirm exporting all vault contents", false)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to confirm")
	}

	if !doExport {
		log.Info().Msg("Not exporting all vault contents, user aborted")
		return
	}

	passphrase := utils.AskForPassphrase()
	defer passphrase.Destroy()

	doReconfirm, err := utils.PromptConfirm("Reconfirm exporting all vault contents", false)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to confirm")
	}

	if !doReconfirm {
		log.Info().Msg("Not exporting all vault contents, user aborted")
		return
	}

	httpClient := httpclient.New(state.Config())

	var response model.ListVaultItemsResponse
	err = httpClient.Post(model.PathGetItemList, model.ListVaultItemsRequest{
		PassphraseRequest:   model.PassphraseRequest{Passphrase: passphrase.String()},
		DescriptionContains: nil,
	}, &response)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to retrieve list of items")
	}

	log.Info().Msgf("Retrieved %d items", len(response.Items))

	var values []*exportItem
	var readVaultItemResponse model.ReadVaultItemResponse
	for _, item := range response.Items {
		err = httpClient.Post(model.PathGetItem, model.ReadVaultItemRequest{
			PassphraseRequest: model.PassphraseRequest{
				Passphrase: passphrase.String(),
			},
			ItemId: item.Id,
		}, &readVaultItemResponse)

		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to retrieve item %s for export", item.Id)
		}

		strVal := string(readVaultItemResponse.Value)
		readVaultItemResponse.Wipe()

		values = append(values, &exportItem{
			description: &item.Description,
			id:          item.Id,
			value:       &strVal,
		})
	}

	finalData := exportData{values}
	defer finalData.wipe()

	var marshalled []byte
	if cmd.pretty {
		marshalled, err = json.MarshalIndent(finalData, "", "  ")
	} else {
		marshalled, err = json.Marshal(finalData)
	}

	defer memguard.WipeBytes(marshalled)

	_, _ = os.Stdout.Write(marshalled)
	_, _ = os.Stdout.Write([]byte{'\n'})
}

type exportData struct {
	items []*exportItem
}

func (ed *exportData) wipe() {
	for i := range ed.items {
		ed.items[i].wipe()
	}
}

type exportItem struct {
	description *string
	id          uuid.UUID
	value       *string
}

func (ei *exportItem) wipe() {
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(ei.description)))
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(ei.value)))
}
