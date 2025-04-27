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

package borg

import (
	"crypto/rand"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"github.com/vemilyus/borg-collective/internal/drone"
)

var (
	supportedVersionMin   = semver.MustParse("1.2.5")
	supportedVersionUpper = semver.MustParse("2.0.0")
)

type Borg struct {
	config        *drone.Config
	scheduledJobs map[string]cron.EntryID
}

func New(config *drone.Config) (*Borg, error) {
	b := &Borg{
		config:        config,
		scheduledJobs: make(map[string]cron.EntryID),
	}

	version, err := b.runVersionLocal()
	if err != nil {
		return nil, fmt.Errorf("failed to check borg version: %v", err)
	}

	if version.LessThan(supportedVersionMin) || version.GreaterThanEqual(supportedVersionUpper) {
		return nil, fmt.Errorf("unsupported borg version (must be >= %v and < %v): %v", supportedVersionMin, supportedVersionUpper, version)
	}

	log.Info().Msgf("Borg version: %v", version)

	return b, nil
}

func (b *Borg) Compact() Action {
	return &closureAction{
		id: fmt.Sprintf("compact-%s", rand.Text()),
		action: func() error {
			return b.runCompact()
		},
	}
}

func (b *Borg) ScheduleBackups(backups []drone.BackupConfig, scheduler *cron.Cron) error {
	for _, backup := range backups {
		existing, ok := b.scheduledJobs[backup.Name]
		if ok {
			scheduler.Remove(existing)
			delete(b.scheduledJobs, backup.Name)
		}

		backupAction, err := b.buildBackupAction(backup)
		if err != nil {
			return err
		}

		action := NewComposedAction(backupAction)
		if len(backup.PreCommand) > 0 {
			action.Pre(NewExecAction(backup.PreCommand))
		}

		if len(backup.PostCommand) > 0 {
			action.Post(NewExecAction(backup.PostCommand))
		}

		if len(backup.FinallyCommand) > 0 {
			action.Finally(NewExecAction(backup.FinallyCommand))
		}

		action.SetId(backup.Name)

		entryId := scheduler.Schedule(backup.ScheduleParsed(), Wrap(action))
		b.scheduledJobs[backup.Name] = entryId
	}

	return nil
}

func (b *Borg) buildBackupAction(backup drone.BackupConfig) (Action, error) {
	// TODO
	return nil, nil
}
