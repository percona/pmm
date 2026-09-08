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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/config"
)

const (
	testAgentID       = "5a2b8a4b-2b9d-4a5f-9a11-2b6a3f6f9a11"
	testServerAddress = "pmm.example.com:443"
	testNodeName      = "test-node"
	testNodeAddress   = "10.20.30.40"
)

// answer makes PMM Server answer the registration check with the given state.
func answer(state registrationState) registrationCheck {
	return func(*config.Config, *logrus.Entry) registrationState { return state }
}

// notAsked fails the test if PMM Server is asked about the registration at all.
func notAsked(t *testing.T) registrationCheck {
	t.Helper()

	return func(*config.Config, *logrus.Entry) registrationState {
		t.Errorf("PMM Server should not be asked about the registration")
		return registrationMissing
	}
}

// found makes PMM Server answer the Agent lookup with the given Node.
func found(node serverNode) agentLookup {
	return func(string) (serverNode, error) { return node, nil }
}

// failed makes the Agent lookup fail with the given error.
func failed(err error) agentLookup {
	return func(string) (serverNode, error) { return serverNode{}, err }
}

func TestCheckRegistration(t *testing.T) {
	t.Parallel()

	registeredNode := serverNode{Name: testNodeName, Address: testNodeAddress}

	for _, tc := range []struct {
		name     string
		nodeName string
		lookup   agentLookup
		want     registrationState
	}{
		{
			name:     "PMM Server knows the Agent on this Node",
			nodeName: testNodeName,
			lookup:   found(registeredNode),
			want:     registrationConfirmed,
		},
		{
			name:     "an address which no longer matches keeps the registration",
			nodeName: testNodeName,
			lookup:   found(serverNode{Name: testNodeName, Address: "10.20.30.41"}),
			want:     registrationConfirmed,
		},
		{
			name:   "the Node name is not known locally",
			lookup: found(registeredNode),
			want:   registrationConfirmed,
		},
		{
			name:     "PMM Server has the Agent on another Node",
			nodeName: "another-node",
			lookup:   found(registeredNode),
			want:     registrationConflict,
		},
		{
			name:     "PMM Server does not know the Agent",
			nodeName: testNodeName,
			lookup:   failed(errAgentNotFound),
			want:     registrationMissing,
		},
		{
			name:     "PMM Server does not accept the credentials the Agent runs with",
			nodeName: testNodeName,
			lookup:   failed(errCredentialsRejected),
			want:     registrationMissing,
		},
		{
			name:     "PMM Server cannot be asked",
			nodeName: testNodeName,
			lookup:   failed(errors.New("connection refused")),
			want:     registrationUnverified,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				ID:     testAgentID,
				Server: config.Server{Address: testServerAddress},
				Setup:  config.Setup{NodeName: tc.nodeName, Address: testNodeAddress},
			}
			assert.Equal(t, tc.want, checkRegistration(cfg, tc.lookup))
		})
	}
}

func TestRunningCredentials(t *testing.T) {
	t.Parallel()

	t.Run("the registration is checked with the credentials the Agent runs with", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{ID: testAgentID, Server: config.Server{Username: "admin", Password: "admin"}}
		fileCfg := &config.Config{ID: testAgentID, Server: config.Server{Username: "service_token", Password: "glsa_token"}}

		check := runningCredentials(cfg, fileCfg)
		assert.Equal(t, "service_token", check.Server.Username)
		assert.Equal(t, "glsa_token", check.Server.Password)
		// The credentials given to setup are still the ones to register with.
		assert.Equal(t, "admin", cfg.Server.Username)
		assert.Equal(t, "admin", cfg.Server.Password)
	})

	t.Run("the credentials given to setup are used when the file holds none", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{ID: testAgentID, Server: config.Server{Username: "admin", Password: "admin"}}
		assert.Same(t, cfg, runningCredentials(cfg, &config.Config{ID: testAgentID}))
	})
}

