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
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/docker"
)

// InstallCommand is used by Kong for CLI flags and commands.
type InstallCommand struct {
	AdminPassword      string `default:"admin" help:"Password to be configured for the PMM server's \"admin\" user"`
	DockerImage        string `default:"percona/pmm-server:2" help:"Docker image to use to install PMM Server"`
	HTTPSListenPort    uint16 `default:"443" help:"HTTPS port to listen on"`
	HTTPListenPort     uint16 `default:"80" help:"HTTP port to listen on"`
	ContainerName      string `default:"pmm-server" help:"Name of the PMM Server container"`
	VolumeName         string `default:"pmm-data" help:"Name of the volume used by PMM Server"`
	SkipDockerInstall  bool   `help:"Do not install Docker if it's not installed"`
	SkipDockerCheck    bool   `help:"Do not check if Docker is installed."`
	SkipChangePassword bool   `help:"Do not change password after PMM Server is installed"`

	dockerFn DockerFunctions
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

	ctx := context.Background()

	if c.dockerFn == nil {
		d, err := docker.New(nil)
		if err != nil {
			return nil, err
		}

		c.dockerFn = d
	}

	if err := c.installDocker(); err != nil {
		return nil, err
	}

	if !c.dockerFn.HaveDockerAccess(ctx) {
		return nil, fmt.Errorf("Docker is either not running or this user has no access to Docker. Try running as root.")
	}

	volume, err := c.dockerFn.CreateVolume(ctx, c.VolumeName)
	if err != nil {
		return nil, err
	}

	reader, err := c.dockerFn.PullImage(ctx, c.DockerImage, types.ImagePullOptions{})
	if err != nil {
		return nil, err
	}
	io.Copy(os.Stdout, reader)

	containerID, err := c.runContainer(ctx, volume, c.DockerImage)
	if err != nil {
		return nil, err
	}

	logrus.Info("Waiting until PMM boots")
	healthy := <-c.dockerFn.WaitForHealthyContainer(ctx, containerID)
	if healthy.Error != nil {
		return nil, healthy.Error
	}

	if !c.SkipChangePassword {
		err = c.dockerFn.ChangeServerPassword(ctx, containerID, c.AdminPassword)
		if err != nil {
			return nil, err
		}
	}

	return &installResult{
		adminPassword: c.AdminPassword,
	}, nil
}

func (c *InstallCommand) installDocker() error {
	if c.SkipDockerCheck {
		logrus.Debugf("Docker check is disabled")
		return nil
	}

	isInstalled, err := c.dockerFn.IsDockerInstalled()
	if err != nil {
		return err
	}

	if isInstalled {
		return nil
	}

	if c.SkipDockerInstall {
		logrus.Infoln("Skipped Docker installation")
		return nil
	}

	logrus.Infoln("Installing Docker")
	err = c.dockerFn.InstallDocker()
	if err != nil {
		return err
	}

	return nil
}

// runContainer runs PMM Server and returns the containerID.
func (c *InstallCommand) runContainer(ctx context.Context, volume *types.Volume, dockerImage string) (string, error) {
	logrus.Info("Starting PMM Server")

	containerID, err := c.dockerFn.RunContainer(ctx, &container.Config{
		Image: dockerImage,
		Labels: map[string]string{
			"percona.pmm": "server",
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"443/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: strconv.Itoa(int(c.HTTPSListenPort))}},
			"80/tcp":  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: strconv.Itoa(int(c.HTTPListenPort))}},
		},
		Binds: []string{
			volume.Name + ":/srv:rw",
		},
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}, c.ContainerName)
	if err != nil {
		return "", err
	}

	logrus.Debugf("Started PMM Server in container %q", containerID)

	return containerID, nil
}
