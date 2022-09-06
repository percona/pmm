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
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
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

	err := installDocker()
	if err != nil {
		return nil, err
	}

	dockerImage := "percona/pmm-server:2"

	ctx := context.Background()
	cli, err := docker.GetDockerClient(ctx)
	if err != nil {
		return nil, err
	}

	if !docker.HaveDockerAccess(ctx, cli) {
		logrus.Panic("Docker is either not running or this user has no access. Try running as root.")
	}

	volume, err := createVolume(ctx, cli)
	if err != nil {
		return nil, err
	}

	reader, err := cli.ImagePull(ctx, dockerImage, types.ImagePullOptions{})
	if err != nil {
		return nil, err
	}
	io.Copy(os.Stdout, reader)

	containerID, err := runContainer(ctx, cli, volume, dockerImage)
	if err != nil {
		return nil, err
	}

	logrus.Info("Waiting until PMM boots")
	w := docker.WaitForHealthyContainer(ctx, cli, containerID)
	healthy := <-w
	if healthy.Error != nil {
		return nil, healthy.Error
	}

	err = docker.ChangeServerPassword(ctx, cli, containerID, c.AdminPassword)
	if err != nil {
		return nil, err
	}

	return &installResult{
		adminPassword: c.AdminPassword,
	}, nil
}

func installDocker() error {
	isInstalled, err := docker.IsDockerInstalled()
	if err != nil {
		return err
	}

	if !isInstalled {
		logrus.Infoln("Installing Docker")
		err := docker.InstallDocker()
		if err != nil {
			return err
		}
	} else {
		logrus.Infoln("Docker is installed")
	}

	return nil
}

func createVolume(ctx context.Context, cli *client.Client) (*types.Volume, error) {
	volumeName := "pmm-data"
	v, err := cli.VolumeList(ctx, filters.NewArgs(filters.Arg("name", volumeName)))
	if err != nil {
		return nil, err
	}

	if len(v.Volumes) != 0 {
		logrus.Panicf("Docker volume with name %s already exists", volumeName)
	}

	volume, err := cli.VolumeCreate(ctx, volume.VolumeCreateBody{
		Name: volumeName,
		Labels: map[string]string{
			"percona.pmm.volume": "server",
		},
	})
	if err != nil {
		return nil, err
	}

	return &volume, nil
}

func runContainer(ctx context.Context, cli *client.Client, volume *types.Volume, dockerImage string) (string, error) {
	logrus.Info("Creating PMM Server")
	res, err := cli.ContainerCreate(ctx, &container.Config{
		Image: dockerImage,
		Labels: map[string]string{
			"percona.pmm": "server",
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"443/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "443"}},
			"80/tcp":  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "80"}},
		},
		Binds: []string{
			volume.Name + ":/srv:rw",
		},
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}, nil, nil, "")
	if err != nil {
		return "", err
	}

	logrus.Info("Starting PMM Server")
	if err := cli.ContainerStart(ctx, res.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	return res.ID, nil
}
