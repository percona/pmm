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

		fileCfg := registeredConfig(path, &config.Config{})
		require.NotNil(t, fileCfg)
		assert.Equal(t, testAgentID, fileCfg.ID)
		assert.Equal(t, testServerAddress, fileCfg.Server.Address)
	})

	t.Run("a missing file means no registration", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, registeredConfig(filepath.Join(t.TempDir(), "pmm-agent.yaml"), &config.Config{}))
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
}

func TestUnappliedSetupFlags(t *testing.T) {
	t.Parallel()

	assert.Empty(t, unappliedSetupFlags(&config.Setup{NodeName: "host", MetricsMode: "auto"}))
	assert.Equal(t, []string{"--region", "--custom-labels", "--expose-exporter"},
		unappliedSetupFlags(&config.Setup{Region: "eu", CustomLabels: "env=prod", ExposeExporter: true}))
}
