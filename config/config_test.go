// pmm-agent
// Copyright 2019 Percona LLC
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
		actual, configFilepath, err := get([]string{
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
				NodeExporter:     "/usr/local/percona/pmm2/exporters/node_exporter",
				MySQLdExporter:   "/usr/local/percona/pmm2/exporters/mysqld_exporter",
				MongoDBExporter:  "/usr/local/percona/pmm2/exporters/mongodb_exporter",
				PostgresExporter: "/usr/local/percona/pmm2/exporters/postgres_exporter",
				ProxySQLExporter: "/usr/local/percona/pmm2/exporters/proxysql_exporter",
				TempDir:          os.TempDir(),
			},
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
		}
		assert.Equal(t, expected, actual)
		assert.Empty(t, configFilepath)
	})

	t.Run("OnlyConfig", func(t *testing.T) {
		name := writeConfig(t, &Config{
			ID: "agent-id",
			Server: Server{
				Address: "127.0.0.1",
			},
		})
		defer removeConfig(t, name)

		actual, configFilepath, err := get([]string{
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
				NodeExporter:     "/usr/local/percona/pmm2/exporters/node_exporter",
				MySQLdExporter:   "/usr/local/percona/pmm2/exporters/mysqld_exporter",
				MongoDBExporter:  "/usr/local/percona/pmm2/exporters/mongodb_exporter",
				PostgresExporter: "/usr/local/percona/pmm2/exporters/postgres_exporter",
				ProxySQLExporter: "/usr/local/percona/pmm2/exporters/proxysql_exporter",
				TempDir:          os.TempDir(),
			},
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
		}
		assert.Equal(t, expected, actual)
		assert.Equal(t, name, configFilepath)
	})

	t.Run("Mix", func(t *testing.T) {
		name := writeConfig(t, &Config{
			ID: "config-id",
			Server: Server{
				Address: "127.0.0.1",
			},
		})
		defer removeConfig(t, name)

		actual, configFilepath, err := get([]string{
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
				NodeExporter:     "/usr/local/percona/pmm2/exporters/node_exporter",
				MySQLdExporter:   "/usr/local/percona/pmm2/exporters/mysqld_exporter",
				MongoDBExporter:  "/usr/local/percona/pmm2/exporters/mongodb_exporter",
				PostgresExporter: "/usr/local/percona/pmm2/exporters/postgres_exporter",
				ProxySQLExporter: "/usr/local/percona/pmm2/exporters/proxysql_exporter",
				TempDir:          os.TempDir(),
			},
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
			Debug: true,
		}
		assert.Equal(t, expected, actual)
		assert.Equal(t, name, configFilepath)
	})

	t.Run("MixExportersBase", func(t *testing.T) {
		name := writeConfig(t, &Config{
			ID: "config-id",
			Server: Server{
				Address: "127.0.0.1",
			},
			Paths: Paths{
				PostgresExporter: "/bar/postgres_exporter",
				ProxySQLExporter: "pro_exporter",
			},
		})
		defer removeConfig(t, name)

		actual, configFilepath, err := get([]string{
			"--config-file=" + name,
			"--id=flag-id",
			"--debug",
			"--paths-exporters_base=/base",
			"--paths-mysqld_exporter=/foo/mysqld_exporter",
			"--paths-mongodb_exporter=mongo_exporter",
		}, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expected := &Config{
			ID:         "flag-id",
			ListenPort: 7777,
			Server: Server{
				Address: "127.0.0.1:443",
			},
			Paths: Paths{
				ExportersBase:    "/base",
				NodeExporter:     "/base/node_exporter",    // default value
				MySQLdExporter:   "/foo/mysqld_exporter",   // respect absolute value from flag
				MongoDBExporter:  "/base/mongo_exporter",   // respect relative value from flag
				PostgresExporter: "/bar/postgres_exporter", // respect absolute value from config file
				ProxySQLExporter: "/base/pro_exporter",     // respect relative value from config file
				TempDir:          os.TempDir(),
			},
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
			Debug: true,
		}
		assert.Equal(t, expected, actual)
		assert.Equal(t, name, configFilepath)
	})

	t.Run("NoFile", func(t *testing.T) {
		wd, err := os.Getwd()
		require.NoError(t, err)
		name := t.Name()
		actual, configFilepath, err := get([]string{
			"--config-file=" + name,
			"--id=flag-id",
			"--debug",
		}, logrus.WithField("test", t.Name()))
		expected := &Config{
			ID:         "flag-id",
			ListenPort: 7777,
			Paths: Paths{
				ExportersBase:    "/usr/local/percona/pmm2/exporters",
				NodeExporter:     "/usr/local/percona/pmm2/exporters/node_exporter",
				MySQLdExporter:   "/usr/local/percona/pmm2/exporters/mysqld_exporter",
				MongoDBExporter:  "/usr/local/percona/pmm2/exporters/mongodb_exporter",
				PostgresExporter: "/usr/local/percona/pmm2/exporters/postgres_exporter",
				ProxySQLExporter: "/usr/local/percona/pmm2/exporters/proxysql_exporter",
				TempDir:          os.TempDir(),
			},
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
			Debug: true,
		}
		assert.Equal(t, expected, actual)
		assert.Equal(t, filepath.Join(wd, name), configFilepath)
		assert.Equal(t, ErrConfigFileDoesNotExist(filepath.Join(wd, name)), err)
	})
}

func TestFilteredURL(t *testing.T) {
	s := &Server{
		Address:  "1.2.3.4:443",
		Username: "username",
	}
	require.Equal(t, "https://username@1.2.3.4:443/", s.URL().String())
	require.Equal(t, "https://username@1.2.3.4:443/", s.FilteredURL())

	for _, password := range []string{
		"password",
		"$&+,/:*;=?@", // all special reserved characters from RFC plus *
	} {
		t.Run(password, func(t *testing.T) {
			s.Password = password
			assert.Equal(t, "https://username:***@1.2.3.4:443/", s.FilteredURL())
		})
	}
}
