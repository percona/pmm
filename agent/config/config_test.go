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

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const defaultWindowPeriod = time.Hour

func writeConfig(t *testing.T, cfg *Config) string {
	t.Helper()
	f, err := os.CreateTemp("", "pmm-agent-test-")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, SaveToFile(f.Name(), cfg, t.Name()))
	return f.Name()
}

func removeConfig(t *testing.T, name string) {
	t.Helper()
	require.NoError(t, os.Remove(name))
}

func generateTempDirPath(t *testing.T, basePath string) string {
	t.Helper()
	return filepath.Join(basePath, agentTmpPath)
}

func TestLoadFromFile(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		name := writeConfig(t, &Config{ID: "agent-id"})
		t.Cleanup(func() { removeConfig(t, name) })

		cfg, err := loadFromFile(name)
		require.NoError(t, err)
		assert.Equal(t, &Config{ID: "agent-id"}, cfg)
	})

	t.Run("NotExist", func(t *testing.T) {
		cfg, err := loadFromFile("not-exist.yaml")
		assert.Equal(t, ConfigFileDoesNotExistError("not-exist.yaml"), err)
		assert.Nil(t, cfg)
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		name := writeConfig(t, &Config{ID: "agent-id"})
		require.NoError(t, os.Chmod(name, 0o000))
		t.Cleanup(func() { removeConfig(t, name) })

		cfg, err := loadFromFile(name)
		require.IsType(t, (*os.PathError)(nil), err)
		assert.Equal(t, "open", err.(*os.PathError).Op)                     //nolint:errorlint
		require.EqualError(t, err.(*os.PathError).Err, "permission denied") //nolint:errorlint
		assert.Nil(t, cfg)
	})

	t.Run("NotYAML", func(t *testing.T) {
		name := writeConfig(t, nil)
		require.NoError(t, os.WriteFile(name, []byte(`not YAML`), 0o666)) //nolint:gosec
		t.Cleanup(func() { removeConfig(t, name) })

		cfg, err := loadFromFile(name)
		require.IsType(t, (*yaml.TypeError)(nil), err)
		require.EqualError(t, err, "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `not YAML` into config.Config")
		assert.Nil(t, cfg)
	})
}

