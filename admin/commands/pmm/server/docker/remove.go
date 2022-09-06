// Copyright 2019 Percona LLC
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

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/docker"
)

type RemoveCommand struct{}

type removeResult struct{}

// Result is a command run result.
func (res *removeResult) Result() {}

// String stringifies command result.
func (res *removeResult) String() string {
	return "ok"
}

func (c *RemoveCommand) RunCmd() (commands.Result, error) {
	ctx := context.Background()
	cli, err := docker.GetDockerClient(ctx)
	if err != nil {
		return nil, err
	}

	containers, err := docker.FindServerContainers(ctx, cli)
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		if container.State != "exited" {
			logrus.Infof("Stopping %s in state %s", container.ID, container.State)
			cli.ContainerStop(ctx, container.ID, nil)
		}

		logrus.Infof("Removing %s", container.ID)
		cli.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{})
	}

	return &removeResult{}, nil
}
