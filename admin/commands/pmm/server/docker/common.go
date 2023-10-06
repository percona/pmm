// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package docker

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
)

func startPMMServer(
	ctx context.Context,
	volume *volume.Volume,
	volumesFromContainerID string,
	dockerImage string,
	dockerFn Functions,
	portBindings nat.PortMap,
	containerName string,
) (string, error) {
	if volume == nil && volumesFromContainerID == "" {
		logrus.Panic("Both volume and volumesFromContainer are empty")
	}

	if volume != nil && volumesFromContainerID != "" {
		logrus.Panic("Both volume and volumesFromContainer are defined")
	}

	hostConfig := &container.HostConfig{
		PortBindings:  portBindings,
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}

	if volume != nil {
		hostConfig.Binds = []string{volume.Name + ":/srv:rw"}
	} else if volumesFromContainerID != "" {
		hostConfig.VolumesFrom = []string{volumesFromContainerID + ":rw"}
	}

	return dockerFn.RunContainer(ctx, &container.Config{
		Image: dockerImage,
		Labels: map[string]string{
			"percona.pmm": "server",
		},
	}, hostConfig, containerName)
}
