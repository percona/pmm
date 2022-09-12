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
	"os/exec"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIsDockerInstalled(t *testing.T) {
	t.Parallel()

	t.Run("shall find Docker executable", func(t *testing.T) {
		ef := &MockExecFunctions{}
		t.Cleanup(func() { ef.AssertExpectations(t) })

		ef.Mock.On("LookPath", "docker").Return("docker", nil)

		d := &Base{}
		found, err := d.IsDockerInstalled(ef)

		require.NoError(t, err)
		require.True(t, found)
	})

	t.Run("shall return false on error", func(t *testing.T) {
		ef := &MockExecFunctions{}
		t.Cleanup(func() { ef.AssertExpectations(t) })

		ef.Mock.On("LookPath", "docker").Return("", fmt.Errorf("err"))

		d := &Base{}
		found, err := d.IsDockerInstalled(ef)

		require.Error(t, err)
		require.False(t, found)
	})

	t.Run("shall return false on not found without error", func(t *testing.T) {
		ef := &MockExecFunctions{}
		t.Cleanup(func() { ef.AssertExpectations(t) })

		ef.Mock.On("LookPath", "docker").Return("", &exec.Error{Err: exec.ErrNotFound})

		d := &Base{}
		found, err := d.IsDockerInstalled(ef)

		require.NoError(t, err)
		require.False(t, found)
	})
}

func TestHaveDockerAccess(t *testing.T) {
	t.Parallel()

	t.Run("shall have access", func(t *testing.T) {
		m := &MockDockerClient{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		m.Mock.On("Info", mock.Anything).Return(types.Info{}, nil)

		d := &Base{Cli: m}

		require.True(t, d.HaveDockerAccess(context.Background()))
	})

	t.Run("shall not have access", func(t *testing.T) {
		m := &MockDockerClient{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		m.Mock.On("Info", mock.Anything).Return(types.Info{}, fmt.Errorf("No access"))

		d := &Base{Cli: m}

		require.False(t, d.HaveDockerAccess(context.Background()))
	})
}

func TestGetDockerClient(t *testing.T) {
	t.Parallel()

	m := &MockDockerClient{}
	t.Cleanup(func() { m.AssertExpectations(t) })

	d := &Base{Cli: m}
	cli := d.GetDockerClient()

	require.Equal(t, m, cli)
}

func TestFindServerContainers(t *testing.T) {
	t.Parallel()

	m := &MockDockerClient{}
	t.Cleanup(func() { m.AssertExpectations(t) })

	m.Mock.On("ContainerList", mock.Anything, mock.Anything).Return([]types.Container{{}, {}}, nil)
	d := &Base{Cli: m}
	containers, _ := d.FindServerContainers(context.Background())

	require.Equal(t, len(containers), 2)
}
