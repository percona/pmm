// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package prometheus

import (
	"context"
	"database/sql"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
)

var configPath = filepath.Join("..", "..", "testdata", "prometheus", "prometheus.yml")

func setup(t *testing.T) (*reform.DB, *Service, []byte) {
	t.Helper()

	sqlDB := testdb.Open(t, models.SkipFixtures)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	svc, err := NewService(configPath, "promtool", db, "http://127.0.0.1:9090/prometheus/")
	require.NoError(t, err)

	original, err := ioutil.ReadFile(configPath) //nolint:gosec
	require.NoError(t, err)

	require.NoError(t, svc.Check(context.Background()))

	return db, svc, original
}

func teardown(t *testing.T, db *reform.DB, svc *Service, original []byte) { //nolint:golint
	t.Helper()

	assert.NoError(t, ioutil.WriteFile(configPath, original, 0644))
	assert.NoError(t, svc.reload())

	assert.NoError(t, db.DBInterface().(*sql.DB).Close())
}

func TestPrometheus(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		db, svc, original := setup(t)
		defer teardown(t, db, svc, original)

		assert.NoError(t, svc.updateConfiguration())

		actual, err := ioutil.ReadFile(configPath) //nolint:gosec
		require.NoError(t, err)
		assert.Equal(t, string(original), string(actual))
	})

	t.Run("Normal", func(t *testing.T) {
		db, svc, original := setup(t)
		defer teardown(t, db, svc, original)

		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeType:     models.GenericNodeType,
				NodeName:     "test-generic-node",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_node_label": "foo"}`),
			},
			&models.Agent{
				AgentID:      "/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d"),
			},

			// listen port not known
			&models.Agent{
				AgentID:    "/agent_id/711674c2-36e6-42d5-8e63-5d7c84c9053a",
				AgentType:  models.NodeExporterType,
				PMMAgentID: pointer.ToString("/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853"),
				ListenPort: nil,
			},
			&models.AgentNode{
				AgentID: "/agent_id/711674c2-36e6-42d5-8e63-5d7c84c9053a",
				NodeID:  "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
			},

			&models.Service{
				ServiceID:    "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
				ServiceType:  models.MySQLServiceType,
				ServiceName:  "test-mysql",
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				CustomLabels: []byte(`{"_service_label": "bar"}`),
			},

			&models.Agent{
				AgentID:      "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   pointer.ToString("/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853"),
				CustomLabels: []byte(`{"_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			},
			&models.AgentService{
				AgentID:   "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				ServiceID: "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
			},

			&models.Service{
				ServiceID:    "/service_id/9cffbdd4-3cd2-47f8-a5f9-a749c3d5fee1",
				ServiceType:  models.PostgreSQLServiceType,
				ServiceName:  "test-postgresql",
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				CustomLabels: []byte(`{"_service_label": "bar"}`),
			},

			&models.Agent{
				AgentID:      "/agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac",
				AgentType:    models.PostgresExporterType,
				PMMAgentID:   pointer.ToString("/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853"),
				CustomLabels: []byte(`{"_agent_label": "postgres-baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			},
			&models.AgentService{
				AgentID:   "/agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac",
				ServiceID: "/service_id/9cffbdd4-3cd2-47f8-a5f9-a749c3d5fee1",
			},

			// disabled
			&models.Agent{
				AgentID:    "/agent_id/4226ddb5-8197-443c-9891-7772b38324a7",
				AgentType:  models.NodeExporterType,
				PMMAgentID: pointer.ToString("/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853"),
				Disabled:   true,
				ListenPort: pointer.ToUint16(12345),
			},
			&models.AgentNode{
				AgentID: "/agent_id/4226ddb5-8197-443c-9891-7772b38324a7",
				NodeID:  "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
			},
		} {
			require.NoError(t, db.Insert(str), "%+v", str)
		}

		assert.NoError(t, svc.updateConfiguration())

		expected := `# Managed by pmm-managed. DO NOT EDIT.
---
global:
  scrape_interval: 1m
  scrape_timeout: 10s
  evaluation_interval: 1m
rule_files:
- /etc/prometheus.d/*.rules.yml
scrape_configs:
- job_name: prometheus
  scrape_interval: 1s
  scrape_timeout: 1s
  metrics_path: /prometheus/metrics
  static_configs:
  - targets:
    - 127.0.0.1:9090
    labels:
      instance: pmm-server
- job_name: grafana
  scrape_interval: 5s
  scrape_timeout: 4s
  metrics_path: /metrics
  static_configs:
  - targets:
    - 127.0.0.1:3000
    labels:
      instance: pmm-server
- job_name: pmm-managed
  scrape_interval: 5s
  scrape_timeout: 4s
  metrics_path: /debug/metrics
  static_configs:
  - targets:
    - 127.0.0.1:7773
    labels:
      instance: pmm-server
- job_name: mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_1s
  params:
    collect[]:
    - global_status
    - info_schema.innodb_metrics
    - custom_query.hr
    - standard.process
    - standard.go
  scrape_interval: 1s
  scrape_timeout: 1s
  metrics_path: /metrics
  static_configs:
  - targets:
    - 1.2.3.4:12345
    labels:
      _agent_label: baz
      _node_label: foo
      _service_label: bar
      agent_id: /agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd
      agent_type: mysqld_exporter
      instance: /agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd
      node_id: /node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d
      node_name: test-generic-node
      node_type: generic
      service_id: /service_id/014647c3-b2f5-44eb-94f4-d943260a968c
      service_name: test-mysql
      service_type: mysql
- job_name: mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_5s
  params:
    collect[]:
    - engine_innodb_status
    - info_schema.innodb_cmp
    - info_schema.innodb_cmpmem
    - info_schema.processlist
    - info_schema.query_response_time
    - perf_schema.eventswaits
    - perf_schema.file_events
    - perf_schema.tablelocks
    - slave_status
    - custom_query.mr
  scrape_interval: 5s
  scrape_timeout: 4s
  metrics_path: /metrics
  static_configs:
  - targets:
    - 1.2.3.4:12345
    labels:
      _agent_label: baz
      _node_label: foo
      _service_label: bar
      agent_id: /agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd
      agent_type: mysqld_exporter
      instance: /agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd
      node_id: /node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d
      node_name: test-generic-node
      node_type: generic
      service_id: /service_id/014647c3-b2f5-44eb-94f4-d943260a968c
      service_name: test-mysql
      service_type: mysql
- job_name: mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_1m0s
  params:
    collect[]:
    - auto_increment.columns
    - binlog_size
    - engine_tokudb_status
    - global_variables
    - heartbeat
    - info_schema.clientstats
    - info_schema.innodb_tablespaces
    - info_schema.tables
    - info_schema.tablestats
    - info_schema.userstats
    - perf_schema.eventsstatements
    - perf_schema.file_instances
    - perf_schema.indexiowaits
    - perf_schema.tableiowaits
    - perf_schema.tablestats
    - custom_query.lr
  scrape_interval: 1m
  scrape_timeout: 10s
  metrics_path: /metrics
  static_configs:
  - targets:
    - 1.2.3.4:12345
    labels:
      _agent_label: baz
      _node_label: foo
      _service_label: bar
      agent_id: /agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd
      agent_type: mysqld_exporter
      instance: /agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd
      node_id: /node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d
      node_name: test-generic-node
      node_type: generic
      service_id: /service_id/014647c3-b2f5-44eb-94f4-d943260a968c
      service_name: test-mysql
      service_type: mysql
- job_name: postgres_exporter_agent_id_29e14468-d479-4b4d-bfb7-4ac2fb865bac_1s
  params:
    collect[]:
    - exporter
    - custom_query.hr
    - standard.process
    - standard.go
  scrape_interval: 1s
  scrape_timeout: 1s
  metrics_path: /metrics
  static_configs:
  - targets:
    - 1.2.3.4:12345
    labels:
      _agent_label: postgres-baz
      _node_label: foo
      _service_label: bar
      agent_id: /agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac
      agent_type: postgres_exporter
      instance: /agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac
      node_id: /node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d
      node_name: test-generic-node
      node_type: generic
      service_id: /service_id/9cffbdd4-3cd2-47f8-a5f9-a749c3d5fee1
      service_name: test-postgresql
      service_type: postgresql
- job_name: postgres_exporter_agent_id_29e14468-d479-4b4d-bfb7-4ac2fb865bac_5s
  params:
    collect[]:
    - custom_query.mr
  scrape_interval: 5s
  scrape_timeout: 4s
  metrics_path: /metrics
  static_configs:
  - targets:
    - 1.2.3.4:12345
    labels:
      _agent_label: postgres-baz
      _node_label: foo
      _service_label: bar
      agent_id: /agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac
      agent_type: postgres_exporter
      instance: /agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac
      node_id: /node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d
      node_name: test-generic-node
      node_type: generic
      service_id: /service_id/9cffbdd4-3cd2-47f8-a5f9-a749c3d5fee1
      service_name: test-postgresql
      service_type: postgresql
- job_name: postgres_exporter_agent_id_29e14468-d479-4b4d-bfb7-4ac2fb865bac_1m0s
  params:
    collect[]:
    - custom_query.lr
  scrape_interval: 1m
  scrape_timeout: 10s
  metrics_path: /metrics
  static_configs:
  - targets:
    - 1.2.3.4:12345
    labels:
      _agent_label: postgres-baz
      _node_label: foo
      _service_label: bar
      agent_id: /agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac
      agent_type: postgres_exporter
      instance: /agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac
      node_id: /node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d
      node_name: test-generic-node
      node_type: generic
      service_id: /service_id/9cffbdd4-3cd2-47f8-a5f9-a749c3d5fee1
      service_name: test-postgresql
      service_type: postgresql
`
		actual, err := ioutil.ReadFile(configPath) //nolint:gosec
		require.NoError(t, err)
		assert.Equal(t, expected, string(actual))
	})
}
