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
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/cli/flags"
	"github.com/percona/pmm/admin/commands"
)

// UpgradeCommand is used by Kong for CLI flags and commands.
type UpgradeCommand struct {
	DockerImage            string `default:"percona/pmm-server:2" help:"Docker image to use to upgrade PMM Server. Defaults to latest version"`
	ContainerID            string `default:"pmm-server" help:"Container ID of the PMM Server to upgrade"`
	NewContainerName       string `help:"Name of the new container for PMM Server. If this flag is set, --new-container-name-prefix is ignored. Must be different from the current container name"` //nolint:lll
	NewContainerNamePrefix string `default:"pmm-server" help:"Prefix for the name of the new container for PMM Server"`
	AssumeYes              bool   `name:"yes" short:"y" help:"Assume yes for all prompts"`
	dockerFn               Functions
}

const (
	dateSuffixFormat = "2006-01-02-15-04-05"
	volumeCopyImage  = "alpine:3"
)

var copyLabelsToVolumeBackup = []string{
	"org.label-schema.version",
	"org.opencontainers.image.version",
}

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
func (c *UpgradeCommand) RunCmdWithContext(ctx context.Context, globals *flags.GlobalFlags) (commands.Result, error) { //nolint:unparam,revive
	logrus.Info("Starting PMM Server upgrade via Docker")

	d, err := prepareDocker(ctx, c.dockerFn, prepareOpts{install: false})
	if err != nil {
		return nil, err
	}
	c.dockerFn = d

	currentContainer, err := c.dockerFn.ContainerInspect(ctx, c.ContainerID)
	if err != nil {
		return nil, err
	}

	if !c.isInstalledViaCli(currentContainer) {
		if !c.confirmToContinue(c.ContainerID) {
			return nil, fmt.Errorf("%w: the existing PMM Server was not installed via pmm cli", ErrNotInstalledFromCli)
		}
	}

	logrus.Infof("Downloading PMM Server %s", c.DockerImage)
	if err := c.pullImage(ctx, c.DockerImage); err != nil {
		return nil, err
	}

	logrus.Infof("Stopping PMM Server in container %q", currentContainer.Name)
	noTimeout := -1
	if err = c.dockerFn.ContainerStop(ctx, currentContainer.ID, &noTimeout); err != nil {
		return nil, err
	}

	if err = c.backupVolumes(ctx, &currentContainer); err != nil {
		return nil, err
	}

	if err = c.runPMMServer(ctx, currentContainer); err != nil {
		return nil, err
	}

	// Disable restart policy in the old container
	_, err = c.dockerFn.ContainerUpdate(ctx, currentContainer.ID, container.UpdateConfig{
		RestartPolicy: container.RestartPolicy{Name: "no"},
	})
	if err != nil {
		logrus.Info("We could not disable restart policy in the old container.")
		logrus.Infof(`We strongly recommend removing the old container manually with "docker rm %s"`, currentContainer.Name)
		logrus.Errorf("Error for reference: %#v", err)
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

func (c *UpgradeCommand) confirmToContinue(containerID string) bool {
	//nolint:forbidigo
	fmt.Printf(`
PMM Server in the container %[1]q was not installed via pmm cli.
We will attempt to upgrade the container and perform the following actions:

- Stop the container %[1]q
- Back up all volumes in %[1]q
- Mount all volumes from %[1]q in the new container
- Share the same network ports as in %[1]q

The container %[1]q will NOT be removed. You can remove it manually later, if needed.

`, containerID)

	if c.AssumeYes {
		return true
	}

	fmt.Print("Are you sure you want to continue? [y/N] ") //nolint:forbidigo

	s := bufio.NewScanner(os.Stdin)
	s.Scan()
	input := s.Text()

	return strings.ToLower(input) == "y"
}

func (c *UpgradeCommand) backupVolumes(ctx context.Context, container *types.ContainerJSON) error {
	logrus.Info("Starting backup of volumes")

	logrus.Infof("Downloading %q", volumeCopyImage)
	if err := c.pullImage(ctx, volumeCopyImage); err != nil {
		return err
	}

	now := time.Now()
	for _, m := range container.Mounts {
		if m.Type != mount.TypeVolume {
			continue
		}

		// Copy labels from original container to backup volume
		labels := make(map[string]string, 1+len(copyLabelsToVolumeBackup))
		for _, l := range copyLabelsToVolumeBackup {
			if _, ok := container.Config.Labels[l]; ok {
				labels[l] = container.Config.Labels[l]
			}
		}

		labels["percona.pmm.created"] = now.Format(dateSuffixFormat)

		backupName := fmt.Sprintf("%s-backup-%s", m.Name, now.Format(dateSuffixFormat))
		_, err := c.dockerFn.CreateVolume(ctx, backupName, labels)
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

func (c *UpgradeCommand) pullImage(ctx context.Context, imageName string) error {
	reader, err := c.dockerFn.PullImage(ctx, imageName, image.PullOptions{})
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

	logrus.Info("Backing up volume data")
	waitC, errC := c.dockerFn.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
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
		c.dockerFn, currentContainer.HostConfig.PortBindings, containerName)
	if err != nil {
		return err
	}

	logrus.Debugf("Started PMM Server in container %q", containerID)

	return nil
}
