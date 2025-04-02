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

// Package docker stores common functions for working with Docker.
package docker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/pkg/common"
)

// ErrPasswordChangeFailed represents an error indicating that password change failed.
var ErrPasswordChangeFailed = errors.New("ErrPasswordChangeFailed")

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
	l := logrus.WithField("component", "docker")
	lw := l.Writer()

	cmd := exec.Command("sh", "-s")
	cmd.Stdin = script
	cmd.Stdout = lw
	cmd.Stderr = lw

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

// FindServerContainers finds all containers running PMM Server.
func (b *Base) FindServerContainers(ctx context.Context) ([]types.Container, error) {
	return b.Cli.ContainerList(ctx, container.ListOptions{ //nolint:exhaustruct
		All: true,
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "label",
			Value: "percona.pmm=server",
		}),
	})
}

// ChangeServerPassword changes password for PMM Server's admin user.
func (b *Base) ChangeServerPassword(ctx context.Context, containerID, newPassword string) error {
	logrus.Info("Changing password")

	exitCode, err := b.ContainerExecPrintOutput(ctx, containerID, []string{"change-admin-password", newPassword})
	if err != nil {
		return err
	}

	if exitCode != 0 {
		logrus.Errorf("Password change exit code: %d", exitCode)
		logrus.Error(`Password change failed. Use the default password "admin"`)
		return ErrPasswordChangeFailed
	}

	logrus.Info("Password changed")

	return nil
}

// WaitHealthyResponse holds information about container being healthy.
type WaitHealthyResponse struct {
	Healthy bool
	Error   error
}

// WaitForHealthyContainer waits until a containers is healthy.
func (b *Base) WaitForHealthyContainer(ctx context.Context, containerID string) <-chan WaitHealthyResponse {
	healthyChan := make(chan WaitHealthyResponse, 1)
	go func() {
		var res WaitHealthyResponse
		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			logrus.Info("Checking if container is healthy...")
			status, err := b.Cli.ContainerInspect(ctx, containerID)
			if err != nil {
				res.Error = err
				break
			}

			if status.State == nil || status.State.Health == nil || status.State.Health.Status == "healthy" {
				res.Healthy = true
				break
			}

			<-t.C
		}

		healthyChan <- res
	}()

	return healthyChan
}

// RunContainer creates and runs a container. It returns the container ID.
func (b *Base) RunContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, containerName string) (string, error) {
	res, err := b.Cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", err
	}

	if err := b.Cli.ContainerStart(ctx, res.ID, container.StartOptions{}); err != nil { //nolint:exhaustruct
		return "", err
	}

	return res.ID, nil
}

// ErrVolumeExists is returned when Docker volume already exists.
var ErrVolumeExists = fmt.Errorf("VolumeExists")

// CreateVolume first checks if the volume exists and creates it.
func (b *Base) CreateVolume(ctx context.Context, volumeName string, labels map[string]string) (*volume.Volume, error) {
	// We need to first manually check if the volume exists because
	// cli.VolumeCreate() does not complain if it already exists.
	v, err := b.Cli.VolumeList(ctx, volume.ListOptions{Filters: filters.NewArgs(filters.Arg("name", volumeName))})
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

	volumeLabels["percona.pmm"] = "server"

	volume, err := b.Cli.VolumeCreate(ctx, volume.CreateOptions{ //nolint:exhaustruct
		Name:   volumeName,
		Labels: volumeLabels,
	})
	if err != nil {
		return nil, err
	}

	return &volume, nil
}

// ContainerInspect returns information about a container.
func (b *Base) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return b.Cli.ContainerInspect(ctx, containerID)
}

// ContainerStop stops a container.
func (b *Base) ContainerStop(ctx context.Context, containerID string, timeout *int) error {
	return b.Cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: timeout})
}

// ContainerUpdate updates container configuration.
func (b *Base) ContainerUpdate(ctx context.Context, containerID string, updateConfig container.UpdateConfig) (container.ContainerUpdateOKBody, error) {
	return b.Cli.ContainerUpdate(ctx, containerID, updateConfig)
}

// ContainerWait waits until a container is in a specific state.
func (b *Base) ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	return b.Cli.ContainerWait(ctx, containerID, condition)
}

// ContainerExecPrintOutput runs a command in a container and prints output to stdout/stderr.
func (b *Base) ContainerExecPrintOutput(ctx context.Context, containerID string, cmd []string) (int, error) {
	cresp, err := b.Cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          cmd,
		AttachStderr: true,
		AttachStdout: true,
	})
	if err != nil {
		return 0, err
	}

	execID := cresp.ID

	// run it, with stdout/stderr attached
	aresp, err := b.Cli.ContainerExecAttach(ctx, execID, container.ExecStartOptions{})
	if err != nil {
		return 0, err
	}
	defer aresp.Close()

	// read the output
	outputDone := make(chan error)

	go func() {
		// StdCopy demultiplexes the stream into two buffers
		_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, aresp.Reader)
		outputDone <- err
	}()

	select {
	case err := <-outputDone:
		if err != nil {
			return 0, err
		}
		break

	case <-ctx.Done():
		return 0, ctx.Err()
	}

	// get the exit code
	iresp, err := b.Cli.ContainerExecInspect(ctx, execID)
	if err != nil {
		return 0, err
	}

	return iresp.ExitCode, nil
}
