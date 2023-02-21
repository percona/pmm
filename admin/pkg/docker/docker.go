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

// Package docker stores common functions for working with Docker
package docker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/pkg/common"
)

// Base contains methods to interact with Docker.
type Base struct {
	Cli *client.Client
}

// New creates new instance of Base struct.
func New(cli *client.Client) (*Base, error) {
	if cli != nil {
		return &Base{Cli: cli}, nil
	}

	c, err := NewDockerClient()
	if err != nil {
		return nil, err
	}

	return &Base{Cli: c}, nil
}

// NewDockerClient returns a configured Docker client.
func NewDockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.WithVersion("1.41"))
}

// IsDockerInstalled checks if Docker is installed locally.
func (b *Base) IsDockerInstalled() (bool, error) {
	path, err := common.LookupCommand("docker")
	if err != nil {
		return false, err
	}

	if path == "" {
		return false, nil
	}

	logrus.Debugf("Found docker in %s", path)

	return true, nil
}

// HaveDockerAccess checks if the current user has access to Docker.
func (b *Base) HaveDockerAccess(ctx context.Context) bool {
	if _, err := b.Cli.Info(ctx); err != nil {
		logrus.Error(errors.WithStack(err))
		return false
	}

	return true
}

// ErrInvalidStatusCode is returned when HTTP status is not 200.
var ErrInvalidStatusCode = fmt.Errorf("InvalidStatusCode")

func (b *Base) downloadDockerInstallScript(ctx context.Context) (io.ReadCloser, error) {
	logrus.Debug("Downloading Docker installation script")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://get.docker.com/", nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: received HTTP %d when downloading Docker install script", ErrInvalidStatusCode, res.StatusCode)
	}

	return res.Body, nil
}

// InstallDocker installs Docker locally.
func (b *Base) InstallDocker(ctx context.Context) error {
	script, err := b.downloadDockerInstallScript(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err := script.Close(); err != nil {
			logrus.Error(err)
		}
	}()

	logrus.Debug("Running Docker installation script")
	cmd := exec.Command("sh", "-s")
	cmd.Stdin = script
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	logrus.Debug("Finished Docker installation")

	return nil
}

// GetDockerClient returns instance of Docker client.
func (b *Base) GetDockerClient() *client.Client {
	return b.Cli
}

// ChangeServerPassword changes password for PMM Server's admin user.
func (b *Base) ChangeServerPassword(ctx context.Context, containerID, newPassword string) error {
	logrus.Info("Changing password")

	exec, err := b.Cli.ContainerExecCreate(ctx, containerID, types.ExecConfig{ //nolint:exhaustruct
		Cmd:          []string{"change-admin-password", newPassword},
		Tty:          true,
		AttachStderr: true,
		AttachStdout: true,
	})
	if err != nil {
		return err
	}

	if err := b.Cli.ContainerExecStart(ctx, exec.ID, types.ExecStartCheck{}); err != nil { //nolint:exhaustruct
		return err
	}

	logrus.Info("Password changed")

	return nil
}

// ErrVolumeExists is returned when Docker volume already exists.
var ErrVolumeExists = fmt.Errorf("VolumeExists")

// CreateVolume first checks if the volume exists and creates it.
func (b *Base) CreateVolume(ctx context.Context, volumeName string, labels map[string]string) (*types.Volume, error) {
	// We need to first manually check if the volume exists because
	// cli.VolumeCreate() does not complain if it already exists.
	v, err := b.Cli.VolumeList(ctx, filters.NewArgs(filters.Arg("name", volumeName)))
	if err != nil {
		return nil, err
	}

	for _, vol := range v.Volumes {
		if vol.Name == volumeName {
			return nil, fmt.Errorf("%w: docker volume with name %q already exists", ErrVolumeExists, volumeName)
		}
	}

	volumeLabels := make(map[string]string, 1+len(labels))
	for k, v := range labels {
		volumeLabels[k] = v
	}

	volumeLabels["percona.pmm.source"] = "cli"

	volume, err := b.Cli.VolumeCreate(ctx, volume.VolumeCreateBody{ //nolint:exhaustruct
		Name:   volumeName,
		Labels: volumeLabels,
	})
	if err != nil {
		return nil, err
	}

	return &volume, nil
}
