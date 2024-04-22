// Copyright (C) 2024 Percona LLC
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

// Package docker holds the "pmm server install docker" command.
package docker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInstallDocker(t *testing.T) {
	t.Parallel()

	t.Run("shall not install Docker if installed", func(t *testing.T) {
		t.Parallel()
		m := &MockFunctions{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		m.Mock.On("IsDockerInstalled", mock.Anything).Return(true, nil)

		err := installDocker(context.Background(), m)

		require.NoError(t, err)
	})

	t.Run("shall install Docker if not installed", func(t *testing.T) {
		t.Parallel()
		m := &MockFunctions{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		m.Mock.On("IsDockerInstalled", mock.Anything).Return(false, nil)
		m.Mock.On("InstallDocker", mock.Anything).Return(nil)

		err := installDocker(context.Background(), m)

		require.NoError(t, err)
	})

	t.Run("shall skip Docker installation", func(t *testing.T) {
		t.Parallel()
		m := &MockFunctions{}
		t.Cleanup(func() { m.AssertExpectations(t) })

		m.Mock.On("HaveDockerAccess", mock.Anything).Return(true)

		_, err := prepareDocker(context.Background(), m, prepareOpts{
			install: false,
		})

		require.NoError(t, err)
	})
}
