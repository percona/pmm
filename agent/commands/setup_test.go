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
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/config"
	aservice "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

const (
	testAgentID       = "5a2b8a4b-2b9d-4a5f-9a11-2b6a3f6f9a11"
	testServerAddress = "pmm.example.com:443"
	testNodeName      = "test-node"
	testNodeAddress   = "10.20.30.40"
)

// answer makes PMM Server answer the registration check with the given state.
func answer(state registrationState) registrationCheck {
	return func(*config.Config, *config.Config, *logrus.Entry) registrationState { return state }
}

// notAsked fails the test if PMM Server is asked about the registration at all.
func notAsked(t *testing.T) registrationCheck {
	t.Helper()

	return func(*config.Config, *config.Config, *logrus.Entry) registrationState {
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
		name string
		// nodeName is the name setup was given; nameGiven marks it as one the operator asked for rather
		// than the hostname it falls back to.
		nodeName  string
		nameGiven bool
		lookup    agentLookup
		want      registrationState
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
			name:      "PMM Server has the Agent on another Node",
			nodeName:  "another-node",
			nameGiven: true,
			lookup:    found(registeredNode),
			want:      registrationConflict,
		},
		{
			// `pmm-admin config <addr> generic` on a host whose name is not the Node's: the operator named
			// no Node, so there is nothing to resolve and the registration stands. Failing here refused
			// every re-run on a Node registered under a name which is not the hostname, and took down any
			// container which runs setup on start.
			name:     "the Node is registered under a name which is not this host's",
			nodeName: "the-hostname",
			lookup:   found(registeredNode),
			want:     registrationConfirmed,
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
				Setup: config.Setup{
					NodeName:      tc.nodeName,
					NodeNameGiven: tc.nameGiven,
					Address:       testNodeAddress,
				},
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

// The subtests configure the package level API clients, so they cannot run in parallel.
func TestWithGivenCredentials(t *testing.T) {
	registeredNode := serverNode{Name: testNodeName, Address: testNodeAddress}
	// PMM Server answers 401 for a token it no longer accepts, with a gRPC code which says no more than
	// that. Removing a Node deletes the service account the token of its Agent belongs to.
	refused := aservice.NewGetAgentDefault(http.StatusUnauthorized)

	// answers replies to consecutive lookups, and fails the test on a lookup it has no answer for.
	answers := func(t *testing.T, results ...func(string) (serverNode, error)) (agentLookup, *int) {
		t.Helper()

		calls := 0
		return func(agentID string) (serverNode, error) {
			calls++
			if calls > len(results) {
				t.Errorf("PMM Server was asked %d times, expected %d", calls, len(results))
				return serverNode{}, errors.New("asked too many times")
			}
			return results[calls-1](agentID)
		}, &calls
	}

	for _, tc := range []struct {
		name string
		// given holds the credentials setup was given, defaulting to those of a full command line
		given   *config.Config
		results []func(string) (serverNode, error)
		calls   int
		node    serverNode
		err     error
	}{
		{
			name:    "an answer PMM Server gave is the answer",
			results: []func(string) (serverNode, error){found(registeredNode)},
			calls:   1,
			node:    registeredNode,
		},
		{
			name:    "a Node PMM Server does not know is not asked about twice",
			results: []func(string) (serverNode, error){failed(errAgentNotFound)},
			calls:   1,
			err:     errAgentNotFound,
		},
		{
			name:    "a refused token is asked about again, and the Node is gone",
			results: []func(string) (serverNode, error){failed(refused), failed(errAgentNotFound)},
			calls:   2,
			err:     errAgentNotFound,
		},
		{
			// Registering again would remove that Node together with every Service on it.
			name:    "a refused token does not register a Node which is still there",
			results: []func(string) (serverNode, error){failed(refused), found(registeredNode)},
			calls:   2,
			err:     refused,
		},
		{
			name:    "a refusal is kept when the credentials given to setup answer no better",
			results: []func(string) (serverNode, error){failed(refused), failed(refused)},
			calls:   2,
			err:     refused,
		},
		{
			name:    "the credentials the Agent runs with are not tried twice",
			given:   &config.Config{Server: config.Server{Address: testServerAddress, Username: "service_token", Password: "glsa_token"}},
			results: []func(string) (serverNode, error){failed(refused)},
			calls:   1,
			err:     refused,
		},
		{
			name:    "there is nothing to ask again without a PMM Server address",
			given:   &config.Config{Server: config.Server{Username: "admin", Password: "admin"}},
			results: []func(string) (serverNode, error){failed(refused)},
			calls:   1,
			err:     refused,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			running := &config.Config{
				ID:     testAgentID,
				Server: config.Server{Address: testServerAddress, Username: "service_token", Password: "glsa_token"},
			}
			given := tc.given
			if given == nil {
				given = &config.Config{
					ID:     testAgentID,
					Server: config.Server{Address: testServerAddress, Username: "admin", Password: "admin"},
				}
			}

			lookup, calls := answers(t, tc.results...)
			node, err := withGivenCredentials(lookup, running, given, logrus.WithField("test", t.Name()))(testAgentID)
			assert.Equal(t, tc.calls, *calls)
			assert.Equal(t, tc.node, node)
			if tc.err == nil {
				require.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tc.err)
		})
	}
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

	t.Run("a file there is no key for says nothing about the registration", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "pmm-agent.yaml")
		require.NoError(t, os.WriteFile(path, []byte{0xd7, 0x2a, 0x00, 0xff, 0xfe}, 0o600))

		fileCfg, err := registeredConfig(path, &config.Config{})
		require.Error(t, err)
		assert.Nil(t, fileCfg)
	})
}

