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
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/pkg/flags"
)

func TestUpgradeCmd(t *testing.T) {
	t.Parallel()

	t.Run("shall properly upgrade", func(t *testing.T) {
		t.Parallel()
		m := &MockFunctions{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		c := UpgradeCommand{dockerFn: m}

		oldContainerID := "containerID"
		m.Mock.On("HaveDockerAccess", mock.Anything).Return(true)
		m.Mock.On("ContainerInspect", mock.Anything, mock.Anything).Return(container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				ID:         oldContainerID,
				Name:       "container-name",
				HostConfig: &container.HostConfig{},
			},
			Config: &container.Config{
				Labels: map[string]string{"percona.pmm": "server"},
			},
		}, nil)
		m.Mock.On("PullImage", mock.Anything, c.DockerImage, mock.Anything).Return(&bytes.Buffer{}, nil)
		m.Mock.On("PullImage", mock.Anything, volumeCopyImage, mock.Anything).Return(&bytes.Buffer{}, nil)
		m.Mock.On("ContainerStop", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		m.Mock.On("RunContainer", mock.Anything, mock.MatchedBy(func(cfg *container.Config) bool {
			return cfg.Image == c.DockerImage
		}), mock.Anything, mock.Anything).Return("new-container-id", nil)
		m.Mock.On("ContainerUpdate", mock.Anything, oldContainerID, mock.Anything).Return(container.UpdateResponse{}, nil)

		_, err := c.RunCmdWithContext(context.Background(), &flags.GlobalFlags{})

		require.NoError(t, err)
	})

	t.Run("shall stop on PMM Server container not found", func(t *testing.T) {
		t.Parallel()
		m := &MockFunctions{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		c := UpgradeCommand{dockerFn: m}

		m.Mock.On("HaveDockerAccess", mock.Anything).Return(true)
		m.Mock.On("ContainerInspect", mock.Anything, mock.Anything).Return(container.InspectResponse{
			Config: &container.Config{},
		}, nil)

		_, err := c.RunCmdWithContext(context.Background(), &flags.GlobalFlags{})

		require.Error(t, err)
		require.True(t, errors.Is(err, ErrNotInstalledFromCli))
	})

	t.Run("shall backup all volumes", func(t *testing.T) {
		t.Parallel()
		m := &MockFunctions{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		c := UpgradeCommand{dockerFn: m}

		oldContainerID := "containerID"
		m.Mock.On("HaveDockerAccess", mock.Anything).Return(true)
		m.Mock.On("ContainerInspect", mock.Anything, mock.Anything).Return(container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				ID:         oldContainerID,
				Name:       "container-name",
				HostConfig: &container.HostConfig{},
			},
			Config: &container.Config{
				Labels: map[string]string{"percona.pmm": "server"},
			},
			Mounts: []types.MountPoint{
				{Type: mount.TypeVolume, Name: "vol1"},
				{Type: mount.TypeVolume, Name: "vol2"},
				{Type: mount.TypeBind, Name: "bind"},
			},
		}, nil)
		m.Mock.On("PullImage", mock.Anything, c.DockerImage, mock.Anything).Return(&bytes.Buffer{}, nil)
		m.Mock.On("PullImage", mock.Anything, volumeCopyImage, mock.Anything).Return(&bytes.Buffer{}, nil)
		m.Mock.On("CreateVolume", mock.Anything, mock.MatchedBy(func(v string) bool {
			return strings.HasPrefix(v, "vol1-")
		}), mock.Anything).Return(&volume.Volume{}, nil)
		m.Mock.On("CreateVolume", mock.Anything, mock.MatchedBy(func(v string) bool {
			return strings.HasPrefix(v, "vol2-")
		}), mock.Anything).Return(&volume.Volume{}, nil)
		m.Mock.On("ContainerStop", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		setWaitForContainerMock(m)
		setWaitForContainerMock(m)
		m.Mock.On("RunContainer", mock.Anything, mock.MatchedBy(func(cfg *container.Config) bool {
			return cfg.Image == volumeCopyImage
		}), mock.Anything, mock.Anything).Twice().Return("backup-container-id", nil)
		m.Mock.On("RunContainer", mock.Anything, mock.MatchedBy(func(cfg *container.Config) bool {
			return cfg.Image == c.DockerImage
		}), mock.Anything, mock.Anything).Return("new-container-id", nil)
		m.Mock.On("ContainerUpdate", mock.Anything, oldContainerID, mock.Anything).Return(container.UpdateResponse{}, nil)

		_, err := c.RunCmdWithContext(context.Background(), &flags.GlobalFlags{})

		require.NoError(t, err)
	})
}

func setWaitForContainerMock(m *MockFunctions) {
	ch := func() <-chan container.WaitResponse {
		c := make(chan container.WaitResponse)
		close(c)
		return c
	}()
	errC := func() <-chan error {
		c := make(chan error)
		close(c)
		return c
	}()
	m.Mock.On("ContainerWait", mock.Anything, mock.Anything, mock.Anything).Return(ch, errC)
}
