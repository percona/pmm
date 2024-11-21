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
	"fmt"
	"io"
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/commands/pmm/common"
	"github.com/percona/pmm/admin/pkg/bubbles/progress"
	"github.com/percona/pmm/admin/pkg/docker"
	"github.com/percona/pmm/admin/pkg/flags"
)

const defaultGrafanaAdminPassword = "admin"

// InstallCommand is used by Kong for CLI flags and commands.
type InstallCommand struct {
	AdminPassword      string `default:"admin" help:"Password to be configured for the PMM server's \"admin\" user"`
	DockerImage        string `default:"percona/pmm-server:3" help:"Docker image to use to install PMM Server. Defaults to latest version"`
	HTTPSListenPort    uint16 `default:"443" help:"HTTPS port to listen on"`
	HTTPListenPort     uint16 `default:"80" help:"HTTP port to listen on"`
	ContainerName      string `default:"pmm-server" help:"Name of the PMM Server container"`
	VolumeName         string `default:"pmm-data" help:"Name of the volume used by PMM Server"`
	SkipDockerInstall  bool   `help:"Do not install Docker if it's not installed"`
	SkipChangePassword bool   `help:"Do not change password after PMM Server is installed"`

	dockerFn Functions
}

type installResult struct {
	URL      string `json:"url"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// Result is a command run result.
func (r *installResult) Result() {}

// String stringifies command result.
func (r *installResult) String() string {
	return `
	
PMM Server is now available at ` + r.URL + `

User: ` + r.User + `
Password: ` + r.Password
}

// ErrDockerNoAccess is returned when there is no access to Docker or Docker is not running.
var ErrDockerNoAccess = fmt.Errorf("DockerNoAccess")

// RunCmdWithContext runs install command.
func (c *InstallCommand) RunCmdWithContext(ctx context.Context, globals *flags.GlobalFlags) (commands.Result, error) { //nolint:unparam
	logrus.Info("Starting PMM Server installation in Docker")

	d, err := prepareDocker(ctx, c.dockerFn, prepareOpts{install: !c.SkipDockerInstall})
	if err != nil {
		return nil, err
	}
	c.dockerFn = d

	volume, err := c.dockerFn.CreateVolume(ctx, c.VolumeName, nil)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Downloading %q", c.DockerImage)
	res, err := c.pullImage(ctx, globals)
	if res != nil || err != nil {
		return res, err
	}

	containerID, err := c.runContainer(ctx, volume, c.DockerImage)
	if err != nil {
		return nil, err
	}

	logrus.Info("Waiting until PMM boots")
	healthy := <-c.dockerFn.WaitForHealthyContainer(ctx, containerID)
	if healthy.Error != nil {
		return nil, healthy.Error
	}

	finalPassword := c.AdminPassword
	if !c.SkipChangePassword && c.AdminPassword != defaultGrafanaAdminPassword {
		err = c.dockerFn.ChangeServerPassword(ctx, containerID, c.AdminPassword)
		if err != nil {
			if !errors.Is(err, docker.ErrPasswordChangeFailed) {
				return nil, err
			}

			finalPassword = defaultGrafanaAdminPassword
		}
	}

	return &installResult{
		URL:      "http://localhost",
		User:     "admin",
		Password: finalPassword,
	}, nil
}

// runContainer runs PMM Server and returns the containerID.
func (c *InstallCommand) runContainer(ctx context.Context, volume *volume.Volume, dockerImage string) (string, error) {
	logrus.Info("Starting PMM Server")

	ports := nat.PortMap{
		"8443/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: strconv.Itoa(int(c.HTTPSListenPort))}},
		"8080/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: strconv.Itoa(int(c.HTTPListenPort))}},
	}

	containerID, err := startPMMServer(ctx, volume, "", dockerImage, c.dockerFn, ports, c.ContainerName)
	if err != nil {
		return "", err
	}

	logrus.Debugf("Started PMM Server in container %q", containerID)

	return containerID, nil
}

// pullImage pulls a docker image and displays progress.
func (c *InstallCommand) pullImage(ctx context.Context, globals *flags.GlobalFlags) (commands.Result, error) {
	reader, err := c.dockerFn.PullImage(ctx, c.DockerImage, image.PullOptions{})
	if err != nil {
		return nil, err
	}

	if globals.JSON {
		_, err := io.Copy(os.Stdout, reader)
		if err != nil {
			logrus.Error(err)
		}
		return nil, nil
	}

	return c.startProgressProgram(reader)
}

func (c *InstallCommand) startProgressProgram(reader io.Reader) (commands.Result, error) {
	p := tea.NewProgram(progress.NewSize())
	doneC, errC := c.dockerFn.ParsePullImageProgress(reader, p)
	go func() {
		<-doneC
		p.Send(tea.Quit())
	}()

	model, err := p.Run()
	if err != nil {
		return nil, err
	}

	if m, ok := model.(progress.SizeModel); ok {
		if m.Quitting {
			return common.ShutdownResult{}, nil
		}
	}
	if err := <-errC; err != nil {
		return nil, err
	}

	return nil, nil
}
