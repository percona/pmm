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
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/pkg/flags"
)

func TestConfigCommandArgs(t *testing.T) {
	cmd := &ConfigCommand{
		NodeAddress: "1.2.3.4",
		NodeType:    "generic",
		NodeName:    "node1",
	}

	t.Run("SwitchToTLS1", func(t *testing.T) {
		u, err := url.Parse("http://127.0.0.1:80")
		require.NoError(t, err)
		args, switchedToTLS := cmd.args(&flags.GlobalFlags{ServerURL: u})
		expected := []string{
			"--server-address=127.0.0.1:443",
			"--server-insecure-tls",
			"setup", "1.2.3.4", "generic", "node1",
		}
		assert.Equal(t, expected, args)
		assert.True(t, switchedToTLS)
	})

	t.Run("SwitchToTLS2", func(t *testing.T) {
		cmd := &ConfigCommand{
			NodeAddress: "1.2.3.4",
			NodeType:    "generic",
			NodeName:    "node1",
		}
		u, err := url.Parse("http://admin:admin@127.0.0.1")
		require.NoError(t, err)
		args, switchedToTLS := cmd.args(&flags.GlobalFlags{ServerURL: u})
		expected := []string{
			"--server-address=127.0.0.1:443",
			"--server-username=admin",
			"--server-password=admin",
			"--server-insecure-tls",
			"setup", "1.2.3.4", "generic", "node1",
		}
		assert.Equal(t, expected, args)
		assert.True(t, switchedToTLS)
	})
	t.Run("DisableCollectors", func(t *testing.T) {
		cmd := &ConfigCommand{
			NodeAddress:       "1.2.3.4",
			NodeType:          "generic",
			NodeName:          "node1",
			DisableCollectors: []string{"cpu", "diskstats"},
		}
		u, err := url.Parse("http://admin:admin@127.0.0.1")
		require.NoError(t, err)
		args, switchedToTLS := cmd.args(&flags.GlobalFlags{ServerURL: u})
		expected := []string{
			"--server-address=127.0.0.1:443",
			"--server-username=admin",
			"--server-password=admin",
			"--server-insecure-tls",
			"setup",
			"--disable-collectors=cpu,diskstats",
			"1.2.3.4", "generic", "node1",
		}
		assert.Equal(t, expected, args)
		assert.True(t, switchedToTLS)
	})

	t.Run("LoggingLevel", func(t *testing.T) {
		cmd := &ConfigCommand{
			NodeAddress: "1.2.3.4",
			NodeType:    "generic",
			NodeName:    "node1",
			LogLevelFatalFlags: flags.LogLevelFatalFlags{
				LogLevel: "info",
			},
		}

		u, err := url.Parse("http://admin:admin@127.0.0.1")
		require.NoError(t, err)
		args, switchedToTLS := cmd.args(&flags.GlobalFlags{
			ServerURL:   u,
			EnableDebug: true,
			EnableTrace: true,
		})
		expected := []string{
			"--server-address=127.0.0.1:443",
			"--server-username=admin",
			"--server-password=admin",
			"--server-insecure-tls",
			"--log-level=info",
			"--debug",
			"--trace",
			"setup",
			"1.2.3.4",
			"generic",
			"node1",
		}
		assert.Equal(t, expected, args)
		assert.True(t, switchedToTLS)
	})

	t.Run("ConfigFileKey", func(t *testing.T) {
		// `pmm-agent setup` reads the configuration file to tell whether this Node is already registered,
		// and without the key an encrypted file reads exactly like a damaged one. The flag belongs to
		// pmm-agent rather than to its setup subcommand, so it goes before `setup`. The key file password
		// is not passed here: an argv is readable by every local user, and RunCmd hands it to the child
		// in the environment instead.
		cmd := &ConfigCommand{
			NodeAddress:           "1.2.3.4",
			NodeType:              "generic",
			NodeName:              "node1",
			ConfigFileKeyFile:     "/etc/percona/pmm/encryption.key",
			ConfigFileKeyPassword: "keypass",
		}

		u, err := url.Parse("https://admin:admin@127.0.0.1:443")
		require.NoError(t, err)
		args, switchedToTLS := cmd.args(&flags.GlobalFlags{ServerURL: u})
		expected := []string{
			"--server-address=127.0.0.1:443",
			"--server-username=admin",
			"--server-password=admin",
			"--config-file-key-file=/etc/percona/pmm/encryption.key",
			"setup", "1.2.3.4", "generic", "node1",
		}
		assert.Equal(t, expected, args)
		assert.False(t, switchedToTLS)
		assert.NotContains(t, strings.Join(args, " "), "keypass")
	})
}
