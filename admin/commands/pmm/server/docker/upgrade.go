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
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/cli/flags"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/docker"
)

// UpgradeCommand is used by Kong for CLI flags and commands.
type UpgradeCommand struct {
	DockerImage            string `default:"percona/pmm-server:2" help:"Docker image to use to upgrade PMM Server. Defaults to latest version"`
	ContainerID            string `default:"pmm-server" help:"Container ID of the PMM Server to upgrade"`
	NewContainerName       string `help:"Name of the new container for PMM Server. If this flag is set, --new-container-name-prefix is ignored. Must be different from the current container name"`
	NewContainerNamePrefix string `default:"pmm-server" help:"Prefix for the name of the new container for PMM Server"`

	dockerFn Functions
}

const (
	dateSuffixFormat = "2006-01-02-15-04-05"
	volumeCopyImage  = "alpine:3"
)

var (
	// ErrContainerWait is returned on error response from container wait.
	ErrContainerWait = errors.New("ContainerWait")
	// ErrVolumeBackup is returned on error with volume backup.
	ErrVolumeBackup = errors.New("VolumeBackup")
	// ErrNotInstalledFromCli is returned when the current container was not installed via cli.
	ErrNotInstalledFromCli = errors.New("NotInstalledFromCli")
)

type upgradeResult struct{}

// Result is a command run result.
func (u *upgradeResult) Result() {}

// String stringifies command result.
func (u *upgradeResult) String() string {
	return "ok"
}

// RunCmdWithContext runs upgrade command.
func (c *UpgradeCommand) RunCmdWithContext(ctx context.Context, globals *flags.GlobalFlags) (commands.Result, error) { //nolint:unparam
	logrus.Info("Starting PMM Server upgrade via Docker")

	if err := c.prepareDocker(ctx); err != nil {
		return nil, err
	}

	currentContainer, err := c.dockerFn.GetDockerClient().ContainerInspect(ctx, c.ContainerID)
	if err != nil {
		return nil, err
	}

	if !c.isInstalledViaCli(currentContainer) {
		return nil, fmt.Errorf("%w: the existing PMM Server was not installed via pmm cli", ErrNotInstalledFromCli)
	}

	logrus.Infof("Stopping PMM Server in container %q", currentContainer.Name)
	noTimeout := -1 * time.Second
	if err = c.dockerFn.GetDockerClient().ContainerStop(ctx, currentContainer.ID, &noTimeout); err != nil {
		return nil, err
	}

	if err = c.backupVolumes(ctx, &currentContainer); err != nil {
		return nil, err
	}

	if err = c.runPMMServer(ctx, currentContainer); err != nil {
		return nil, err
	}

	// Disable restart policy in the old container
	_, err = c.dockerFn.GetDockerClient().ContainerUpdate(ctx, currentContainer.ID, container.UpdateConfig{
		RestartPolicy: container.RestartPolicy{Name: "no"},
	})
	if err != nil {
		logrus.Info("We could not disable restart policy in the old container.")
		logrus.Infof(`We strongly recommend removing the old container manually with "docker rm %s"`, currentContainer.Name)
		logrus.Info("Error for reference:")
		logrus.Error(err)
	}

	return &upgradeResult{}, nil
}

func (c *UpgradeCommand) isInstalledViaCli(container types.ContainerJSON) bool {
	for k, v := range container.Config.Labels {
		if k == "percona.pmm" && v == "server" {
			return true
		}
	}

	return false
}

func (c *UpgradeCommand) backupVolumes(ctx context.Context, container *types.ContainerJSON) error {
	logrus.Info("Starting backup of volumes")

	logrus.Infof("Downloading %q", volumeCopyImage)
	if err := c.pullBackupImage(ctx); err != nil {
		return err
	}

	now := time.Now()
	for _, m := range container.Mounts {
		if m.Type != mount.TypeVolume {
			continue
		}

		backupName := fmt.Sprintf("%s-backup-%s", m.Name, now.Format(dateSuffixFormat))
		_, err := c.dockerFn.CreateVolume(ctx, backupName)
		if err != nil {
			return err
		}

		logrus.Infof("Backing up volume %q to %q", m.Name, backupName)
		if err = c.backupVolumeViaContainer(ctx, m.Name, backupName); err != nil {
			return err
		}
	}

	return nil
}

func (c *UpgradeCommand) pullBackupImage(ctx context.Context) error {
	reader, err := c.dockerFn.PullImage(ctx, volumeCopyImage, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		logrus.Error(err)
	}

	return nil
}

func (c *UpgradeCommand) backupVolumeViaContainer(ctx context.Context, srcVolume, dstVolume string) error {
	logrus.Infof("Starting container to backup %q to %q", srcVolume, dstVolume)
	containerID, err := c.dockerFn.RunContainer(ctx, &container.Config{
		Image: volumeCopyImage,
		Cmd:   strslice.StrSlice{"cp", "-prT", "/srv-original", "/srv-backup"},
		Labels: map[string]string{
			"percona.pmm": "backup-container",
		},
	}, &container.HostConfig{
		Binds: []string{
			srcVolume + ":/srv-original:ro",
			dstVolume + ":/srv-backup:rw",
		},
		AutoRemove: true,
	}, "")
	if err != nil {
		return err
	}

	waitC, errC := c.dockerFn.GetDockerClient().ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case res := <-waitC:
		if res.Error != nil {
			return fmt.Errorf("%w: %s", ErrContainerWait, res.Error.Message)
		}

		if res.StatusCode != 0 {
			return fmt.Errorf("%w: backup exited with code %d", ErrVolumeBackup, res.StatusCode)
		}
	case err := <-errC:
		return err
	}

	return nil
}

func (c *UpgradeCommand) runPMMServer(ctx context.Context, currentContainer types.ContainerJSON) error {
	logrus.Info("Starting PMM Server")

	containerName := fmt.Sprintf("%s-%s", c.NewContainerNamePrefix, time.Now().Format(dateSuffixFormat))
	if c.NewContainerName != "" {
		containerName = c.NewContainerName
	}

	containerID, err := startPMMServer(
		ctx, nil, currentContainer.ID, c.DockerImage,
		c.dockerFn, currentContainer.HostConfig.PortBindings, containerName,
	)
	if err != nil {
		return err
	}

	logrus.Debugf("Started PMM Server in container %q", containerID)

	return nil
}

func (c *UpgradeCommand) prepareDocker(ctx context.Context) error {
	if c.dockerFn == nil {
		d, err := docker.New(nil)
		if err != nil {
			return err
		}

		c.dockerFn = d
	}

	if !c.dockerFn.HaveDockerAccess(ctx) {
		return fmt.Errorf("%w: docker is either not running or this user has no access to Docker. Try running as root", ErrDockerNoAccess)
	}

	return nil
}