func TestGet(t *testing.T) {
	t.Run("OnlyFlags", func(t *testing.T) {
		var actual Config
		configFilepath, err := get([]string{
			"--id=agent-id",
			"--listen-port=9999",
			"--server-address=127.0.0.1",
		}, &actual, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expected := Config{
			ID:            "agent-id",
			ListenAddress: "127.0.0.1",
			ListenPort:    9999,
			Server: Server{
				Address: "127.0.0.1:443",
			},
			Paths: Paths{
				PathsBase:        "/usr/local/percona/pmm",
				ExportersBase:    "/usr/local/percona/pmm/exporters",
				NodeExporter:     "/usr/local/percona/pmm/exporters/node_exporter",
				MySQLdExporter:   "/usr/local/percona/pmm/exporters/mysqld_exporter",
				MongoDBExporter:  "/usr/local/percona/pmm/exporters/mongodb_exporter",
				PostgresExporter: "/usr/local/percona/pmm/exporters/postgres_exporter",
				ProxySQLExporter: "/usr/local/percona/pmm/exporters/proxysql_exporter",
				RDSExporter:      "/usr/local/percona/pmm/exporters/rds_exporter",
				AzureExporter:    "/usr/local/percona/pmm/exporters/azure_exporter",
				ValkeyExporter:   "/usr/local/percona/pmm/exporters/valkey_exporter",
				VMAgent:          "/usr/local/percona/pmm/exporters/vmagent",
				TempDir:          "/usr/local/percona/pmm/tmp",
				NomadDataDir:     "/usr/local/percona/pmm/data/nomad",
				PTSummary:        "/usr/local/percona/pmm/tools/pt-summary",
				PTPGSummary:      "/usr/local/percona/pmm/tools/pt-pg-summary",
				PTMySQLSummary:   "/usr/local/percona/pmm/tools/pt-mysql-summary",
				PTMongoDBSummary: "/usr/local/percona/pmm/tools/pt-mongodb-summary",
				Nomad:            "/usr/local/percona/pmm/tools/nomad",
			},
			WindowConnectedTime: defaultWindowPeriod,
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
			LogLinesCount:         1024,
			PerfschemaRefreshRate: 5,
		}
		assert.Equal(t, expected, actual)
		assert.Empty(t, configFilepath)
	})

	t.Run("OnlyConfig", func(t *testing.T) {
		var name string
		var actual Config

		tmpDir := generateTempDirPath(t, pathBaseDefault)
		t.Cleanup(func() {
			removeConfig(t, name)
		})

		name = writeConfig(t, &Config{
			ID:            "agent-id",
			ListenAddress: "0.0.0.0",
			Server: Server{
				Address: "127.0.0.1",
			},
			Paths: Paths{
				TempDir: tmpDir,
			},
			PerfschemaRefreshRate: 2,
		})

		configFilepath, err := get([]string{
			"--config-file=" + name,
		}, &actual, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expected := Config{
			ID:            "agent-id",
			ListenAddress: "0.0.0.0",
			ListenPort:    7777,
			Server: Server{
				Address: "127.0.0.1:443",
			},
			Paths: Paths{
				PathsBase:        "/usr/local/percona/pmm",
				ExportersBase:    "/usr/local/percona/pmm/exporters",
				NodeExporter:     "/usr/local/percona/pmm/exporters/node_exporter",
				MySQLdExporter:   "/usr/local/percona/pmm/exporters/mysqld_exporter",
				MongoDBExporter:  "/usr/local/percona/pmm/exporters/mongodb_exporter",
				PostgresExporter: "/usr/local/percona/pmm/exporters/postgres_exporter",
				ProxySQLExporter: "/usr/local/percona/pmm/exporters/proxysql_exporter",
				RDSExporter:      "/usr/local/percona/pmm/exporters/rds_exporter",
				AzureExporter:    "/usr/local/percona/pmm/exporters/azure_exporter",
				ValkeyExporter:   "/usr/local/percona/pmm/exporters/valkey_exporter",
				VMAgent:          "/usr/local/percona/pmm/exporters/vmagent",
				TempDir:          "/usr/local/percona/pmm/tmp",
				NomadDataDir:     "/usr/local/percona/pmm/data/nomad",
				PTSummary:        "/usr/local/percona/pmm/tools/pt-summary",
				PTPGSummary:      "/usr/local/percona/pmm/tools/pt-pg-summary",
				PTMongoDBSummary: "/usr/local/percona/pmm/tools/pt-mongodb-summary",
				PTMySQLSummary:   "/usr/local/percona/pmm/tools/pt-mysql-summary",
				Nomad:            "/usr/local/percona/pmm/tools/nomad",
			},
			WindowConnectedTime: defaultWindowPeriod,
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
			LogLinesCount:         1024,
			PerfschemaRefreshRate: 2,
		}
		assert.Equal(t, expected, actual)
		assert.Equal(t, name, configFilepath)
	})

	t.Run("BothFlagsAndConfig", func(t *testing.T) {
		var name string
		var actual Config
		tmpDir := generateTempDirPath(t, "/foo/bar")
		t.Cleanup(func() {
			removeConfig(t, name)
		})

		name = writeConfig(t, &Config{
			ID: "config-id",
			Server: Server{
				Address: "127.0.0.1",
			},
			PerfschemaRefreshRate: 2,
		})

		configFilepath, err := get([]string{
			"--config-file=" + name,
			"--paths-tempdir=" + tmpDir,
			"--id=flag-id",
			"--log-level=info",
			"--debug",
			"--perfschema-refresh-rate=1",
		}, &actual, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expected := Config{
			ID:            "flag-id",
			ListenAddress: "127.0.0.1",
			ListenPort:    7777,
			Server: Server{
				Address: "127.0.0.1:443",
			},
			Paths: Paths{
				PathsBase:        "/usr/local/percona/pmm",
				ExportersBase:    "/usr/local/percona/pmm/exporters",
				NodeExporter:     "/usr/local/percona/pmm/exporters/node_exporter",
				MySQLdExporter:   "/usr/local/percona/pmm/exporters/mysqld_exporter",
				MongoDBExporter:  "/usr/local/percona/pmm/exporters/mongodb_exporter",
				PostgresExporter: "/usr/local/percona/pmm/exporters/postgres_exporter",
				ProxySQLExporter: "/usr/local/percona/pmm/exporters/proxysql_exporter",
				RDSExporter:      "/usr/local/percona/pmm/exporters/rds_exporter",
				AzureExporter:    "/usr/local/percona/pmm/exporters/azure_exporter",
				ValkeyExporter:   "/usr/local/percona/pmm/exporters/valkey_exporter",
				VMAgent:          "/usr/local/percona/pmm/exporters/vmagent",
				TempDir:          "/foo/bar/tmp",
				NomadDataDir:     "/usr/local/percona/pmm/data/nomad",
				PTSummary:        "/usr/local/percona/pmm/tools/pt-summary",
				PTPGSummary:      "/usr/local/percona/pmm/tools/pt-pg-summary",
				PTMySQLSummary:   "/usr/local/percona/pmm/tools/pt-mysql-summary",
				PTMongoDBSummary: "/usr/local/percona/pmm/tools/pt-mongodb-summary",
				Nomad:            "/usr/local/percona/pmm/tools/nomad",
			},
			WindowConnectedTime: defaultWindowPeriod,
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
			LogLevel:              "info",
			Debug:                 true,
			LogLinesCount:         1024,
			PerfschemaRefreshRate: 1,
		}
		assert.Equal(t, expected, actual)
		assert.Equal(t, name, configFilepath)
	})

	t.Run("MixExportersBase", func(t *testing.T) {
		var name string
		var actual Config

		t.Cleanup(func() {
			removeConfig(t, name)
		})

		name = writeConfig(t, &Config{
			ID: "config-id",
			Server: Server{
				Address: "127.0.0.1",
			},
			Paths: Paths{
				PostgresExporter: "/bar/postgres_exporter",
				ProxySQLExporter: "pro_exporter",
				TempDir:          "tmp",
			},
		})

		configFilepath, err := get([]string{
			"--config-file=" + name,
			"--id=flag-id",
			"--debug",
			"--paths-exporters_base=/base",
			"--paths-mysqld_exporter=/foo/mysqld_exporter",
			"--paths-mongodb_exporter=mongo_exporter",
		}, &actual, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expected := Config{
			ID:            "flag-id",
			ListenAddress: "127.0.0.1",
			ListenPort:    7777,
			Server: Server{
				Address: "127.0.0.1:443",
			},
			Paths: Paths{
				PathsBase:        "/usr/local/percona/pmm",
				ExportersBase:    "/base",
				NodeExporter:     "/base/node_exporter",    // default value
				MySQLdExporter:   "/foo/mysqld_exporter",   // respect absolute value from flag
				MongoDBExporter:  "/base/mongo_exporter",   // respect relative value from flag
				PostgresExporter: "/bar/postgres_exporter", // respect absolute value from config file
				ProxySQLExporter: "/base/pro_exporter",     // respect relative value from config file
				RDSExporter:      "/base/rds_exporter",     // default value
				AzureExporter:    "/base/azure_exporter",   // default value
				ValkeyExporter:   "/base/valkey_exporter",  // default value
				VMAgent:          "/base/vmagent",          // default value
				TempDir:          "/usr/local/percona/pmm/tmp",
				NomadDataDir:     "/usr/local/percona/pmm/data/nomad",
				PTSummary:        "/usr/local/percona/pmm/tools/pt-summary",
				PTPGSummary:      "/usr/local/percona/pmm/tools/pt-pg-summary",
				PTMongoDBSummary: "/usr/local/percona/pmm/tools/pt-mongodb-summary",
				PTMySQLSummary:   "/usr/local/percona/pmm/tools/pt-mysql-summary",
				Nomad:            "/usr/local/percona/pmm/tools/nomad",
			},
			WindowConnectedTime: defaultWindowPeriod,
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
			Debug:                 true,
			LogLinesCount:         1024,
			PerfschemaRefreshRate: 5,
		}
		assert.Equal(t, expected, actual)
		assert.Equal(t, name, configFilepath)
	})

	t.Run("MixPathsBase", func(t *testing.T) {
		var name string
		var actual Config

		t.Cleanup(func() {
			removeConfig(t, name)
		})

		name = writeConfig(t, &Config{
			ID: "config-id",
			Server: Server{
				Address: "127.0.0.1",
			},
			Paths: Paths{
				PostgresExporter: "/foo/postgres_exporter",
				ProxySQLExporter: "/base/exporters/pro_exporter",
			},
		})

		configFilepath, err := get([]string{
			"--config-file=" + name,
			"--id=flag-id",
			"--debug",
			"--paths-base=/base",
			"--paths-mysqld_exporter=/foo/mysqld_exporter",
			"--paths-mongodb_exporter=dir/mongo_exporter",
		}, &actual, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expected := Config{
			ID:            "flag-id",
			ListenAddress: "127.0.0.1",
			ListenPort:    7777,
			Server: Server{
				Address: "127.0.0.1:443",
			},
			Paths: Paths{
				PathsBase:        "/base",
				ExportersBase:    "/base/exporters",
				NodeExporter:     "/base/exporters/node_exporter",      // default value
				MySQLdExporter:   "/foo/mysqld_exporter",               // respect absolute value from flag
				MongoDBExporter:  "/base/exporters/dir/mongo_exporter", // respect relative value from flag
				PostgresExporter: "/foo/postgres_exporter",             // respect absolute value from config file
				ProxySQLExporter: "/base/exporters/pro_exporter",       // respect relative value from config file
				RDSExporter:      "/base/exporters/rds_exporter",       // default value
				AzureExporter:    "/base/exporters/azure_exporter",     // default value
				ValkeyExporter:   "/base/exporters/valkey_exporter",    // default value
				VMAgent:          "/base/exporters/vmagent",            // default value
				TempDir:          "/base/tmp",
				NomadDataDir:     "/base/data/nomad",
				PTSummary:        "/base/tools/pt-summary",
				PTPGSummary:      "/base/tools/pt-pg-summary",
				PTMongoDBSummary: "/base/tools/pt-mongodb-summary",
				PTMySQLSummary:   "/base/tools/pt-mysql-summary",
				Nomad:            "/base/tools/nomad",
			},
			WindowConnectedTime: defaultWindowPeriod,
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
			Debug:                 true,
			LogLinesCount:         1024,
			PerfschemaRefreshRate: 5,
		}
		assert.Equal(t, expected, actual)
		assert.Equal(t, name, configFilepath)
	})

	t.Run("MixPathsBaseExporterBase", func(t *testing.T) {
		var name string
		var actual Config

		t.Cleanup(func() {
			removeConfig(t, name)
		})

		name = writeConfig(t, &Config{
			ID: "config-id",
			Server: Server{
				Address: "127.0.0.1",
			},
			Paths: Paths{
				ExportersBase: "/foo/exporters",
				TempDir:       "/foo/tmp",
			},
		})

		configFilepath, err := get([]string{
			"--config-file=" + name,
			"--id=flag-id",
			"--debug",
			"--paths-base=/base",
		}, &actual, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expected := Config{
			ID:            "flag-id",
			ListenAddress: "127.0.0.1",
			ListenPort:    7777,
			Server: Server{
				Address: "127.0.0.1:443",
			},
			Paths: Paths{
				PathsBase:        "/base",
				ExportersBase:    "/foo/exporters",
				NodeExporter:     "/foo/exporters/node_exporter",     // default value
				MySQLdExporter:   "/foo/exporters/mysqld_exporter",   // default value
				MongoDBExporter:  "/foo/exporters/mongodb_exporter",  // default value
				PostgresExporter: "/foo/exporters/postgres_exporter", // default value
				ProxySQLExporter: "/foo/exporters/proxysql_exporter", // default value
				RDSExporter:      "/foo/exporters/rds_exporter",      // default value
				AzureExporter:    "/foo/exporters/azure_exporter",    // default value
				ValkeyExporter:   "/foo/exporters/valkey_exporter",   // default value
				VMAgent:          "/foo/exporters/vmagent",           // default value
				TempDir:          "/foo/tmp",
				NomadDataDir:     "/base/data/nomad",
				PTSummary:        "/base/tools/pt-summary",
				PTPGSummary:      "/base/tools/pt-pg-summary",
				PTMongoDBSummary: "/base/tools/pt-mongodb-summary",
				PTMySQLSummary:   "/base/tools/pt-mysql-summary",
				Nomad:            "/base/tools/nomad",
			},
			WindowConnectedTime: defaultWindowPeriod,
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
			Debug:                 true,
			LogLinesCount:         1024,
			PerfschemaRefreshRate: 5,
		}
		assert.Equal(t, expected, actual)
		assert.Equal(t, name, configFilepath)
	})

	t.Run("NoFile", func(t *testing.T) {
		wd, err := os.Getwd()
		require.NoError(t, err)

		name := t.Name()
		tmpDir := generateTempDirPath(t, pathBaseDefault)

		var actual Config
		configFilepath, err := get([]string{
			"--config-file=" + name,
			"--paths-tempdir=" + tmpDir,
			"--id=flag-id",
			"--debug",
		}, &actual, logrus.WithField("test", name))

		expected := Config{
			ID:            "flag-id",
			ListenAddress: "127.0.0.1",
			ListenPort:    7777,
			Paths: Paths{
				PathsBase:        "/usr/local/percona/pmm",
				ExportersBase:    "/usr/local/percona/pmm/exporters",
				NodeExporter:     "/usr/local/percona/pmm/exporters/node_exporter",
				MySQLdExporter:   "/usr/local/percona/pmm/exporters/mysqld_exporter",
				MongoDBExporter:  "/usr/local/percona/pmm/exporters/mongodb_exporter",
				PostgresExporter: "/usr/local/percona/pmm/exporters/postgres_exporter",
				ProxySQLExporter: "/usr/local/percona/pmm/exporters/proxysql_exporter",
				RDSExporter:      "/usr/local/percona/pmm/exporters/rds_exporter",
				AzureExporter:    "/usr/local/percona/pmm/exporters/azure_exporter",
				ValkeyExporter:   "/usr/local/percona/pmm/exporters/valkey_exporter",
				VMAgent:          "/usr/local/percona/pmm/exporters/vmagent",
				TempDir:          "/usr/local/percona/pmm/tmp",
				NomadDataDir:     "/usr/local/percona/pmm/data/nomad",
				PTSummary:        "/usr/local/percona/pmm/tools/pt-summary",
				PTPGSummary:      "/usr/local/percona/pmm/tools/pt-pg-summary",
				PTMongoDBSummary: "/usr/local/percona/pmm/tools/pt-mongodb-summary",
				PTMySQLSummary:   "/usr/local/percona/pmm/tools/pt-mysql-summary",
				Nomad:            "/usr/local/percona/pmm/tools/nomad",
			},
			WindowConnectedTime: defaultWindowPeriod,
			Ports: Ports{
				Min: 42000,
				Max: 51999,
			},
			Debug:                 true,
			LogLinesCount:         1024,
			PerfschemaRefreshRate: 5,
		}
		assert.Equal(t, expected, actual)
		assert.Equal(t, filepath.Join(wd, name), configFilepath)
		assert.Equal(t, ConfigFileDoesNotExistError(filepath.Join(wd, name)), err)
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
