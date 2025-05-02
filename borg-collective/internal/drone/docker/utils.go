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

package docker

import (
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/robfig/cron/v3"
	"github.com/vemilyus/borg-collective/internal/drone/model"
)

func createProjectFromLabels(labels map[string]string) (*model.ContainerBackupProject, error) {
	var projectName string
	var schedule cron.Schedule
	var err error

	for key, value := range labels {
		if key == "io.v47.borgd.project_name" {
			projectName = value
		} else if key == "io.v47.borgd.when" {
			schedule, err = cron.ParseStandard(value)
			if err != nil {
				return nil, err
			}
		}
	}

	if projectName == "" {
		return nil, fmt.Errorf("project name not found in labels")
	}

	if schedule == nil {
		return nil, fmt.Errorf("schedule not found in labels")
	}

	return &model.ContainerBackupProject{
		ProjectName: projectName,
		Schedule:    schedule,
		JobId:       nil,
		Containers:  make(map[string]model.ContainerBackup),
	}, nil
}

func createContainerBackupFromInspect(inspect container.InspectResponse) (*model.ContainerBackup, error) {
	// TODO
	return nil, nil
}
