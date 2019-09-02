// pmm-admin
// Copyright (C) 2018 Percona LLC
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
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigCommandArgs(t *testing.T) {
	cmd := &configCommand{
		NodeAddress: "1.2.3.4",
		NodeType:    "generic",
		NodeName:    "node1",
	}

	t.Run("SwitchToTLS1", func(t *testing.T) {
		u, err := url.Parse("http://127.0.0.1:80")
		require.NoError(t, err)
		GlobalFlags = &globalFlagsValues{
			ServerURL: u,
		}
		args, switchedToTLS := cmd.args()
		expected := []string{
			"--server-address=127.0.0.1:443",
			"--server-insecure-tls",
			"setup", "1.2.3.4", "generic", "node1",
		}
		assert.Equal(t, expected, args)
		assert.True(t, switchedToTLS)
	})

	t.Run("SwitchToTLS2", func(t *testing.T) {
		cmd := &configCommand{
			NodeAddress: "1.2.3.4",
			NodeType:    "generic",
			NodeName:    "node1",
		}
		u, err := url.Parse("http://admin:admin@127.0.0.1")
		require.NoError(t, err)
		GlobalFlags = &globalFlagsValues{
			ServerURL: u,
		}
		args, switchedToTLS := cmd.args()
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
}
