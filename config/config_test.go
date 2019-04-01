// pmm-agent
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

package config

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func writeConfig(t *testing.T, cfg *Config) string {
	b, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	f, err := ioutil.TempFile("", "pmm-agent-test-")
	require.NoError(t, err)
	_, err = f.Write(b)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func removeConfig(t *testing.T, name string) {
	require.NoError(t, os.Remove(name))
}

func TestConfig(t *testing.T) {
	t.Run("OnlyFlags", func(t *testing.T) {
		actual, err := Get([]string{
			"--id=agent-id",
			"--address=127.0.0.1:11111",
		}, logrus.WithField("test", t.Name))
		require.NoError(t, err)

		expected := &Config{
			ID:         "agent-id",
			Address:    "127.0.0.1:11111",
			ListenPort: 7777,
			Paths: Paths{
				TempDir: os.TempDir(),
			},
			Ports: Ports{
				Min: 32768,
				Max: 60999,
			},
		}
		assert.Equal(t, expected, actual)
	})

	t.Run("OnlyConfig", func(t *testing.T) {
		name := writeConfig(t, &Config{
			ID:      "agent-id",
			Address: "127.0.0.1:11111",
		})
		defer removeConfig(t, name)

		actual, err := Get([]string{
			"--config-file=" + name,
		}, logrus.WithField("test", t.Name))
		require.NoError(t, err)

		expected := &Config{
			ID:         "agent-id",
			Address:    "127.0.0.1:11111",
			ListenPort: 7777,
			Paths: Paths{
				TempDir: os.TempDir(),
			},
			Ports: Ports{
				Min: 32768,
				Max: 60999,
			},
		}
		assert.Equal(t, expected, actual)
	})

	t.Run("Mix", func(t *testing.T) {
		name := writeConfig(t, &Config{
			ID:      "config-id",
			Address: "127.0.0.1:11111",
		})
		defer removeConfig(t, name)

		actual, err := Get([]string{
			"--config-file=" + name,
			"--id=flag-id",
			"--debug",
		}, logrus.WithField("test", t.Name))
		require.NoError(t, err)

		expected := &Config{
			ID:         "flag-id",
			Address:    "127.0.0.1:11111",
			ListenPort: 7777,
			Debug:      true,
			Paths: Paths{
				TempDir: os.TempDir(),
			},
			Ports: Ports{
				Min: 32768,
				Max: 60999,
			},
		}
		assert.Equal(t, expected, actual)
	})
}
