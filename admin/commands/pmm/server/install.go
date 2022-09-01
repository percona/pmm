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

package server

import (
	"context"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/docker"
)

// InstallCommand is used by Kong for CLI flags and commands.
type InstallCommand struct {
	AdminPassword string `default:"admin123" help:"Password to be configured for the \"admin\" user during installation"`
}

type installResult struct {
	adminPassword string
}

// Result is a command run result.
func (r *installResult) Result() {}

// String stringifies command result.
func (r *installResult) String() string {
	return `
	
PMM Server is now available at http://localhost/

User: admin
Password: ` + r.adminPassword
}

// RunCmd runs install command.
func (c *InstallCommand) RunCmd() (commands.Result, error) {
	logrus.Info("Starting PMM Server installation")

	dockerImage := "percona/pmm-server:2"
	// dockerImage := "docker.io/library/alpine"

	ctx := context.Background()
	cli, err := docker.GetDockerClient(ctx)
	if err != nil {
		return nil, err
	}
	reader, err := cli.ImagePull(ctx, dockerImage, types.ImagePullOptions{})
	if err != nil {
		return nil, err
	}
	io.Copy(os.Stdout, reader)

	logrus.Info("Creating PMM Server")
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: dockerImage,
		// Cmd:   []string{"echo", "hello world"},
		Labels: map[string]string{
			"percona.pmm": "server",
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"443/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "443"}},
			"80/tcp":  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "80"}},
		},
	}, nil, nil, "")
	if err != nil {
		return nil, err
	}

	logrus.Info("Starting PMM Server")
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	logrus.Info("Waiting until PMM boots")
	w := docker.WaitForHealthyContainer(ctx, cli, resp.ID)
	healthy := <-w
	if healthy.Error != nil {
		return nil, healthy.Error
	}

	docker.ChangeServerPassword(ctx, cli, resp.ID, c.AdminPassword)

	return &installResult{
		adminPassword: c.AdminPassword,
	}, nil
}
