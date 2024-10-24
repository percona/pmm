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

package commands

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/pkg/flags"
)

func TestSummary(t *testing.T) {
	agentlocal.SetTransport(context.TODO(), true, agentlocal.DefaultPMMAgentListenPort)

	f, err := os.CreateTemp("", "pmm-admin-test-summary")
	require.NoError(t, err)
	filename := f.Name()
	t.Log(filename)

	defer os.Remove(filename) //nolint:errcheck
	assert.NoError(t, f.Close())

	t.Run("Summary default", func(t *testing.T) {
		cmd := &SummaryCommand{
			Filename: filename,
		}
		res, err := cmd.RunCmdWithContext(context.TODO(), &flags.GlobalFlags{})
		require.NoError(t, err)
		expected := &summaryResult{
			Filename: filename,
		}
		assert.Equal(t, expected, res)
	})

	t.Run("Summary skip server", func(t *testing.T) {
		cmd := &SummaryCommand{
			Filename:   filename,
			SkipServer: true,
		}
		res, err := cmd.RunCmdWithContext(context.TODO(), &flags.GlobalFlags{})
		require.NoError(t, err)
		expected := &summaryResult{
			Filename: filename,
		}
		assert.Equal(t, expected, res)
	})

	t.Run("Summary pprof", func(t *testing.T) {
		if os.Getenv("DEVCONTAINER") == "" {
			t.Skip("can be tested only inside devcontainer")
		}

		cmd := &SummaryCommand{
			Filename:   filename,
			SkipServer: true,
			Pprof:      true,
		}
		res, err := cmd.RunCmdWithContext(context.TODO(), &flags.GlobalFlags{})
		require.NoError(t, err)
		expected := &summaryResult{
			Filename: filename,
		}

		// Check there is a pprof dir with data inside the zip file
		reader, err := zip.OpenReader(filename)
		assert.NoError(t, err)
		assert.Equal(t, expected, res)

		hasPprofDir := false

		for _, file := range reader.File {
			if filepath.Dir(file.Name) == "pprof" {
				hasPprofDir = true
				break
			}
		}

		assert.True(t, hasPprofDir)
	})
}
