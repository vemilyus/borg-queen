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
	"context"
	"encoding/json"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"github.com/vemilyus/borg-collective/internal/drone/borg"
	"github.com/vemilyus/borg-collective/internal/drone/model"
)

type Docker struct {
	dc        *client.Client
	b         *borg.Borg
	scheduler *cron.Cron
	projects  map[string]*model.ContainerBackupProject
}

func New(dc *client.Client, b *borg.Borg, scheduler *cron.Cron) *Docker {
	return &Docker{
		dc:        dc,
		b:         b,
		scheduler: scheduler,
		projects:  make(map[string]*model.ContainerBackupProject),
	}
}

func (d *Docker) Start() error {
	containerList, err := d.dc.ContainerList(
		context.Background(),
		container.ListOptions{
			All:     true,
			Filters: filters.NewArgs(filters.Arg("label", "io.v47.borgd.enabled=true")),
		},
	)

	if err != nil {
		return err
	}

	for _, containerSummary := range containerList {
		err = d.handleContainerUpdated(containerSummary.ID)
	}

	dcEvents, errChan := d.dc.Events(context.Background(), events.ListOptions{
		Since: "",
		Until: "",
		Filters: filters.NewArgs(
			filters.Arg("label", "io.v47.borgd.enabled"),
			filters.Arg("event", "create"),
			filters.Arg("event", "update"),
			filters.Arg("event", "destroy"),
			filters.Arg("event", "mount"),
			filters.Arg("event", "umount"),
		),
	})

	for event := range dcEvents {
		eventHandled := false

		if event.Type == events.ContainerEventType {
			if event.Action == "create" || event.Action == "update" {
				eventHandled = true
				err = d.handleContainerUpdated(event.Actor.ID)
			} else if event.Action == "destroy" {
				eventHandled = true
				d.handleContainerDestroyed(event.Actor.ID)
			}
		} else if event.Type == events.VolumeEventType {
			if event.Action == "mount" {
				eventHandled = true
				err = d.handleContainerUpdated(event.Actor.Attributes["container"])
			} else if event.Action == "umount" {
				eventHandled = true
				err = d.handleContainerUpdated(event.Actor.Attributes["container"])
			}
		}

		if !eventHandled {
			evtJson, _ := json.Marshal(event)
			log.Debug().RawJSON("event", evtJson).Msg("received unhandled event from docker daemon")
		} else if err != nil {
			evtJson, _ := json.Marshal(event)
			log.Warn().
				Err(err).
				RawJSON("event", evtJson).
				Msg("failed to handle event")
		}
	}

	return <-errChan
}

func (d *Docker) findProjectForContainer(id string) *model.ContainerBackupProject {
	var result *model.ContainerBackupProject
	for _, project := range d.projects {
		for _, bc := range project.Containers {
			if bc.ID == id {
				result = project
			}
		}
	}

	return result
}

func (d *Docker) handleContainerUpdated(id string) error {
	inspect, err := d.dc.ContainerInspect(context.Background(), id)
	if err != nil {
		return err
	}

	project := d.findProjectForContainer(id)
	if project == nil {
		project, err = createProjectFromLabels(inspect.Config.Labels)
		if err != nil {
			return err
		}

		d.projects[project.ProjectName] = project
	}

	containerBackup, err := createContainerBackupFromInspect(inspect)
	if err != nil {
		return err
	}

	project.Containers[inspect.ID] = *containerBackup

	if project.JobId != nil {
		d.scheduler.Remove(*project.JobId)
	}

	backupAction, err := d.createProjectBackupAction(*project)
	if err != nil {
		return err
	}

	jobId := d.scheduler.Schedule(project.Schedule, borg.Wrap(backupAction))
	project.JobId = &jobId

	return nil
}

func (d *Docker) handleContainerDestroyed(id string) {
	project := d.findProjectForContainer(id)
	if project == nil {
		return
	}

	delete(project.Containers, id)
	if len(project.Containers) == 0 {
		if project.JobId != nil {
			d.scheduler.Remove(*project.JobId)
		}

		delete(d.projects, project.ProjectName)
	}
}

func (d *Docker) createProjectBackupAction(project model.ContainerBackupProject) (borg.Action, error) {
	// TODO
	return nil, nil
}
