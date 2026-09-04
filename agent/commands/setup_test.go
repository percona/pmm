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

	// serverKnows answers as PMM Server which still has the Agent registered.
	serverKnows := func(*config.Config, *logrus.Entry) bool { return false }

	// serverForgot answers as PMM Server which was reinstalled and knows nothing about the Agent.
	serverForgot := func(*config.Config, *logrus.Entry) bool { return true }

	// serverNotAsked fails the test if PMM Server is asked at all.
	serverNotAsked := func(*config.Config, *logrus.Entry) bool {
		t.Errorf("PMM Server should not be asked about the registration")
		return false
	}

	t.Run("a new Agent registers", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{Server: config.Server{Address: serverAddress}}
		path := writeConfigFile(t, serverAddress)
		assert.False(t, skipRegistration(cfg, path, serverNotAsked, logrus.WithField("test", t.Name())))
	})

	t.Run("a registered Agent does not register again", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{ID: agentID, Server: config.Server{Address: serverAddress}}
		path := writeConfigFile(t, serverAddress)
		assert.True(t, skipRegistration(cfg, path, serverKnows, logrus.WithField("test", t.Name())))
	})

	t.Run("a registered Agent registers again when PMM Server does not know it", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{ID: agentID, Server: config.Server{Address: serverAddress}}
		path := writeConfigFile(t, serverAddress)
		assert.False(t, skipRegistration(cfg, path, serverForgot, logrus.WithField("test", t.Name())))
	})

	t.Run("a registered Agent registers with a different PMM Server", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{ID: agentID, Server: config.Server{Address: "new-pmm.example.com:443"}}
		path := writeConfigFile(t, serverAddress)
		assert.False(t, skipRegistration(cfg, path, serverNotAsked, logrus.WithField("test", t.Name())))
	})

	t.Run("a registered Agent registers again when forced", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{ID: agentID, Server: config.Server{Address: serverAddress}, Setup: config.Setup{Force: true}}
		path := writeConfigFile(t, serverAddress)
		assert.False(t, skipRegistration(cfg, path, serverNotAsked, logrus.WithField("test", t.Name())))
	})

	t.Run("registration is skipped on demand", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{ID: agentID, Setup: config.Setup{SkipRegistration: true, Force: true}}
		assert.True(t, skipRegistration(cfg, "not-exist.yaml", serverNotAsked, logrus.WithField("test", t.Name())))
	})

	t.Run("a registered Agent registers when the configuration file cannot be read", func(t *testing.T) {
		t.Parallel()

		path := writeConfigFile(t, serverAddress)
		require.NoError(t, os.Remove(path))

		cfg := &config.Config{ID: agentID, Server: config.Server{Address: serverAddress}}
		assert.False(t, skipRegistration(cfg, path, serverNotAsked, logrus.WithField("test", t.Name())))
	})
}
