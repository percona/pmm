// Copyright (C) 2023 Percona LLC
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

package agents

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

func TestMySQLdExporterConfig(t *testing.T) {
	mysql := &models.Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(3306),
	}
	node := &models.Node{
		Address: "1.2.3.4",
	}
	exporter := &models.Agent{
		AgentID:         "agent-id",
		AgentType:       models.MySQLdExporterType,
		Username:        pointer.ToString("username"),
		Password:        pointer.ToString("s3cur3 p@$$w0r4."),
		AgentPassword:   pointer.ToString("agent-password"),
		ExporterOptions: models.ExporterOptions{},
	}
	pmmAgentVersion := version.MustParse("2.21.0")

	actual, err := mysqldExporterConfig(node, mysql, exporter, redactSecrets, pmmAgentVersion)
	expected := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--collect.auto_increment.columns",
			"--collect.binlog_size",
			"--collect.custom_query.hr",
			"--collect.custom_query.hr.directory=/usr/local/percona/pmm/collectors/custom-queries/mysql/high-resolution",
			"--collect.custom_query.lr",
			"--collect.custom_query.lr.directory=/usr/local/percona/pmm/collectors/custom-queries/mysql/low-resolution",
			"--collect.custom_query.mr",
			"--collect.custom_query.mr.directory=/usr/local/percona/pmm/collectors/custom-queries/mysql/medium-resolution",
			"--collect.engine_innodb_status",
			"--collect.engine_tokudb_status",
			"--collect.global_status",
			"--collect.global_variables",
			"--collect.heartbeat",
			"--collect.info_schema.clientstats",
			"--collect.info_schema.innodb_cmp",
			"--collect.info_schema.innodb_cmpmem",
			"--collect.info_schema.innodb_metrics",
			"--collect.info_schema.innodb_tablespaces",
			"--collect.info_schema.processlist",
			"--collect.info_schema.query_response_time",
			"--collect.info_schema.tables",
			"--collect.info_schema.tablestats",
			"--collect.info_schema.userstats",
			"--collect.perf_schema.eventsstatements",
			"--collect.perf_schema.eventswaits",
			"--collect.perf_schema.file_events",
			"--collect.perf_schema.file_instances",
			"--collect.perf_schema.indexiowaits",
			"--collect.perf_schema.tableiowaits",
			"--collect.perf_schema.tablelocks",
			"--collect.slave_status",
			"--collect.standard.go",
			"--collect.standard.process",
			"--exporter.conn-max-lifetime=55s",
			"--exporter.global-conn-pool",
			"--exporter.max-idle-conns=3",
			"--exporter.max-open-conns=3",
			"--web.listen-address=0.0.0.0:{{ .listen_port }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:3306)/?timeout=1s",
			"HTTP_AUTH=pmm:agent-password",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4.", "agent-password"},
	}
	requireNoDuplicateFlags(t, actual.Args)
	require.NoError(t, err)
	require.Equal(t, expected.Args, actual.Args)
	require.Equal(t, expected.Env, actual.Env)
	require.Equal(t, expected, actual)

	t.Run("EmptyPassword", func(t *testing.T) {
		exporter.Password = nil
		actual, err := mysqldExporterConfig(node, mysql, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		assert.Equal(t, "DATA_SOURCE_NAME=username@tcp(1.2.3.4:3306)/?timeout=1s", actual.Env[0])
	})

	t.Run("EmptyUsername", func(t *testing.T) {
		exporter.Username = nil
		actual, err := mysqldExporterConfig(node, mysql, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		assert.Equal(t, "DATA_SOURCE_NAME=tcp(1.2.3.4:3306)/?timeout=1s", actual.Env[0])
	})

	t.Run("SSLEnabled", func(t *testing.T) {
		exporter.TLS = true
		exporter.MySQLOptions = models.MySQLOptions{
			TLSCa:   "content-of-tls-ca",
			TLSCert: "content-of-tls-certificate-key",
			TLSKey:  "content-of-tls-key",
		}
		actual, err := mysqldExporterConfig(node, mysql, exporter, exposeSecrets, pmmAgentVersion)
		expected := "DATA_SOURCE_NAME=tcp(1.2.3.4:3306)/?timeout=1s&tls=custom"
		assert.Equal(t, expected, actual.Env[0])
		expectedFiles := map[string]string{
			"tlsCa":   exporter.MySQLOptions.TLSCa,
			"tlsCert": exporter.MySQLOptions.TLSCert,
			"tlsKey":  exporter.MySQLOptions.TLSKey,
		}
		require.NoError(t, err)
		assert.Equal(t, expectedFiles, actual.TextFiles)
	})

	t.Run("with allowCleartextPasswords dsn param", func(t *testing.T) {
		pmmAgentVersion = version.MustParse("3.4.0")
		t.Cleanup(func() {
			pmmAgentVersion = version.MustParse("2.21.0")
		})
		exporter.MySQLOptions = models.MySQLOptions{
			ExtraDSNParams: map[string]string{
				"allowCleartextPasswords": "true",
			},
		}
		actual, err := mysqldExporterConfig(node, mysql, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		assert.Contains(t, actual.TextFiles, "myCnf")
	})
}

func TestMySQLdExporterConfigTablestatsGroupDisabled(t *testing.T) {
	node := &models.Node{
		Address: "1.2.3.4",
	}
	mysql := &models.Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(3306),
	}
	exporter := &models.Agent{
		AgentID:         "agent-id",
		AgentType:       models.MySQLdExporterType,
		Username:        pointer.ToString("username"),
		Password:        pointer.ToString("s3cur3 p@$$w0r4."),
		TLS:             true,
		ExporterOptions: models.ExporterOptions{},
		MySQLOptions: models.MySQLOptions{
			TableCountTablestatsGroupLimit: -1,
			TLSCa:                          "content-of-tls-ca",
			TLSCert:                        "content-of-tls-cert",
			TLSKey:                         "content-of-tls-key",
		},
	}
	pmmAgentVersion := version.MustParse("2.24.0")

	actual, err := mysqldExporterConfig(node, mysql, exporter, redactSecrets, pmmAgentVersion)
	expected := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--collect.binlog_size",
			"--collect.custom_query.hr",
			"--collect.custom_query.hr.directory={{ .paths_base }}/collectors/custom-queries/mysql/high-resolution",
			"--collect.custom_query.lr",
			"--collect.custom_query.lr.directory={{ .paths_base }}/collectors/custom-queries/mysql/low-resolution",
			"--collect.custom_query.mr",
			"--collect.custom_query.mr.directory={{ .paths_base }}/collectors/custom-queries/mysql/medium-resolution",
			"--collect.engine_innodb_status",
			"--collect.engine_tokudb_status",
			"--collect.global_status",
			"--collect.global_variables",
			"--collect.heartbeat",
			"--collect.info_schema.clientstats",
			"--collect.info_schema.innodb_cmp",
			"--collect.info_schema.innodb_cmpmem",
			"--collect.info_schema.innodb_metrics",
			"--collect.info_schema.processlist",
			"--collect.info_schema.query_response_time",
			"--collect.info_schema.userstats",
			"--collect.perf_schema.eventsstatements",
			"--collect.perf_schema.eventswaits",
			"--collect.perf_schema.file_events",
			"--collect.slave_status",
			"--collect.standard.go",
			"--collect.standard.process",
			"--exporter.conn-max-lifetime=55s",
			"--exporter.global-conn-pool",
			"--exporter.max-idle-conns=3",
			"--exporter.max-open-conns=3",
			"--mysql.ssl-ca-file={{ .TextFiles.tlsCa }}",
			"--mysql.ssl-cert-file={{ .TextFiles.tlsCert }}",
			"--mysql.ssl-key-file={{ .TextFiles.tlsKey }}",
			"--web.listen-address=0.0.0.0:{{ .listen_port }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:3306)/?timeout=1s&tls=custom",
			"HTTP_AUTH=pmm:agent-id",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4.", "content-of-tls-key"},
		TextFiles: map[string]string{
			"tlsCa":   "content-of-tls-ca",
			"tlsCert": "content-of-tls-cert",
			"tlsKey":  "content-of-tls-key",
		},
	}
	requireNoDuplicateFlags(t, actual.Args)
	require.NoError(t, err)
	require.Equal(t, expected.Args, actual.Args)
	require.Equal(t, expected.Env, actual.Env)
	require.Equal(t, expected, actual)

	t.Run("EmptyPassword", func(t *testing.T) {
		exporter.Password = nil
		actual, err := mysqldExporterConfig(node, mysql, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		assert.Equal(t, "DATA_SOURCE_NAME=username@tcp(1.2.3.4:3306)/?timeout=1s&tls=custom", actual.Env[0])
	})

	t.Run("EmptyUsername", func(t *testing.T) {
		exporter.Username = nil
		actual, err := mysqldExporterConfig(node, mysql, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		assert.Equal(t, "DATA_SOURCE_NAME=tcp(1.2.3.4:3306)/?timeout=1s&tls=custom", actual.Env[0])
	})

	t.Run("V236_EnablesPluginCollector", func(t *testing.T) {
		pmmAgentVersion := version.MustParse("2.36.0")
		actual, err := mysqldExporterConfig(node, mysql, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		assert.Contains(t, actual.Args, "--collect.plugins")
	})

	t.Run("beforeV236_NoPluginCollector", func(t *testing.T) {
		pmmAgentVersion := version.MustParse("2.35.0")
		actual, err := mysqldExporterConfig(node, mysql, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		assert.NotContains(t, actual.Args, "--collect.plugins")
	})
}

func TestMySQLdExporterConfigDisabledCollectors(t *testing.T) {
	node := &models.Node{
		Address: "1.2.3.4",
	}
	mysql := &models.Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(3306),
	}
	exporter := &models.Agent{
		AgentID:   "agent-id",
		AgentType: models.MySQLdExporterType,
		Username:  pointer.ToString("username"),
		Password:  pointer.ToString("s3cur3 p@$$w0r4."),
		ExporterOptions: models.ExporterOptions{
			DisabledCollectors: []string{"heartbeat", "info_schema.clientstats", "perf_schema.eventsstatements", "custom_query.hr"},
		},
		MySQLOptions: models.MySQLOptions{},
	}
	pmmAgentVersion := version.MustParse("2.24.0")

	actual, err := mysqldExporterConfig(node, mysql, exporter, redactSecrets, pmmAgentVersion)
	expected := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--collect.auto_increment.columns",
			"--collect.binlog_size",
			"--collect.custom_query.hr.directory={{ .paths_base }}/collectors/custom-queries/mysql/high-resolution",
			"--collect.custom_query.lr",
			"--collect.custom_query.lr.directory={{ .paths_base }}/collectors/custom-queries/mysql/low-resolution",
			"--collect.custom_query.mr",
			"--collect.custom_query.mr.directory={{ .paths_base }}/collectors/custom-queries/mysql/medium-resolution",
			"--collect.engine_innodb_status",
			"--collect.engine_tokudb_status",
			"--collect.global_status",
			"--collect.global_variables",
			"--collect.info_schema.innodb_cmp",
			"--collect.info_schema.innodb_cmpmem",
			"--collect.info_schema.innodb_metrics",
			"--collect.info_schema.innodb_tablespaces",
			"--collect.info_schema.processlist",
			"--collect.info_schema.query_response_time",
			"--collect.info_schema.tables",
			"--collect.info_schema.tablestats",
			"--collect.info_schema.userstats",
			"--collect.perf_schema.eventswaits",
			"--collect.perf_schema.file_events",
			"--collect.perf_schema.file_instances",
			"--collect.perf_schema.indexiowaits",
			"--collect.perf_schema.tableiowaits",
			"--collect.perf_schema.tablelocks",
			"--collect.slave_status",
			"--collect.standard.go",
			"--collect.standard.process",
			"--exporter.conn-max-lifetime=55s",
			"--exporter.global-conn-pool",
			"--exporter.max-idle-conns=3",
			"--exporter.max-open-conns=3",
			"--web.listen-address=0.0.0.0:{{ .listen_port }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:3306)/?timeout=1s",
			"HTTP_AUTH=pmm:agent-id",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4."},
	}
	requireNoDuplicateFlags(t, actual.Args)
	require.NoError(t, err)
	require.Equal(t, expected.Args, actual.Args)
	require.Equal(t, expected.Env, actual.Env)
	require.Equal(t, expected, actual)
}

func TestMySQLdExporterConfigMySQL8Support(t *testing.T) {
	mysql := &models.Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(3306),
	}
	node := &models.Node{
		Address: "1.2.3.4",
	}
	exporter := &models.Agent{
		AgentID:         "agent-id",
		AgentType:       models.MySQLdExporterType,
		Username:        pointer.ToString("username"),
		Password:        pointer.ToString("s3cur3 p@$$w0r4."),
		AgentPassword:   pointer.ToString("agent-password"),
		ExporterOptions: models.ExporterOptions{},
	}
	pmmAgentVersion := version.MustParse("3.2.0")

	t.Run("SSLEnabled", func(t *testing.T) {
		exporter.MySQLOptions = models.MySQLOptions{
			TLSCa:   "content-of-tls-ca",
			TLSCert: "content-of-tls-certificate-key",
			TLSKey:  "content-of-tls-key",
		}

		actual, err := mysqldExporterConfig(node, mysql, exporter, redactSecrets, pmmAgentVersion)
		expected := &agentv1.SetStateRequest_AgentProcess{
			Type:               inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args: []string{
				"--collect.auto_increment.columns",
				"--collect.binlog_size",
				"--collect.custom_query.hr",
				"--collect.custom_query.hr.directory={{ .paths_base }}/collectors/custom-queries/mysql/high-resolution",
				"--collect.custom_query.lr",
				"--collect.custom_query.lr.directory={{ .paths_base }}/collectors/custom-queries/mysql/low-resolution",
				"--collect.custom_query.mr",
				"--collect.custom_query.mr.directory={{ .paths_base }}/collectors/custom-queries/mysql/medium-resolution",
				"--collect.engine_innodb_status",
				"--collect.engine_tokudb_status",
				"--collect.global_status",
				"--collect.global_variables",
				"--collect.heartbeat",
				"--collect.info_schema.clientstats",
				"--collect.info_schema.innodb_cmp",
				"--collect.info_schema.innodb_cmpmem",
				"--collect.info_schema.innodb_metrics",
				"--collect.info_schema.innodb_tablespaces",
				"--collect.info_schema.processlist",
				"--collect.info_schema.query_response_time",
				"--collect.info_schema.tables",
				"--collect.info_schema.tablestats",
				"--collect.info_schema.userstats",
				"--collect.perf_schema.eventsstatements",
				"--collect.perf_schema.eventswaits",
				"--collect.perf_schema.file_events",
				"--collect.perf_schema.file_instances",
				"--collect.perf_schema.indexiowaits",
				"--collect.perf_schema.tableiowaits",
				"--collect.perf_schema.tablelocks",
				"--collect.plugins",
				"--collect.slave_status",
				"--collect.standard.go",
				"--collect.standard.process",
				"--exporter.conn-max-lifetime=55s",
				"--exporter.max-idle-conns=3",
				"--exporter.max-open-conns=3",
				"--web.listen-address=0.0.0.0:{{ .listen_port }}",
				"--config.my-cnf={{ .TextFiles.myCnf }}",
				"--web.config.file={{ .TextFiles.webConfig }}",
			},
			RedactWords: []string{"s3cur3 p@$$w0r4.", "agent-password", "content-of-tls-key"},
			TextFiles: map[string]string{
				"myCnf":     "[client]\nhost=1.2.3.4\nport=3306\nuser=username\npassword=s3cur3 p@$$w0r4.\n\nssl-ca={{ .TextFiles.tlsCa }}\nssl-cert={{ .TextFiles.tlsCert }}\nssl-key={{ .TextFiles.tlsKey }}\n",
				"tlsCa":     "content-of-tls-ca",
				"tlsCert":   "content-of-tls-certificate-key",
				"tlsKey":    "content-of-tls-key",
				"webConfig": "basic_auth_users:\n    pmm: agent-password\n",
			},
		}
		requireNoDuplicateFlags(t, actual.Args)
		require.NoError(t, err)
		require.Equal(t, expected.Args, actual.Args)
		require.Equal(t, expected.Env, actual.Env)
		require.Equal(t, expected, actual)
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		exporter.Password = nil
		actual, err := mysqldExporterConfig(node, mysql, exporter, redactSecrets, pmmAgentVersion)
		expected := &agentv1.SetStateRequest_AgentProcess{
			Type:               inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args: []string{
				"--collect.auto_increment.columns",
				"--collect.binlog_size",
				"--collect.custom_query.hr",
				"--collect.custom_query.hr.directory={{ .paths_base }}/collectors/custom-queries/mysql/high-resolution",
				"--collect.custom_query.lr",
				"--collect.custom_query.lr.directory={{ .paths_base }}/collectors/custom-queries/mysql/low-resolution",
				"--collect.custom_query.mr",
				"--collect.custom_query.mr.directory={{ .paths_base }}/collectors/custom-queries/mysql/medium-resolution",
				"--collect.engine_innodb_status",
				"--collect.engine_tokudb_status",
				"--collect.global_status",
				"--collect.global_variables",
				"--collect.heartbeat",
				"--collect.info_schema.clientstats",
				"--collect.info_schema.innodb_cmp",
				"--collect.info_schema.innodb_cmpmem",
				"--collect.info_schema.innodb_metrics",
				"--collect.info_schema.innodb_tablespaces",
				"--collect.info_schema.processlist",
				"--collect.info_schema.query_response_time",
				"--collect.info_schema.tables",
				"--collect.info_schema.tablestats",
				"--collect.info_schema.userstats",
				"--collect.perf_schema.eventsstatements",
				"--collect.perf_schema.eventswaits",
				"--collect.perf_schema.file_events",
				"--collect.perf_schema.file_instances",
				"--collect.perf_schema.indexiowaits",
				"--collect.perf_schema.tableiowaits",
				"--collect.perf_schema.tablelocks",
				"--collect.plugins",
				"--collect.slave_status",
				"--collect.standard.go",
				"--collect.standard.process",
				"--exporter.conn-max-lifetime=55s",
				"--exporter.max-idle-conns=3",
				"--exporter.max-open-conns=3",
				"--web.listen-address=0.0.0.0:{{ .listen_port }}",
				"--config.my-cnf={{ .TextFiles.myCnf }}",
				"--web.config.file={{ .TextFiles.webConfig }}",
			},
			RedactWords: []string{"agent-password", "content-of-tls-key"},
			TextFiles: map[string]string{
				"myCnf":     "[client]\nhost=1.2.3.4\nport=3306\nuser=username\n\n\nssl-ca={{ .TextFiles.tlsCa }}\nssl-cert={{ .TextFiles.tlsCert }}\nssl-key={{ .TextFiles.tlsKey }}\n",
				"tlsCa":     "content-of-tls-ca",
				"tlsCert":   "content-of-tls-certificate-key",
				"tlsKey":    "content-of-tls-key",
				"webConfig": "basic_auth_users:\n    pmm: agent-password\n",
			},
		}
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("EmptyUsername", func(t *testing.T) {
		exporter.Username = nil
		exporter.Password = pointer.ToString("s3cur3 p@$$w0r4.")
		exporter.MySQLOptions = models.MySQLOptions{}
		actual, err := mysqldExporterConfig(node, mysql, exporter, exposeSecrets, pmmAgentVersion)
		expected := &agentv1.SetStateRequest_AgentProcess{
			Type:               inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args: []string{
				"--collect.auto_increment.columns",
				"--collect.binlog_size",
				"--collect.custom_query.hr",
				"--collect.custom_query.hr.directory={{ .paths_base }}/collectors/custom-queries/mysql/high-resolution",
				"--collect.custom_query.lr",
				"--collect.custom_query.lr.directory={{ .paths_base }}/collectors/custom-queries/mysql/low-resolution",
				"--collect.custom_query.mr",
				"--collect.custom_query.mr.directory={{ .paths_base }}/collectors/custom-queries/mysql/medium-resolution",
				"--collect.engine_innodb_status",
				"--collect.engine_tokudb_status",
				"--collect.global_status",
				"--collect.global_variables",
				"--collect.heartbeat",
				"--collect.info_schema.clientstats",
				"--collect.info_schema.innodb_cmp",
				"--collect.info_schema.innodb_cmpmem",
				"--collect.info_schema.innodb_metrics",
				"--collect.info_schema.innodb_tablespaces",
				"--collect.info_schema.processlist",
				"--collect.info_schema.query_response_time",
				"--collect.info_schema.tables",
				"--collect.info_schema.tablestats",
				"--collect.info_schema.userstats",
				"--collect.perf_schema.eventsstatements",
				"--collect.perf_schema.eventswaits",
				"--collect.perf_schema.file_events",
				"--collect.perf_schema.file_instances",
				"--collect.perf_schema.indexiowaits",
				"--collect.perf_schema.tableiowaits",
				"--collect.perf_schema.tablelocks",
				"--collect.plugins",
				"--collect.slave_status",
				"--collect.standard.go",
				"--collect.standard.process",
				"--exporter.conn-max-lifetime=55s",
				"--exporter.max-idle-conns=3",
				"--exporter.max-open-conns=3",
				"--web.listen-address=0.0.0.0:{{ .listen_port }}",
				"--config.my-cnf={{ .TextFiles.myCnf }}",
				"--web.config.file={{ .TextFiles.webConfig }}",
			},
			TextFiles: map[string]string{
				"myCnf":     "[client]\nhost=1.2.3.4\nport=3306\n\npassword=s3cur3 p@$$w0r4.\n\n\n\n\n",
				"webConfig": "basic_auth_users:\n    pmm: agent-password\n",
			},
		}

		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
