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

package victoriametrics

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

const configPath = "../../testdata/victoriametrics/promscrape.yml"

func setup(t *testing.T) (*reform.DB, *Service, []byte) {
	t.Helper()
	check := require.New(t)

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	vmParams := &models.VictoriaMetricsParams{BaseConfigPath: "/srv/prometheus/prometheus.base.yml"}
	svc, err := NewVictoriaMetrics(configPath, db, "http://127.0.0.1:9090/prometheus/", vmParams)
	check.NoError(err)

	original, err := os.ReadFile(configPath)
	check.NoError(err)

	check.NoError(svc.IsReady(context.Background()))

	return db, svc, original
}

func teardown(t *testing.T, db *reform.DB, svc *Service, original []byte) {
	t.Helper()
	check := assert.New(t)

	check.NoError(os.WriteFile(configPath, original, 0o600))
	check.NoError(svc.reload(context.Background()))

	check.NoError(db.DBInterface().(*sql.DB).Close())
}

func TestVictoriaMetrics(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		check := require.New(t)
		db, svc, original := setup(t)
		defer teardown(t, db, svc, original)

		check.NoError(svc.updateConfiguration(context.Background()))

		actual, err := os.ReadFile(configPath)
		check.NoError(err)
		check.Equal(string(original), string(actual), "actual:\n%s", actual)
	})

	t.Run("Normal", func(t *testing.T) {
		check := require.New(t)
		db, svc, original := setup(t)
		defer teardown(t, db, svc, original)
		err := models.SaveSettings(db.Querier, &models.Settings{})
		check.NoError(err)

		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeType:     models.GenericNodeType,
				NodeName:     "test-generic-node",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_node_label": "foo"}`),
			},
			&models.Node{
				NodeID:       "/node_id/4e2e07dc-40a1-18ca-aea9-d943260a9653",
				NodeType:     models.RemoteNodeType,
				NodeName:     "test-remote-node",
				Address:      "10.20.30.40",
				CustomLabels: []byte(`{"_node_label": "remote-foo"}`),
			},
			&models.Agent{
				AgentID:      "/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d"),
				Version:      pointer.ToString("2.26.0"),
			},

			// listen port not known
			&models.Agent{
				AgentID:    "/agent_id/711674c2-36e6-42d5-8e63-5d7c84c9053a",
				AgentType:  models.NodeExporterType,
				PMMAgentID: pointer.ToString("/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853"),
				NodeID:     pointer.ToString("/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d"),
				ListenPort: nil,
			},

			&models.Service{
				ServiceID:    "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
				ServiceType:  models.MySQLServiceType,
				ServiceName:  "test-mysql",
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				Port:         pointer.ToUint16(3306),
				CustomLabels: []byte(`{"_service_label": "bar"}`),
			},

			&models.Service{
				ServiceID:    "/service_id/4f1508fd-12c4-4ecf-b0a4-7ab19c996f61",
				ServiceType:  models.MySQLServiceType,
				ServiceName:  "test-remote-mysql",
				NodeID:       "/node_id/4e2e07dc-40a1-18ca-aea9-d943260a9653",
				Address:      pointer.ToString("50.60.70.80"),
				Port:         pointer.ToUint16(3306),
				CustomLabels: []byte(`{"_service_label": "bar"}`),
			},

			&models.Service{
				ServiceID:    "/service_id/acds89846-3cd2-47f8-a5f9-ac789513cde4",
				ServiceType:  models.MongoDBServiceType,
				ServiceName:  "test-mongodb",
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				Port:         pointer.ToUint16(27017),
				CustomLabels: []byte(`{"_service_label": "bam"}`),
			},

			&models.Agent{
				AgentID:        "/agent_id/ecd8995a-d479-4b4d-bfb7-865bac4ac2fb",
				AgentType:      models.MongoDBExporterType,
				PMMAgentID:     pointer.ToString("/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853"),
				ServiceID:      pointer.ToString("/service_id/acds89846-3cd2-47f8-a5f9-ac789513cde4"),
				CustomLabels:   []byte(`{"_agent_label": "mongodb-baz"}`),
				ListenPort:     pointer.ToUint16(12346),
				MongoDBOptions: &models.MongoDBOptions{EnableAllCollectors: true},
			},

			&models.Agent{
				AgentID:      "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   pointer.ToString("/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853"),
				ServiceID:    pointer.ToString("/service_id/014647c3-b2f5-44eb-94f4-d943260a968c"),
				CustomLabels: []byte(`{"_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			},

			&models.Agent{
				AgentID:      "/agent_id/f9ab9f7b-5e53-4952-a2e7-ff25fb90fe6a",
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   pointer.ToString("/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853"),
				ServiceID:    pointer.ToString("/service_id/4f1508fd-12c4-4ecf-b0a4-7ab19c996f61"),
				CustomLabels: []byte(`{"_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			},

			&models.Service{
				ServiceID:    "/service_id/9cffbdd4-3cd2-47f8-a5f9-a749c3d5fee1",
				ServiceType:  models.PostgreSQLServiceType,
				ServiceName:  "test-postgresql",
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				Port:         pointer.ToUint16(5432),
				CustomLabels: []byte(`{"_service_label": "bar"}`),
			},

			&models.Agent{
				AgentID:      "/agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac",
				AgentType:    models.PostgresExporterType,
				PMMAgentID:   pointer.ToString("/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853"),
				ServiceID:    pointer.ToString("/service_id/9cffbdd4-3cd2-47f8-a5f9-a749c3d5fee1"),
				CustomLabels: []byte(`{"_agent_label": "postgres-baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			},

			// disabled
			&models.Agent{
				AgentID:    "/agent_id/4226ddb5-8197-443c-9891-7772b38324a7",
				AgentType:  models.NodeExporterType,
				PMMAgentID: pointer.ToString("/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853"),
				NodeID:     pointer.ToString("/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d"),
				Disabled:   true,
				ListenPort: pointer.ToUint16(12345),
			},

			// PMM Agent without version
			&models.Agent{
				AgentID:      "/agent_id/892b3d86-12e5-4765-aa32-e5092ecd78e1",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d"),
			},

			&models.Service{
				ServiceID:    "/service_id/1eae647b-f1e2-4e15-bc58-dfdbc3c37cbf",
				ServiceType:  models.MongoDBServiceType,
				ServiceName:  "test-mongodb-noversion",
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.9"),
				Port:         pointer.ToUint16(27017),
				CustomLabels: []byte(`{"_service_label": "bam"}`),
			},

			// Agent with push model
			&models.Agent{
				AgentID:        "/agent_id/386c4ce6-7cd2-4bc9-9d6f-b4691c6e7eb7",
				AgentType:      models.MongoDBExporterType,
				PMMAgentID:     pointer.ToString("/agent_id/892b3d86-12e5-4765-aa32-e5092ecd78e1"),
				ServiceID:      pointer.ToString("/service_id/1eae647b-f1e2-4e15-bc58-dfdbc3c37cbf"),
				CustomLabels:   []byte(`{"_agent_label": "mongodb-baz-push"}`),
				ListenPort:     pointer.ToUint16(12346),
				MongoDBOptions: &models.MongoDBOptions{EnableAllCollectors: true},
				PushMetrics:    true,
			},

			// Agent with pull model
			&models.Agent{
				AgentID:        "/agent_id/cfec996c-4fe6-41d9-83cb-e1a3b1fe10a8",
				AgentType:      models.MongoDBExporterType,
				PMMAgentID:     pointer.ToString("/agent_id/892b3d86-12e5-4765-aa32-e5092ecd78e1"),
				ServiceID:      pointer.ToString("/service_id/1eae647b-f1e2-4e15-bc58-dfdbc3c37cbf"),
				CustomLabels:   []byte(`{"_agent_label": "mongodb-baz-pull"}`),
				ListenPort:     pointer.ToUint16(12346),
				MongoDBOptions: &models.MongoDBOptions{EnableAllCollectors: true},
				PushMetrics:    false,
			},
		} {
			check.NoError(db.Insert(str), "%+v", str)
		}

		check.NoError(svc.updateConfiguration(context.Background()))

		expected := strings.TrimSpace(`
# Managed by pmm-managed. DO NOT EDIT.
---
global:
    scrape_interval: 1m
    scrape_timeout: 54s
scrape_configs:
    - job_name: victoriametrics
      honor_timestamps: false
      scrape_interval: 5s
      scrape_timeout: 4500ms
      metrics_path: /prometheus/metrics
      static_configs:
        - targets:
            - 127.0.0.1:9090
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: vmalert
      honor_timestamps: false
      scrape_interval: 5s
      scrape_timeout: 4500ms
      metrics_path: /metrics
      static_configs:
        - targets:
            - 127.0.0.1:8880
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: alertmanager
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /alertmanager/metrics
      static_configs:
        - targets:
            - 127.0.0.1:9093
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: grafana
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /metrics
      static_configs:
        - targets:
            - 127.0.0.1:3000
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: pmm-managed
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /debug/metrics
      static_configs:
        - targets:
            - 127.0.0.1:7773
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: qan-api2
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /debug/metrics
      static_configs:
        - targets:
            - 127.0.0.1:9933
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: mongodb_exporter_agent_id_cfec996c-4fe6-41d9-83cb-e1a3b1fe10a8_hr
      honor_timestamps: false
      scrape_interval: 5s
      scrape_timeout: 4500ms
      metrics_path: /metrics
      static_configs:
        - targets:
            - 1.2.3.4:12346
          labels:
            _agent_label: mongodb-baz-pull
            _node_label: foo
            _service_label: bam
            agent_id: /agent_id/cfec996c-4fe6-41d9-83cb-e1a3b1fe10a8
            agent_type: mongodb_exporter
            instance: /agent_id/cfec996c-4fe6-41d9-83cb-e1a3b1fe10a8
            node_id: /node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d
            node_name: test-generic-node
            node_type: generic
            service_id: /service_id/1eae647b-f1e2-4e15-bc58-dfdbc3c37cbf
            service_name: test-mongodb-noversion
            service_type: mongodb
      basic_auth:
        username: pmm
        password: /agent_id/cfec996c-4fe6-41d9-83cb-e1a3b1fe10a8
      follow_redirects: false
    - job_name: mongodb_exporter_agent_id_ecd8995a-d479-4b4d-bfb7-865bac4ac2fb_hr
      honor_timestamps: false
      params:
        collect[]:
            - diagnosticdata
            - replicasetstatus
            - topmetrics
      scrape_interval: 5s
      scrape_timeout: 4500ms
      metrics_path: /metrics
      static_configs:
        - targets:
            - 1.2.3.4:12346
          labels:
            _agent_label: mongodb-baz
            _node_label: foo
            _service_label: bam
            agent_id: /agent_id/ecd8995a-d479-4b4d-bfb7-865bac4ac2fb
            agent_type: mongodb_exporter
            instance: /agent_id/ecd8995a-d479-4b4d-bfb7-865bac4ac2fb
            node_id: /node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d
            node_name: test-generic-node
            node_type: generic
            service_id: /service_id/acds89846-3cd2-47f8-a5f9-ac789513cde4
            service_name: test-mongodb
            service_type: mongodb
      basic_auth:
        username: pmm
        password: /agent_id/ecd8995a-d479-4b4d-bfb7-865bac4ac2fb
      follow_redirects: false
    - job_name: mongodb_exporter_agent_id_ecd8995a-d479-4b4d-bfb7-865bac4ac2fb_lr
      honor_timestamps: false
      params:
        collect[]:
            - collstats
            - dbstats
            - indexstats
      scrape_interval: 1m
      scrape_timeout: 54s
      metrics_path: /metrics
      static_configs:
        - targets:
            - 1.2.3.4:12346
          labels:
            _agent_label: mongodb-baz
            _node_label: foo
            _service_label: bam
            agent_id: /agent_id/ecd8995a-d479-4b4d-bfb7-865bac4ac2fb
            agent_type: mongodb_exporter
            instance: /agent_id/ecd8995a-d479-4b4d-bfb7-865bac4ac2fb
            node_id: /node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d
            node_name: test-generic-node
            node_type: generic
            service_id: /service_id/acds89846-3cd2-47f8-a5f9-ac789513cde4
            service_name: test-mongodb
            service_type: mongodb
      basic_auth:
        username: pmm
        password: /agent_id/ecd8995a-d479-4b4d-bfb7-865bac4ac2fb
      follow_redirects: false
    - job_name: mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_hr
      honor_timestamps: false
      params:
        collect[]:
            - custom_query.hr
            - global_status
            - info_schema.innodb_metrics
            - standard.go
            - standard.process
      scrape_interval: 5s
      scrape_timeout: 4500ms
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
      basic_auth:
        username: pmm
        password: /agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd
      follow_redirects: false
    - job_name: mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_mr
      honor_timestamps: false
      params:
        collect[]:
            - custom_query.mr
            - engine_innodb_status
            - info_schema.innodb_cmp
            - info_schema.innodb_cmpmem
            - info_schema.processlist
            - info_schema.query_response_time
            - perf_schema.eventswaits
            - perf_schema.file_events
            - perf_schema.tablelocks
            - slave_status
      scrape_interval: 10s
      scrape_timeout: 9s
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
      basic_auth:
        username: pmm
        password: /agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd
      follow_redirects: false
    - job_name: mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_lr
      honor_timestamps: false
      params:
        collect[]:
            - auto_increment.columns
            - binlog_size
            - custom_query.lr
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
      scrape_interval: 1m
      scrape_timeout: 54s
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
      basic_auth:
        username: pmm
        password: /agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd
      follow_redirects: false
    - job_name: mysqld_exporter_agent_id_f9ab9f7b-5e53-4952-a2e7-ff25fb90fe6a_hr
      honor_timestamps: false
      params:
        collect[]:
            - custom_query.hr
            - global_status
            - info_schema.innodb_metrics
            - standard.go
            - standard.process
      scrape_interval: 5s
      scrape_timeout: 4500ms
      metrics_path: /metrics
      static_configs:
        - targets:
            - 1.2.3.4:12345
          labels:
            _agent_label: baz
            _node_label: remote-foo
            _service_label: bar
            agent_id: /agent_id/f9ab9f7b-5e53-4952-a2e7-ff25fb90fe6a
            agent_type: mysqld_exporter
            instance: /agent_id/f9ab9f7b-5e53-4952-a2e7-ff25fb90fe6a
            node_id: /node_id/4e2e07dc-40a1-18ca-aea9-d943260a9653
            node_name: test-remote-node
            node_type: remote
            service_id: /service_id/4f1508fd-12c4-4ecf-b0a4-7ab19c996f61
            service_name: test-remote-mysql
            service_type: mysql
      basic_auth:
        username: pmm
        password: /agent_id/f9ab9f7b-5e53-4952-a2e7-ff25fb90fe6a
      follow_redirects: false
    - job_name: mysqld_exporter_agent_id_f9ab9f7b-5e53-4952-a2e7-ff25fb90fe6a_mr
      honor_timestamps: false
      params:
        collect[]:
            - custom_query.mr
            - engine_innodb_status
            - info_schema.innodb_cmp
            - info_schema.innodb_cmpmem
            - info_schema.processlist
            - info_schema.query_response_time
            - perf_schema.eventswaits
            - perf_schema.file_events
            - perf_schema.tablelocks
            - slave_status
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /metrics
      static_configs:
        - targets:
            - 1.2.3.4:12345
          labels:
            _agent_label: baz
            _node_label: remote-foo
            _service_label: bar
            agent_id: /agent_id/f9ab9f7b-5e53-4952-a2e7-ff25fb90fe6a
            agent_type: mysqld_exporter
            instance: /agent_id/f9ab9f7b-5e53-4952-a2e7-ff25fb90fe6a
            node_id: /node_id/4e2e07dc-40a1-18ca-aea9-d943260a9653
            node_name: test-remote-node
            node_type: remote
            service_id: /service_id/4f1508fd-12c4-4ecf-b0a4-7ab19c996f61
            service_name: test-remote-mysql
            service_type: mysql
      basic_auth:
        username: pmm
        password: /agent_id/f9ab9f7b-5e53-4952-a2e7-ff25fb90fe6a
      follow_redirects: false
    - job_name: mysqld_exporter_agent_id_f9ab9f7b-5e53-4952-a2e7-ff25fb90fe6a_lr
      honor_timestamps: false
      params:
        collect[]:
            - auto_increment.columns
            - binlog_size
            - custom_query.lr
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
      scrape_interval: 1m
      scrape_timeout: 54s
      metrics_path: /metrics
      static_configs:
        - targets:
            - 1.2.3.4:12345
          labels:
            _agent_label: baz
            _node_label: remote-foo
            _service_label: bar
            agent_id: /agent_id/f9ab9f7b-5e53-4952-a2e7-ff25fb90fe6a
            agent_type: mysqld_exporter
            instance: /agent_id/f9ab9f7b-5e53-4952-a2e7-ff25fb90fe6a
            node_id: /node_id/4e2e07dc-40a1-18ca-aea9-d943260a9653
            node_name: test-remote-node
            node_type: remote
            service_id: /service_id/4f1508fd-12c4-4ecf-b0a4-7ab19c996f61
            service_name: test-remote-mysql
            service_type: mysql
      basic_auth:
        username: pmm
        password: /agent_id/f9ab9f7b-5e53-4952-a2e7-ff25fb90fe6a
      follow_redirects: false
    - job_name: postgres_exporter_agent_id_29e14468-d479-4b4d-bfb7-4ac2fb865bac_hr
      honor_timestamps: false
      params:
        collect[]:
            - custom_query.hr
            - exporter
            - standard.go
            - standard.process
      scrape_interval: 5s
      scrape_timeout: 4500ms
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
      basic_auth:
        username: pmm
        password: /agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac
      follow_redirects: false
      stream_parse: true
    - job_name: postgres_exporter_agent_id_29e14468-d479-4b4d-bfb7-4ac2fb865bac_mr
      honor_timestamps: false
      params:
        collect[]:
            - custom_query.mr
      scrape_interval: 10s
      scrape_timeout: 9s
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
      basic_auth:
        username: pmm
        password: /agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac
      follow_redirects: false
      stream_parse: true
    - job_name: postgres_exporter_agent_id_29e14468-d479-4b4d-bfb7-4ac2fb865bac_lr
      honor_timestamps: false
      params:
        collect[]:
            - custom_query.lr
      scrape_interval: 1m
      scrape_timeout: 54s
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
      basic_auth:
        username: pmm
        password: /agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac
      follow_redirects: false
      stream_parse: true
`) + "\n"
		actual, err := os.ReadFile(configPath)
		check.NoError(err)
		check.Equal(expected, string(actual), "actual:\n%s", actual)
	})
}

func TestConfigReload(t *testing.T) {
	db, svc, original := setup(t)
	defer teardown(t, db, svc, original)
	t.Run("Good file, reload ok", func(t *testing.T) {
		err := svc.configAndReload(context.TODO(), []byte(strings.TrimSpace(`
# Managed by pmm-managed. DO NOT EDIT.
---
global:
  scrape_interval: 1m
  scrape_timeout: 54s
scrape_configs:
- job_name: victoriametrics
  honor_timestamps: false
  scrape_interval: 5s
  scrape_timeout: 4500ms
  metrics_path: /prometheus/metrics
  static_configs:
  - targets:
    - 127.0.0.1:9090
    labels:
      instance: pmm-server
  follow_redirects: false
- job_name: vmalert
  honor_timestamps: false
  scrape_interval: 5s
  scrape_timeout: 4500ms
  metrics_path: /metrics
  static_configs:
  - targets:
    - 127.0.0.1:8880
    labels:
      instance: pmm-server
  follow_redirects: false
- job_name: alertmanager
  honor_timestamps: false
  scrape_interval: 10s
  scrape_timeout: 9s
  metrics_path: /alertmanager/metrics
  static_configs:
  - targets:
    - 127.0.0.1:9093
    labels:
      instance: pmm-server
  follow_redirects: false`)))
		assert.NoError(t, err)
	})

	t.Run("Bad scrape config file", func(t *testing.T) {
		err := svc.configAndReload(context.TODO(), []byte(`unexpected input`))
		assert.Errorf(t, err, "error when checking Prometheus config")
	})

	t.Run("Scrape config file with unknown params", func(t *testing.T) {
		err := svc.configAndReload(context.TODO(), []byte(strings.TrimSpace(`
# Managed by pmm-managed. DO NOT EDIT.
---
global:
  scrape_interval: 1m
  scrape_timeout: 54s
unknown_filed: unknown_value

`)))
		tests.AssertGRPCError(t, status.New(codes.Aborted,
			"yaml: unmarshal errors:\n  line 6: field unknown_filed not found in type promscrape.Config;"+
				" pass -promscrape.config.strictParse=false command-line flag for ignoring unknown fields in yaml config\n",
		), err)
	})
}

func TestBaseConfig(t *testing.T) {
	db, svc, original := setup(t)
	defer teardown(t, db, svc, original)

	svc.baseConfigPath = "../../testdata/victoriametrics/promscrape.base.yml"

	expected := strings.TrimSpace(`
# Managed by pmm-managed. DO NOT EDIT.
---
global:
    scrape_interval: 9m
    scrape_timeout: 19s
scrape_configs:
    - job_name: external-service
      honor_timestamps: true
      scrape_interval: 5s
      scrape_timeout: 4500ms
      metrics_path: /metrics
      scheme: http
      static_configs:
        - targets:
            - 127.0.0.1:1234
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: victoriametrics
      honor_timestamps: false
      scrape_interval: 5s
      scrape_timeout: 4500ms
      metrics_path: /prometheus/metrics
      static_configs:
        - targets:
            - 127.0.0.1:9090
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: vmalert
      honor_timestamps: false
      scrape_interval: 5s
      scrape_timeout: 4500ms
      metrics_path: /metrics
      static_configs:
        - targets:
            - 127.0.0.1:8880
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: alertmanager
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /alertmanager/metrics
      static_configs:
        - targets:
            - 127.0.0.1:9093
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: grafana
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /metrics
      static_configs:
        - targets:
            - 127.0.0.1:3000
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: pmm-managed
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /debug/metrics
      static_configs:
        - targets:
            - 127.0.0.1:7773
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: qan-api2
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /debug/metrics
      static_configs:
        - targets:
            - 127.0.0.1:9933
          labels:
            instance: pmm-server
      follow_redirects: false
`) + "\n"
	newcfg, err := svc.marshalConfig(svc.loadBaseConfig())
	assert.NoError(t, err)
	assert.Equal(t, expected, string(newcfg), "actual:\n%s", newcfg)
}

func TestPMMAgent(t *testing.T) {
	t.Run("PullMode", func(t *testing.T) {
		check := require.New(t)
		db, svc, original := setup(t)
		defer teardown(t, db, svc, original)
		err := models.SaveSettings(db.Querier, &models.Settings{})
		check.NoError(err)

		for _, str := range []reform.Struct{
			// for this case we don't need full server setup, just to test pmm-agent config
			&models.Node{
				NodeID:   "pmm-server",
				NodeType: models.GenericNodeType,
				NodeName: "pmm-server",
				Address:  "127.0.0.1",
			},
			&models.Service{
				ServiceID:    "/service_id/171e31f9-e9b0-496f-bb4f-b522d4f7c4fa",
				ServiceType:  models.PostgreSQLServiceType,
				ServiceName:  "pmm-server-postgresql",
				NodeID:       "pmm-server",
				Address:      pointer.ToString("127.0.0.1"),
				Port:         pointer.ToUint16(5432),
				DatabaseName: "postgres",
			},
			&models.Agent{
				AgentID:      "pmm-server",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("pmm-server"),
				Version:      pointer.ToString("2.28.0"),
				ListenPort:   pointer.ToUint16(7777),
				PushMetrics:  false,
			},

			// other container with pmm-agent having pull metrics-mode
			&models.Node{
				NodeID:   "/node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d",
				NodeType: models.ContainerNodeType,
				NodeName: "306ccd68606f",
				Address:  "172.17.0.2",
				Distro:   "linux",
			},
			&models.Agent{
				AgentID:      "/agent_id/8dddcd70-9ee5-4513-8120-164d1ffec743",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("/node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d"),
				Version:      pointer.ToString("2.28.0"),
				ListenPort:   pointer.ToUint16(7777),
				PushMetrics:  false,
			},
			&models.Agent{
				AgentID:     "/agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3",
				AgentType:   models.NodeExporterType,
				PMMAgentID:  pointer.ToString("/agent_id/8dddcd70-9ee5-4513-8120-164d1ffec743"),
				NodeID:      pointer.ToString("/node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d"),
				ListenPort:  pointer.ToUint16(42000),
				PushMetrics: true,
			},
			&models.Agent{
				AgentID:     "/agent_id/e9117eb5-e2df-41c4-a7e1-7fa6fb9eadc8",
				AgentType:   models.VMAgentType,
				PMMAgentID:  pointer.ToString("/agent_id/8dddcd70-9ee5-4513-8120-164d1ffec743"),
				NodeID:      pointer.ToString("/node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d"),
				ListenPort:  pointer.ToUint16(42001),
				PushMetrics: true,
			},
		} {
			check.NoError(db.Insert(str), "%+v", str)
		}

		expected := strings.TrimSpace(`
# Managed by pmm-managed. DO NOT EDIT.
---
global:
    scrape_interval: 1m
    scrape_timeout: 54s
scrape_configs:
    - job_name: victoriametrics
      honor_timestamps: false
      scrape_interval: 5s
      scrape_timeout: 4500ms
      metrics_path: /prometheus/metrics
      static_configs:
        - targets:
            - 127.0.0.1:9090
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: vmalert
      honor_timestamps: false
      scrape_interval: 5s
      scrape_timeout: 4500ms
      metrics_path: /metrics
      static_configs:
        - targets:
            - 127.0.0.1:8880
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: alertmanager
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /alertmanager/metrics
      static_configs:
        - targets:
            - 127.0.0.1:9093
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: grafana
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /metrics
      static_configs:
        - targets:
            - 127.0.0.1:3000
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: pmm-managed
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /debug/metrics
      static_configs:
        - targets:
            - 127.0.0.1:7773
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: qan-api2
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /debug/metrics
      static_configs:
        - targets:
            - 127.0.0.1:9933
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: pmm-agent_agent_id_8dddcd70-9ee5-4513-8120-164d1ffec743_mr
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /debug/metrics
      static_configs:
        - targets:
            - 172.17.0.2:7777
          labels:
            agent_id: /agent_id/8dddcd70-9ee5-4513-8120-164d1ffec743
            agent_type: pmm-agent
            instance: /agent_id/8dddcd70-9ee5-4513-8120-164d1ffec743
      basic_auth:
        username: pmm
        password: /agent_id/8dddcd70-9ee5-4513-8120-164d1ffec743
      follow_redirects: false
    - job_name: pmm-agent_pmm-server_mr
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /debug/metrics
      static_configs:
        - targets:
            - 127.0.0.1:7777
          labels:
            agent_id: pmm-server
            agent_type: pmm-agent
            instance: pmm-server
      basic_auth:
        username: pmm
        password: pmm-server
      follow_redirects: false
`) + "\n"
		serverConf, err := svc.marshalConfig(svc.loadBaseConfig())
		check.NoError(err)
		check.Equal(expected, string(serverConf), "actual:\n%s", serverConf)

		expected = strings.TrimSpace(`
global: {}
scrape_configs:
    - job_name: node_exporter_agent_id_b637c3dc-ef3b-46ff-88e3-4771af5403f3_hr
      honor_timestamps: false
      params:
        collect[]:
            - buddyinfo
            - cpu
            - diskstats
            - filefd
            - filesystem
            - loadavg
            - meminfo
            - meminfo_numa
            - netdev
            - netstat
            - processes
            - standard.go
            - standard.process
            - stat
            - textfile.hr
            - time
            - vmstat
      scrape_interval: 5s
      scrape_timeout: 4500ms
      metrics_path: /metrics
      static_configs:
        - targets:
            - 127.0.0.1:42000
          labels:
            agent_id: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
            agent_type: node_exporter
            instance: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
            node_id: /node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d
            node_name: 306ccd68606f
            node_type: container
      basic_auth:
        username: pmm
        password: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
      follow_redirects: false
    - job_name: node_exporter_agent_id_b637c3dc-ef3b-46ff-88e3-4771af5403f3_mr
      honor_timestamps: false
      params:
        collect[]:
            - hwmon
            - textfile.mr
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /metrics
      static_configs:
        - targets:
            - 127.0.0.1:42000
          labels:
            agent_id: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
            agent_type: node_exporter
            instance: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
            node_id: /node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d
            node_name: 306ccd68606f
            node_type: container
      basic_auth:
        username: pmm
        password: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
      follow_redirects: false
    - job_name: node_exporter_agent_id_b637c3dc-ef3b-46ff-88e3-4771af5403f3_lr
      honor_timestamps: false
      params:
        collect[]:
            - bonding
            - entropy
            - textfile.lr
            - uname
      scrape_interval: 1m
      scrape_timeout: 54s
      metrics_path: /metrics
      static_configs:
        - targets:
            - 127.0.0.1:42000
          labels:
            agent_id: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
            agent_type: node_exporter
            instance: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
            node_id: /node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d
            node_name: 306ccd68606f
            node_type: container
      basic_auth:
        username: pmm
        password: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
      follow_redirects: false
    - job_name: vmagent_agent_id_e9117eb5-e2df-41c4-a7e1-7fa6fb9eadc8_mr
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      static_configs:
        - targets:
            - 127.0.0.1:42001
          labels:
            agent_id: /agent_id/e9117eb5-e2df-41c4-a7e1-7fa6fb9eadc8
            agent_type: vmagent
            instance: /agent_id/e9117eb5-e2df-41c4-a7e1-7fa6fb9eadc8
            node_id: /node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d
            node_name: 306ccd68606f
            node_type: container
      follow_redirects: false
`) + "\n"
		agentConf, err := svc.BuildScrapeConfigForVMAgent("/agent_id/8dddcd70-9ee5-4513-8120-164d1ffec743")
		check.NoError(err)
		check.Equal(expected, string(agentConf), "actual:\n%s", agentConf)
	})

	t.Run("PushMode", func(t *testing.T) {
		check := require.New(t)
		db, svc, original := setup(t)
		defer teardown(t, db, svc, original)
		err := models.SaveSettings(db.Querier, &models.Settings{})
		check.NoError(err)

		for _, str := range []reform.Struct{
			// for this case we don't need full server setup, just to test pmm-agent config
			&models.Node{
				NodeID:   "pmm-server",
				NodeType: models.GenericNodeType,
				NodeName: "pmm-server",
				Address:  "127.0.0.1",
			},
			&models.Service{
				ServiceID:    "/service_id/171e31f9-e9b0-496f-bb4f-b522d4f7c4fa",
				ServiceType:  models.PostgreSQLServiceType,
				ServiceName:  "pmm-server-postgresql",
				NodeID:       "pmm-server",
				Address:      pointer.ToString("127.0.0.1"),
				Port:         pointer.ToUint16(5432),
				DatabaseName: "postgres",
			},
			&models.Agent{
				AgentID:      "pmm-server",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("pmm-server"),
				Version:      pointer.ToString("2.28.0"),
				ListenPort:   pointer.ToUint16(7777),
				PushMetrics:  false,
			},

			// other container with pmm-agent having push metrics-mode
			&models.Node{
				NodeID:   "/node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d",
				NodeType: models.ContainerNodeType,
				NodeName: "306ccd68606f",
				Address:  "172.17.0.2",
				Distro:   "linux",
			},
			&models.Agent{
				AgentID:      "/agent_id/8dddcd70-9ee5-4513-8120-164d1ffec743",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("/node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d"),
				Version:      pointer.ToString("2.28.0"),
				ListenPort:   pointer.ToUint16(7777),
				PushMetrics:  true,
			},
			&models.Agent{
				AgentID:     "/agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3",
				AgentType:   models.NodeExporterType,
				PMMAgentID:  pointer.ToString("/agent_id/8dddcd70-9ee5-4513-8120-164d1ffec743"),
				NodeID:      pointer.ToString("/node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d"),
				ListenPort:  pointer.ToUint16(42000),
				PushMetrics: true,
			},
			&models.Agent{
				AgentID:     "/agent_id/e9117eb5-e2df-41c4-a7e1-7fa6fb9eadc8",
				AgentType:   models.VMAgentType,
				PMMAgentID:  pointer.ToString("/agent_id/8dddcd70-9ee5-4513-8120-164d1ffec743"),
				NodeID:      pointer.ToString("/node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d"),
				ListenPort:  pointer.ToUint16(42001),
				PushMetrics: true,
			},
		} {
			check.NoError(db.Insert(str), "%+v", str)
		}

		expected := strings.TrimSpace(`
# Managed by pmm-managed. DO NOT EDIT.
---
global:
    scrape_interval: 1m
    scrape_timeout: 54s
scrape_configs:
    - job_name: victoriametrics
      honor_timestamps: false
      scrape_interval: 5s
      scrape_timeout: 4500ms
      metrics_path: /prometheus/metrics
      static_configs:
        - targets:
            - 127.0.0.1:9090
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: vmalert
      honor_timestamps: false
      scrape_interval: 5s
      scrape_timeout: 4500ms
      metrics_path: /metrics
      static_configs:
        - targets:
            - 127.0.0.1:8880
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: alertmanager
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /alertmanager/metrics
      static_configs:
        - targets:
            - 127.0.0.1:9093
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: grafana
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /metrics
      static_configs:
        - targets:
            - 127.0.0.1:3000
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: pmm-managed
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /debug/metrics
      static_configs:
        - targets:
            - 127.0.0.1:7773
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: qan-api2
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /debug/metrics
      static_configs:
        - targets:
            - 127.0.0.1:9933
          labels:
            instance: pmm-server
      follow_redirects: false
    - job_name: pmm-agent_pmm-server_mr
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /debug/metrics
      static_configs:
        - targets:
            - 127.0.0.1:7777
          labels:
            agent_id: pmm-server
            agent_type: pmm-agent
            instance: pmm-server
      basic_auth:
        username: pmm
        password: pmm-server
      follow_redirects: false
`) + "\n"
		serverConf, err := svc.marshalConfig(svc.loadBaseConfig())
		check.NoError(err)
		check.Equal(expected, string(serverConf), "actual:\n%s", serverConf)

		expected = strings.TrimSpace(`
global: {}
scrape_configs:
    - job_name: node_exporter_agent_id_b637c3dc-ef3b-46ff-88e3-4771af5403f3_hr
      honor_timestamps: false
      params:
        collect[]:
            - buddyinfo
            - cpu
            - diskstats
            - filefd
            - filesystem
            - loadavg
            - meminfo
            - meminfo_numa
            - netdev
            - netstat
            - processes
            - standard.go
            - standard.process
            - stat
            - textfile.hr
            - time
            - vmstat
      scrape_interval: 5s
      scrape_timeout: 4500ms
      metrics_path: /metrics
      static_configs:
        - targets:
            - 127.0.0.1:42000
          labels:
            agent_id: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
            agent_type: node_exporter
            instance: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
            node_id: /node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d
            node_name: 306ccd68606f
            node_type: container
      basic_auth:
        username: pmm
        password: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
      follow_redirects: false
    - job_name: node_exporter_agent_id_b637c3dc-ef3b-46ff-88e3-4771af5403f3_mr
      honor_timestamps: false
      params:
        collect[]:
            - hwmon
            - textfile.mr
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /metrics
      static_configs:
        - targets:
            - 127.0.0.1:42000
          labels:
            agent_id: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
            agent_type: node_exporter
            instance: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
            node_id: /node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d
            node_name: 306ccd68606f
            node_type: container
      basic_auth:
        username: pmm
        password: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
      follow_redirects: false
    - job_name: node_exporter_agent_id_b637c3dc-ef3b-46ff-88e3-4771af5403f3_lr
      honor_timestamps: false
      params:
        collect[]:
            - bonding
            - entropy
            - textfile.lr
            - uname
      scrape_interval: 1m
      scrape_timeout: 54s
      metrics_path: /metrics
      static_configs:
        - targets:
            - 127.0.0.1:42000
          labels:
            agent_id: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
            agent_type: node_exporter
            instance: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
            node_id: /node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d
            node_name: 306ccd68606f
            node_type: container
      basic_auth:
        username: pmm
        password: /agent_id/b637c3dc-ef3b-46ff-88e3-4771af5403f3
      follow_redirects: false
    - job_name: pmm-agent_agent_id_8dddcd70-9ee5-4513-8120-164d1ffec743_mr
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      metrics_path: /debug/metrics
      static_configs:
        - targets:
            - 127.0.0.1:7777
          labels:
            agent_id: /agent_id/8dddcd70-9ee5-4513-8120-164d1ffec743
            agent_type: pmm-agent
            instance: /agent_id/8dddcd70-9ee5-4513-8120-164d1ffec743
      basic_auth:
        username: pmm
        password: /agent_id/8dddcd70-9ee5-4513-8120-164d1ffec743
      follow_redirects: false
    - job_name: vmagent_agent_id_e9117eb5-e2df-41c4-a7e1-7fa6fb9eadc8_mr
      honor_timestamps: false
      scrape_interval: 10s
      scrape_timeout: 9s
      static_configs:
        - targets:
            - 127.0.0.1:42001
          labels:
            agent_id: /agent_id/e9117eb5-e2df-41c4-a7e1-7fa6fb9eadc8
            agent_type: vmagent
            instance: /agent_id/e9117eb5-e2df-41c4-a7e1-7fa6fb9eadc8
            node_id: /node_id/bdac8d61-9bd0-45dc-a42d-27cf3bdb5e6d
            node_name: 306ccd68606f
            node_type: container
      follow_redirects: false 
`) + "\n"
		agentConf, err := svc.BuildScrapeConfigForVMAgent("/agent_id/8dddcd70-9ee5-4513-8120-164d1ffec743")
		check.NoError(err)
		check.Equal(expected, string(agentConf), "actual:\n%s", agentConf)
	})
}
