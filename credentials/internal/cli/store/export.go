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
	"github.com/vemilyus/borg-queen/credentials/internal/cli/grpcclient"
	"github.com/vemilyus/borg-queen/credentials/internal/cli/utils"
	"github.com/vemilyus/borg-queen/credentials/internal/proto"
	"os"
	"path"
	"unsafe"
)

type exportCmd struct {
	*flaggy.Subcommand
	pretty     bool
	outputFile string
}

func newExportCmd(parent *flaggy.Subcommand) *exportCmd {
	eCmd := &exportCmd{
		pretty: false,
	}

	cmd := flaggy.NewSubcommand("export")
	cmd.Description = "Exports the entire contents of the vault as JSON (DANGEROUS)"

	cmd.Bool(&eCmd.pretty, "p", "pretty", "Pretty print the output")
	cmd.String(&eCmd.outputFile, "o", "output", "Target file to store the output")

	parent.AttachSubcommand(cmd, 1)

	eCmd.Subcommand = cmd

	return eCmd
}

func (cmd *exportCmd) run(state *config.State) {
	if cmd.outputFile == "" {
		log.Fatal().Msg("Output file not specified")
	}

	log.Warn().Msg("Exporting the entire contents of the vault may potentially compromise\n  the security of your data.")
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

	finalData, err := grpcclient.Run(
		state.Config(),
		func(c grpcclient.GrpcClient) (*exportData, error) {
			rawItems, err := c.ListVaultItems(&proto.ItemSearch{Credentials: &proto.AdminCredentials{Passphrase: passphrase.String()}})
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to retrieve list of items")
			}

			log.Info().Msgf("Retrieved %d items", len(rawItems))

			var exportItems []*exportItem
			for _, rawItem := range rawItems {
				value, err := c.ReadVaultItem(&proto.ItemRequest{
					Credentials: &proto.ItemRequest_Admin{Admin: &proto.AdminCredentials{Passphrase: passphrase.String()}},
					ItemId:      rawItem.GetId(),
				})

				if err != nil {
					log.Fatal().Err(err).Msgf("Failed to read item %s for export", rawItem.Id)
				}

				strVal := string(value.Value)
				descriptionStr := rawItem.GetDescription()

				idVal, err := uuid.Parse(rawItem.GetId())
				if err != nil {
					log.Fatal().Err(err).Msgf("Failed to parse id %s", rawItem.GetId())
				}

				exportItems = append(exportItems, &exportItem{
					Description: &descriptionStr,
					Id:          idVal,
					Value:       &strVal,
				})
			}

			return &exportData{Items: exportItems}, nil
		},
	)

	defer finalData.wipe()

	var marshalled []byte
	if cmd.pretty {
		marshalled, err = json.MarshalIndent(finalData, "", "  ")
	} else {
		marshalled, err = json.Marshal(finalData)
	}

	defer memguard.WipeBytes(marshalled)

	err = os.MkdirAll(path.Dir(cmd.outputFile), 0700)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create output directory")
	}

	if stat, _ := os.Stat(cmd.outputFile); stat != nil {
		doOverwrite, err := utils.PromptConfirm("Output file already exists, overwrite?", true)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to confirm overwriting")
		}

		if !doOverwrite {
			log.Fatal().Msg("Output file already exists, user aborted")
			return
		}
	}

	file, err := os.OpenFile(cmd.outputFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create output file")
	}

	_, err = file.Write(marshalled)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to write output file")
	}

	log.Info().Msgf("Export complete: %s", cmd.outputFile)
}

type exportData struct {
	Items []*exportItem `json:"items"`
}

func (ed *exportData) wipe() {
	for i := range ed.Items {
		ed.Items[i].wipe()
	}
}

type exportItem struct {
	Description *string   `json:"description"`
	Id          uuid.UUID `json:"id"`
	Value       *string   `json:"value"`
}

func (ei *exportItem) wipe() {
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(ei.Description)))
	memguard.WipeBytes(*(*[]byte)(unsafe.Pointer(ei.Value)))
}
