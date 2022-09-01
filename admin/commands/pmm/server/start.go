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

// Server package holds the "pmm server" command
package server

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/docker"
)

type StartCommand struct{}

type startResult struct{}

// Result is a command run result.
func (res *startResult) Result() {}

// String stringifies command result.
func (res *startResult) String() string {
	return "ok"
}

func (c *StartCommand) RunCmd() (commands.Result, error) {
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
			continue
		}

		logrus.Infof("Starting %s in state %s", container.ID, container.State)
		if err := cli.ContainerStart(ctx, container.ID, types.ContainerStartOptions{}); err != nil {
			return nil, err
		}
	}

	return &startResult{}, nil
}
