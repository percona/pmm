// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/config"
)

func TestSkipRegistration(t *testing.T) {
	t.Parallel()

	const (
		agentID       = "5a2b8a4b-2b9d-4a5f-9a11-2b6a3f6f9a11"
		serverAddress = "pmm.example.com:443"
	)

	// writeConfigFile stores a configuration file holding the Agent ID and the PMM Server address.
	writeConfigFile := func(t *testing.T, address string) string {
		t.Helper()

		path := filepath.Join(t.TempDir(), "pmm-agent.yaml")
		err := config.SaveToFile(path, &config.Config{ID: agentID, Server: config.Server{Address: address}}, t.Name())
		require.NoError(t, err)

		return path
	}

	t.Run("a new Agent registers", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{Server: config.Server{Address: serverAddress}}
		assert.False(t, skipRegistration(cfg, writeConfigFile(t, serverAddress), logrus.WithField("test", t.Name())))
	})

	t.Run("a registered Agent does not register again", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{ID: agentID, Server: config.Server{Address: serverAddress}}
		assert.True(t, skipRegistration(cfg, writeConfigFile(t, serverAddress), logrus.WithField("test", t.Name())))
	})

	t.Run("a registered Agent registers with a different PMM Server", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{ID: agentID, Server: config.Server{Address: "new-pmm.example.com:443"}}
		assert.False(t, skipRegistration(cfg, writeConfigFile(t, serverAddress), logrus.WithField("test", t.Name())))
	})

	t.Run("a registered Agent registers again when forced", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{ID: agentID, Server: config.Server{Address: serverAddress}, Setup: config.Setup{Force: true}}
		assert.False(t, skipRegistration(cfg, writeConfigFile(t, serverAddress), logrus.WithField("test", t.Name())))
	})

	t.Run("registration is skipped on demand", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{ID: agentID, Setup: config.Setup{SkipRegistration: true, Force: true}}
		assert.True(t, skipRegistration(cfg, "not-exist.yaml", logrus.WithField("test", t.Name())))
	})

	t.Run("a registered Agent registers when the configuration file cannot be read", func(t *testing.T) {
		t.Parallel()

		path := writeConfigFile(t, serverAddress)
		require.NoError(t, os.Remove(path))

		cfg := &config.Config{ID: agentID, Server: config.Server{Address: serverAddress}}
		assert.False(t, skipRegistration(cfg, path, logrus.WithField("test", t.Name())))
	})
}
