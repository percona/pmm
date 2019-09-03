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
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func writeConfig(t *testing.T, cfg *Config) string {
	f, err := ioutil.TempFile("", "pmm-agent-test-")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, SaveToFile(f.Name(), cfg, t.Name()))
	return f.Name()
}

func removeConfig(t *testing.T, name string) {
	require.NoError(t, os.Remove(name))
}

func TestLoadFromFile(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		name := writeConfig(t, &Config{ID: "agent-id"})
		defer removeConfig(t, name)

		cfg, err := loadFromFile(name)
		require.NoError(t, err)
		assert.Equal(t, &Config{ID: "agent-id"}, cfg)
	})

	t.Run("NotExist", func(t *testing.T) {
		cfg, err := loadFromFile("not-exist.yaml")
		assert.Equal(t, ErrConfigFileDoesNotExist("not-exist.yaml"), err)
		assert.Nil(t, cfg)
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		name := writeConfig(t, &Config{ID: "agent-id"})
		require.NoError(t, os.Chmod(name, 0000))
		defer removeConfig(t, name)

		cfg, err := loadFromFile(name)
		require.IsType(t, (*os.PathError)(nil), err)
		assert.Equal(t, "open", err.(*os.PathError).Op)
		assert.EqualError(t, err.(*os.PathError).Err, `permission denied`)
		assert.Nil(t, cfg)
	})

	t.Run("NotYAML", func(t *testing.T) {
		name := writeConfig(t, nil)
		require.NoError(t, ioutil.WriteFile(name, []byte(`not YAML`), 0666))
		defer removeConfig(t, name)

		cfg, err := loadFromFile(name)
		require.IsType(t, (*yaml.TypeError)(nil), err)
		assert.EqualError(t, err, "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `not YAML` into config.Config")
		assert.Nil(t, cfg)
	})
}

func TestGet(t *testing.T) {
	t.Run("OnlyFlags", func(t *testing.T) {
		actual, configFilePath, err := get([]string{
			"--id=agent-id",
			"--server-address=127.0.0.1",
		}, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expected := &Config{
			ID:         "agent-id",
			ListenPort: 7777,
			Server: Server{
				Address: "127.0.0.1:443",
			},
			Paths: Paths{
				ExportersBase:    "/usr/local/percona/pmm2/exporters",
				NodeExporter:     "node_exporter",
				MySQLdExporter:   "mysqld_exporter",
				MongoDBExporter:  "mongodb_exporter",
				PostgresExporter: "postgres_exporter",
				ProxySQLExporter: "proxysql_exporter",
				PtSummary:        "pt-summary",
				PtMySQLSummary:   "pt-mysql-summary",
				TempDir:          os.TempDir(),
			},
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
		}
		assert.Equal(t, expected, actual)
		assert.Empty(t, configFilePath)
	})

	t.Run("OnlyConfig", func(t *testing.T) {
		name := writeConfig(t, &Config{
			ID: "agent-id",
			Server: Server{
				Address: "127.0.0.1",
			},
		})
		defer removeConfig(t, name)

		actual, configFilePath, err := get([]string{
			"--config-file=" + name,
		}, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expected := &Config{
			ID:         "agent-id",
			ListenPort: 7777,
			Server: Server{
				Address: "127.0.0.1:443",
			},
			Paths: Paths{
				ExportersBase:    "/usr/local/percona/pmm2/exporters",
				NodeExporter:     "node_exporter",
				MySQLdExporter:   "mysqld_exporter",
				MongoDBExporter:  "mongodb_exporter",
				PostgresExporter: "postgres_exporter",
				ProxySQLExporter: "proxysql_exporter",
				PtSummary:        "pt-summary",
				PtMySQLSummary:   "pt-mysql-summary",
				TempDir:          os.TempDir(),
			},
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
		}
		assert.Equal(t, expected, actual)
		assert.Equal(t, name, configFilePath)
	})

	t.Run("Mix", func(t *testing.T) {
		name := writeConfig(t, &Config{
			ID: "config-id",
			Server: Server{
				Address: "127.0.0.1",
			},
		})
		defer removeConfig(t, name)

		actual, configFilePath, err := get([]string{
			"--config-file=" + name,
			"--id=flag-id",
			"--debug",
		}, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expected := &Config{
			ID:         "flag-id",
			ListenPort: 7777,
			Server: Server{
				Address: "127.0.0.1:443",
			},
			Paths: Paths{
				ExportersBase:    "/usr/local/percona/pmm2/exporters",
				NodeExporter:     "node_exporter",
				MySQLdExporter:   "mysqld_exporter",
				MongoDBExporter:  "mongodb_exporter",
				PostgresExporter: "postgres_exporter",
				ProxySQLExporter: "proxysql_exporter",
				PtSummary:        "pt-summary",
				PtMySQLSummary:   "pt-mysql-summary",
				TempDir:          os.TempDir(),
			},
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
			Debug: true,
		}
		assert.Equal(t, expected, actual)
		assert.Equal(t, name, configFilePath)
	})

	t.Run("NoFile", func(t *testing.T) {
		wd, err := os.Getwd()
		require.NoError(t, err)
		name := t.Name()
		actual, configFilePath, err := get([]string{
			"--config-file=" + name,
			"--id=flag-id",
			"--debug",
		}, logrus.WithField("test", t.Name()))
		expected := &Config{
			ID:         "flag-id",
			ListenPort: 7777,
			Paths: Paths{
				ExportersBase:    "/usr/local/percona/pmm2/exporters",
				NodeExporter:     "node_exporter",
				MySQLdExporter:   "mysqld_exporter",
				MongoDBExporter:  "mongodb_exporter",
				PostgresExporter: "postgres_exporter",
				ProxySQLExporter: "proxysql_exporter",
				PtSummary:        "pt-summary",
				PtMySQLSummary:   "pt-mysql-summary",
				TempDir:          os.TempDir(),
			},
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
			Debug: true,
		}
		assert.Equal(t, expected, actual)
		assert.Equal(t, filepath.Join(wd, name), configFilePath)
		assert.Equal(t, ErrConfigFileDoesNotExist(filepath.Join(wd, name)), err)
	})
}