func TestConfigToStore(t *testing.T) {
	t.Parallel()

	l := logrus.WithField("test", t.Name())

	// The configuration the Agent runs with: a firewalled ports range and a /proc/mounts path which no
	// flag carries, plus the service token registering left behind.
	running := func() *config.Config {
		return &config.Config{
			ID:             testAgentID,
			ListenPort:     7777,
			Ports:          config.Ports{Min: 30000, Max: 30100},
			ProcMountsPath: "/host/proc/mounts",
			Paths:          config.Paths{PathsBase: "/opt/pmm"},
			Server: config.Server{
				Address:  testServerAddress,
				Username: "service_token",
				Password: "stored-token",
			},
		}
	}

	// What `pmm-admin config --log-level=debug <addr> generic` assembles: the flags and the defaults, with
	// the credentials keepRegistration settled onto it.
	assembled := func() *config.Config {
		return &config.Config{
			ID:       testAgentID,
			LogLevel: "debug",
			Server: config.Server{
				Address:  testServerAddress,
				Username: "service_token",
				Password: "stored-token",
			},
		}
	}

	t.Run("a configuration file setup loaded is written back", func(t *testing.T) {
		t.Parallel()

		cfg := assembled()
		stored, err := configToStore(cfg, running(), true, nil, l)
		require.NoError(t, err)
		assert.Same(t, cfg, stored)
	})

	t.Run("the first configuration file is written without being loaded", func(t *testing.T) {
		t.Parallel()

		cfg := assembled()
		stored, err := configToStore(cfg, nil, false, nil, l)
		require.NoError(t, err)
		assert.Same(t, cfg, stored)
	})

	t.Run("registering keeps what describes this host's installation", func(t *testing.T) {
		t.Parallel()

		// --force registers the Node again; it does not ask for the Agent to be reconfigured from
		// defaults. The ID and the token are the ones registering just returned.
		registered := assembled()
		registered.ID = "44444444-4444-4444-4444-444444444444"
		registered.Server.Password = "fresh-token"

		args := []string{"setup", "--force", "1.2.3.4", "generic"}
		stored, err := configToStore(registered, running(), false, args, l)
		require.NoError(t, err)

		assert.Equal(t, config.Ports{Min: 30000, Max: 30100}, stored.Ports)
		assert.Equal(t, "/host/proc/mounts", stored.ProcMountsPath)
		assert.Equal(t, "44444444-4444-4444-4444-444444444444", stored.ID)
		assert.Equal(t, "fresh-token", stored.Server.Password)
	})

	t.Run("keeping the registration merges the flags onto the file", func(t *testing.T) {
		t.Parallel()

		args := []string{"--log-level=debug", "--paths-base=/opt/other", "setup", "1.2.3.4", "generic"}
		stored, err := configToStore(assembled(), running(), false, args, l)
		require.NoError(t, err)

		// Given, so applied.
		assert.Equal(t, "debug", stored.LogLevel)
		assert.Equal(t, "/opt/other", stored.Paths.PathsBase)

		// Carried by the file alone: dropping these restarted the Agent onto blocked ports and lost the
		// path the filesystem collector reads.
		assert.Equal(t, config.Ports{Min: 30000, Max: 30100}, stored.Ports)
		assert.Equal(t, "/host/proc/mounts", stored.ProcMountsPath)
		assert.Equal(t, uint16(7777), stored.ListenPort)

		// The credentials the Agent runs with, not the ones given to setup.
		assert.Equal(t, "service_token", stored.Server.Username)
		assert.Equal(t, "stored-token", stored.Server.Password)
		assert.Equal(t, testAgentID, stored.ID)
	})

	t.Run("the file the Agent runs with is not modified in place", func(t *testing.T) {
		t.Parallel()

		fileCfg := running()
		_, err := configToStore(assembled(), fileCfg, false, []string{"--log-level=debug"}, l)
		require.NoError(t, err)
		assert.Empty(t, fileCfg.LogLevel)
	})

	t.Run("flags which do not parse are reported", func(t *testing.T) {
		t.Parallel()

		_, err := configToStore(assembled(), running(), false, []string{"--no-such-flag"}, l)
		require.Error(t, err)
	})
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

	t.Run("credentials which were not used are reported", func(t *testing.T) {
		t.Parallel()

		// A password which the Agent never gets to try, a mistyped one among them, must not pass for
		// accepted just because the stored token still works.
		cfg := &config.Config{ID: testAgentID, Server: config.Server{Username: "admin", Password: "WRONG"}}
		fileCfg := &config.Config{ID: testAgentID, Server: config.Server{Username: "service_token", Password: "glsa_token"}}
		assert.Equal(t, []string{"--server-username", "--server-password"}, unappliedCredentials(cfg, fileCfg))
	})

	t.Run("credentials equal to the stored ones are not reported", func(t *testing.T) {
		t.Parallel()

		stored := config.Server{Username: "service_token", Password: "glsa_token"}
		cfg := &config.Config{ID: testAgentID, Server: stored}
		assert.Empty(t, unappliedCredentials(cfg, &config.Config{ID: testAgentID, Server: stored}))
	})
}

func TestUnappliedSetupFlags(t *testing.T) {
	t.Parallel()

	assert.Empty(t, unappliedSetupFlags(&config.Setup{NodeName: "host", MetricsMode: "auto"}))
	assert.Equal(t, []string{"--region", "--custom-labels", "--expose-exporter"},
		unappliedSetupFlags(&config.Setup{Region: "eu", CustomLabels: "env=prod", ExposeExporter: true}))

	// These describe the Node on PMM Server just as the rest do, and are only applied by registering.
	assert.Equal(t, []string{"--container-id", "--container-name"},
		unappliedSetupFlags(&config.Setup{ContainerID: "abc123", ContainerName: "mysql"}))
}
