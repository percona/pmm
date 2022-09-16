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
	"bytes"
	"context"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/cli/flags"
	"github.com/percona/pmm/admin/pkg/docker"
)

func TestInstallDocker(t *testing.T) {
	t.Parallel()

	t.Run("shall not install Docker if installed", func(t *testing.T) {
		t.Parallel()
		m := &MockDockerFunctions{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		m.Mock.On("IsDockerInstalled", mock.Anything).Return(true, nil)

		c := InstallCommand{dockerFn: m}
		err := c.installDocker()

		require.NoError(t, err)
	})

	t.Run("shall install Docker if not installed", func(t *testing.T) {
		t.Parallel()
		m := &MockDockerFunctions{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		m.Mock.On("IsDockerInstalled", mock.Anything).Return(false, nil)
		m.Mock.On("InstallDocker").Return(nil)

		c := InstallCommand{dockerFn: m}
		err := c.installDocker()

		require.NoError(t, err)
	})

	t.Run("shall skip Docker check", func(t *testing.T) {
		t.Parallel()
		m := &MockDockerFunctions{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		c := InstallCommand{
			dockerFn:        m,
			SkipDockerCheck: true,
		}
		err := c.installDocker()

		require.NoError(t, err)
	})

	t.Run("shall skip Docker installation", func(t *testing.T) {
		t.Parallel()
		m := &MockDockerFunctions{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		m.Mock.On("IsDockerInstalled", mock.Anything).Return(false, nil)

		c := InstallCommand{
			dockerFn:          m,
			SkipDockerInstall: true,
		}
		err := c.installDocker()

		require.NoError(t, err)
	})
}

func TestRunContainer(t *testing.T) {
	t.Parallel()

	t.Run("shall run container", func(t *testing.T) {
		t.Parallel()
		m := &MockDockerFunctions{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		m.Mock.On(
			"RunContainer", mock.Anything, mock.Anything, mock.Anything, "my-container",
		).Return("container-id", nil)

		c := InstallCommand{
			dockerFn:      m,
			ContainerName: "my-container",
		}
		containerID, err := c.runContainer(context.Background(), &types.Volume{}, "docker-image")

		require.NoError(t, err)
		require.Equal(t, containerID, "container-id")
	})
}

func TestRunCmd(t *testing.T) {
	t.Parallel()

	t.Run("shall run command successfully", func(t *testing.T) {
		t.Parallel()

		m := &MockDockerFunctions{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		m.Mock.On("IsDockerInstalled", mock.Anything).Return(true, nil)
		m.Mock.On("HaveDockerAccess", mock.Anything).Return(true)
		m.Mock.On("ChangeServerPassword", mock.Anything, "container-id", "admin123").Return(nil)
		m.Mock.On(
			"RunContainer", mock.Anything, mock.Anything, mock.Anything, "container-name",
		).Return("container-id", nil)
		m.Mock.On("PullImage", mock.Anything, "docker-image", mock.Anything).Return(&bytes.Buffer{}, nil)
		m.Mock.On("CreateVolume", mock.Anything, "volume-name").Return(&types.Volume{}, nil)
		setWaitForHealthyContainerMock(m)

		c := InstallCommand{
			dockerFn:      m,
			AdminPassword: "admin123",
			VolumeName:    "volume-name",
			DockerImage:   "docker-image",
			ContainerName: "container-name",
		}

		_, err := c.RunCmdWithContext(context.Background(), &flags.GlobalFlags{JSON: true})

		require.NoError(t, err)
	})

	t.Run("shall return error without Docker access", func(t *testing.T) {
		t.Parallel()
		m := &MockDockerFunctions{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		m.Mock.On("IsDockerInstalled", mock.Anything).Return(true, nil)
		m.Mock.On("HaveDockerAccess", mock.Anything).Return(false)

		c := InstallCommand{dockerFn: m}

		_, err := c.RunCmdWithContext(context.Background(), &flags.GlobalFlags{JSON: true})

		require.Error(t, err)
	})

	t.Run("shall skip password change", func(t *testing.T) {
		t.Parallel()

		m := &MockDockerFunctions{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		m.Mock.On("IsDockerInstalled", mock.Anything).Return(true, nil)
		m.Mock.On("HaveDockerAccess", mock.Anything).Return(true)
		m.Mock.On(
			"RunContainer", mock.Anything, mock.Anything, mock.Anything, "container-name",
		).Return("container-id", nil)
		m.Mock.On("PullImage", mock.Anything, "docker-image", mock.Anything).Return(&bytes.Buffer{}, nil)
		m.Mock.On("CreateVolume", mock.Anything, "volume-name").Return(&types.Volume{}, nil)
		setWaitForHealthyContainerMock(m)

		c := InstallCommand{
			dockerFn:           m,
			AdminPassword:      "admin123",
			VolumeName:         "volume-name",
			DockerImage:        "docker-image",
			ContainerName:      "container-name",
			SkipChangePassword: true,
		}

		_, err := c.RunCmdWithContext(context.Background(), &flags.GlobalFlags{JSON: true})

		require.NoError(t, err)
	})
}

func TestInstallResult(t *testing.T) {
	t.Parallel()

	r := &installResult{}
	require.NotEmpty(t, r.String())
}

func setWaitForHealthyContainerMock(m *MockDockerFunctions) {
	ch := func() <-chan docker.WaitHealthyResponse {
		c := make(chan docker.WaitHealthyResponse)
		close(c)
		return c
	}()
	m.Mock.On("WaitForHealthyContainer", mock.Anything, "container-id").Return(ch)
}