func TestRegistrationOf(t *testing.T) {
	t.Parallel()

	registered := &config.Config{ID: testAgentID, Server: config.Server{Address: testServerAddress}}

	for _, tc := range []struct {
		name    string
		cfg     *config.Config
		fileCfg *config.Config
		// check answers as PMM Server; nil means PMM Server must not be asked
		check registrationCheck
		want  registrationState
	}{
		{
			name:    "a new Agent registers",
			cfg:     &config.Config{Server: config.Server{Address: testServerAddress}},
			fileCfg: &config.Config{Server: config.Server{Address: testServerAddress}},
			want:    registrationMissing,
		},
		{
			name:    "a registered Agent does not register again",
			cfg:     registered,
			fileCfg: registered,
			check:   answer(registrationConfirmed),
			want:    registrationConfirmed,
		},
		{
			name:    "a registered Agent registers again when PMM Server does not know it",
			cfg:     registered,
			fileCfg: registered,
			check:   answer(registrationMissing),
			want:    registrationMissing,
		},
		{
			name:    "a registered Agent keeps its registration when PMM Server cannot be asked",
			cfg:     registered,
			fileCfg: registered,
			check:   answer(registrationUnverified),
			want:    registrationUnverified,
		},
		{
			name:    "the file may hold the PMM Server address without the default port",
			cfg:     registered,
			fileCfg: &config.Config{ID: testAgentID, Server: config.Server{Address: "PMM.example.com"}},
			check:   answer(registrationConfirmed),
			want:    registrationConfirmed,
		},
		{
			name:    "a registered Agent registers with a different PMM Server",
			cfg:     &config.Config{ID: testAgentID, Server: config.Server{Address: "new-pmm.example.com:443"}},
			fileCfg: registered,
			want:    registrationMissing,
		},
		{
			name:    "a registered Agent registers again when forced",
			cfg:     &config.Config{ID: testAgentID, Server: config.Server{Address: testServerAddress}, Setup: config.Setup{Force: true}},
			fileCfg: registered,
			want:    registrationMissing,
		},
		{
			name: "a registered Agent registers when there is no configuration file",
			cfg:  registered,
			want: registrationMissing,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			check := tc.check
			if check == nil {
				check = notAsked(t)
			}
			assert.Equal(t, tc.want, registrationOf(tc.cfg, tc.fileCfg, check, logrus.WithField("test", t.Name())))
		})
	}
}

func TestRegisteredConfig(t *testing.T) {
	t.Parallel()

	t.Run("the file the Agent runs with is loaded", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "pmm-agent.yaml")
		saved := &config.Config{ID: testAgentID, Server: config.Server{Address: testServerAddress}}
		require.NoError(t, config.SaveToFile(path, saved, t.Name()))

		fileCfg, err := registeredConfig(path, &config.Config{})
		require.NoError(t, err)
		require.NotNil(t, fileCfg)
		assert.Equal(t, testAgentID, fileCfg.ID)
		assert.Equal(t, testServerAddress, fileCfg.Server.Address)
	})

	t.Run("a missing file means no registration", func(t *testing.T) {
		t.Parallel()

		fileCfg, err := registeredConfig(filepath.Join(t.TempDir(), "pmm-agent.yaml"), &config.Config{})
		require.NoError(t, err)
		assert.Nil(t, fileCfg)
	})

	t.Run("a file which cannot be read is not a missing file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "pmm-agent.yaml")
		require.NoError(t, os.WriteFile(path, []byte("not YAML"), 0o600))

		fileCfg, err := registeredConfig(path, &config.Config{})
		require.Error(t, err)
		assert.Nil(t, fileCfg)
	})

	t.Run("a file with no key for it is told from a broken one", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "pmm-agent.yaml")
		require.NoError(t, os.WriteFile(path, []byte{0xd7, 0x2a, 0x00, 0xff, 0xfe}, 0o600))

		fileCfg, err := registeredConfig(path, &config.Config{})
		require.ErrorIs(t, err, config.ErrEncryptedConfigFile)
		assert.Nil(t, fileCfg)
	})
}

func TestStoresConfig(t *testing.T) {
	t.Parallel()

	fileCfg := &config.Config{ID: testAgentID}

	for _, tc := range []struct {
		name           string
		registered     bool
		loadedFromFile bool
		fileCfg        *config.Config
		stores         bool
	}{
		{
			name:       "registering replaces the configuration file",
			registered: true,
			fileCfg:    fileCfg,
			stores:     true,
		},
		{
			name:           "a configuration file setup loaded is written back",
			loadedFromFile: true,
			fileCfg:        fileCfg,
			stores:         true,
		},
		{
			name:   "the first configuration file is written without being loaded",
			stores: true,
		},
		{
			name:    "keeping the registration leaves a file setup did not load alone",
			fileCfg: fileCfg,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.stores, storesConfig(tc.registered, tc.loadedFromFile, tc.fileCfg))
		})
	}
}

func TestKeepRegistration(t *testing.T) {
	t.Parallel()

	t.Run("the stored service token outlives the credentials given to setup", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{ID: testAgentID, Server: config.Server{Username: "admin", Password: "admin"}}
		fileCfg := &config.Config{ID: testAgentID, Server: config.Server{Username: "service_token", Password: "glsa_token"}}
		keepRegistration(cfg, fileCfg)
		assert.Equal(t, "service_token", cfg.Server.Username)
		assert.Equal(t, "glsa_token", cfg.Server.Password)
	})

	t.Run("credentials given to setup are stored when the file holds none", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{ID: testAgentID, Server: config.Server{Username: "admin", Password: "admin"}}
		keepRegistration(cfg, &config.Config{ID: testAgentID})
		assert.Equal(t, "admin", cfg.Server.Username)
		assert.Equal(t, "admin", cfg.Server.Password)
	})
}

func TestUnappliedSetupFlags(t *testing.T) {
	t.Parallel()

	assert.Empty(t, unappliedSetupFlags(&config.Setup{NodeName: "host", MetricsMode: "auto"}))
	assert.Equal(t, []string{"--region", "--custom-labels", "--expose-exporter"},
		unappliedSetupFlags(&config.Setup{Region: "eu", CustomLabels: "env=prod", ExposeExporter: true}))
}
