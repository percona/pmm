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
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/pkg/flags"
)

func TestSummary(t *testing.T) {
	agentlocal.SetTransport(true, agentlocal.DefaultPMMAgentListenPort)

	f, err := os.CreateTemp(t.TempDir(), "pmm-admin-test-summary-*.zip")
	require.NoError(t, err)
	filename := f.Name()
	t.Logf("Using temp file: %s", filename)

	require.NoError(t, f.Close())

	t.Run("Summary default", func(t *testing.T) {
		cmd := &SummaryCommand{
			Filename: filename,
		}
		res, err := cmd.RunCmdWithContext(t.Context(), &flags.GlobalFlags{})
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
		res, err := cmd.RunCmdWithContext(t.Context(), &flags.GlobalFlags{})
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
		res, err := cmd.RunCmdWithContext(t.Context(), &flags.GlobalFlags{})
		require.NoError(t, err)
		expected := &summaryResult{
			Filename: filename,
		}

		// Check there is a pprof dir with data inside the zip file
		reader, err := zip.OpenReader(filename)
		require.NoError(t, err)
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

func TestAddRedactedConfig(t *testing.T) {
	t.Parallel()

	config := filepath.Join(t.TempDir(), "pmm-agent.yaml")
	require.NoError(t, os.WriteFile(config, []byte(`id: pmm-agent-id
server:
    address: 127.0.0.1:443
    username: service_token
    password: glsa_token
vault:
    token: ab
    secret-id: "s3cr3t"
    role-id: role
listen-port: 7777
`), 0o600))

	var buf bytes.Buffer
	zipW := zip.NewWriter(&buf)
	addRedactedConfig(zipW, "client/pmm-agent-config.yaml", config)
	require.NoError(t, zipW.Close())

	zipR, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, zipR.File, 1)
	f, err := zipR.File[0].Open()
	require.NoError(t, err)
	b, err := io.ReadAll(f)
	require.NoError(t, err)

	// Every credential is masked, whatever its length and quoting.
	assert.Equal(t, `id: pmm-agent-id
server:
    address: 127.0.0.1:443
    username: service_token
    password: '***'
vault:
    token: '***'
    secret-id: '***'
    role-id: role
listen-port: 7777
`, string(b))

	t.Run("VaultReference", func(t *testing.T) {
		t.Parallel()

		config := filepath.Join(t.TempDir(), "pmm-agent.yaml")
		require.NoError(t, os.WriteFile(config, []byte("server:\n    password: vault:secret/data/pmm/agents#token\n"), 0o600))

		var buf bytes.Buffer
		zipW := zip.NewWriter(&buf)
		addRedactedConfig(zipW, "client/pmm-agent-config.yaml", config)
		require.NoError(t, zipW.Close())

		zipR, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		require.NoError(t, err)
		f, err := zipR.File[0].Open()
		require.NoError(t, err)
		b, err := io.ReadAll(f)
		require.NoError(t, err)
		assert.Equal(t, "server:\n    password: vault:secret/data/pmm/agents#token\n", string(b))
	})
}
