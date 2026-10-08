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

package management

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/management/v1/json/client"
)

// updateTestResponse is a server response describing a MongoDB Service before and after an update.
const updateTestResponse = `{
	"before": {
		"service_id": "1b6b5e9c-1c3a-4e5f-9a3a-1c3a4e5f9a3a", "service_name": "mongodb_srv_1", "environment": "prod",
		"agents": [
			{"agent_type": "mongodb_exporter", "username": "admin", "is_password_set": true, "push_metrics": true,
			 "log_level": "LOG_LEVEL_WARN", "mongo_db_options": {"enable_all_collectors": true, "collections_limit": -1}},
			{"agent_type": "qan-mongodb-profiler-agent", "max_query_length": 2048}
		]
	},
	"after": {
		"service_id": "1b6b5e9c-1c3a-4e5f-9a3a-1c3a4e5f9a3a", "service_name": "mongodb_srv_1", "environment": "prod",
		"agents": [
			{"agent_type": "mongodb_exporter", "username": "admin", "is_password_set": true, "push_metrics": true,
			 "log_level": "LOG_LEVEL_WARN", "disabled_collectors": ["collstats"],
			 "mongo_db_options": {"enable_all_collectors": true, "collections_limit": 0}},
			{"agent_type": "qan-mongodb-profiler-agent", "max_query_length": 2048}
		]
	}
}`

// updateRun is the request an update command sent and the result it returned.
type updateRun struct {
	path string
	body string
	res  commands.Result
}

// runUpdate parses the arguments like pmm-admin does and runs the command against a fake server.
func runUpdate(t *testing.T, cmd any, args ...string) (*updateRun, error) {
	t.Helper()

	var path, body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		path = r.URL.Path

		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		body = string(b)

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(updateTestResponse))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	originalClient := client.Default
	client.Default = client.New(httptransport.New(serverURL.Host, serverURL.Path, []string{serverURL.Scheme}), nil)
	t.Cleanup(func() { client.Default = originalClient })

	parser, err := kong.New(cmd, kong.Vars{
		"hostname":                     "host",
		"metricsModesEnum":             "auto, push, pull",
		"mongoDbQuerySourcesEnum":      "profiler, mongolog, none",
		"mongoDbQuerySourceDefault":    "profiler",
		"mysqlQuerySourcesEnum":        "slowlog, perfschema, none",
		"mysqlQuerySourceDefault":      "slowlog",
		"externalDefaultServiceName":   "-external",
		"externalDefaultGroupExporter": "external",
	})
	require.NoError(t, err)

	ctx, err := parser.Parse(args)
	require.NoError(t, err)
	cmd.(interface{ SetTypedFlags(map[string]bool) }).SetTypedFlags(commands.TypedFlags(ctx))

	res, err := cmd.(interface {
		RunCmd() (commands.Result, error)
	}).RunCmd()

	return &updateRun{path: path, body: body, res: res}, err
}

func TestUpdateMongoDB(t *testing.T) {
	t.Run("TicketExample", func(t *testing.T) {
		run, err := runUpdate(t, &UpdateMongoDBCommand{},
			"mongodb_srv_1", "--enable-all-collectors", "--max-collections-limit=0", "--disable-collectors=collstats")
		require.NoError(t, err)

		assert.Equal(t, "/v1/management/services/mongodb_srv_1", run.path)
		assert.JSONEq(t, `{"mongodb": {
			"enable_all_collectors": true,
			"collections_limit": 0,
			"disable_collectors": {"values": ["collstats"]}
		}}`, run.body)

		result := run.res.(*updateServiceResult)
		assert.True(t, result.Changed)
		assert.Equal(t, []settingChange{
			{Flag: "--disable-collectors", Before: "", After: "collstats"},
			{Flag: "--max-collections-limit", Before: "-1", After: "0"},
		}, result.Changes)

		out := result.String()
		assert.Contains(t, out, "MongoDB Service updated.")
		assert.Contains(t, out, `--max-collections-limit: "-1" -> "0"`)
		assert.Contains(t, out, "--password=***")
		assert.Contains(t, out, "--query-source=profiler")
	})

	t.Run("OnlyTypedFlagsAreSent", func(t *testing.T) {
		run, err := runUpdate(t, &UpdateMongoDBCommand{}, "mongodb_srv_1", "--log-level=debug")
		require.NoError(t, err)

		assert.JSONEq(t, `{"mongodb": {"log_level": "LOG_LEVEL_DEBUG"}}`, run.body)
	})

	t.Run("TurnsOffAndClears", func(t *testing.T) {
		run, err := runUpdate(t, &UpdateMongoDBCommand{},
			"mongodb_srv_1", "--no-enable-all-collectors", "--tls=false", "--disable-collectors=", "--custom-labels=", "--environment=")
		require.NoError(t, err)

		assert.JSONEq(t, `{"mongodb": {
			"enable_all_collectors": false,
			"tls": false,
			"disable_collectors": {"values": []},
			"custom_labels": {},
			"environment": ""
		}}`, run.body)
	})

	t.Run("AutoMetricsModeAndQuerySource", func(t *testing.T) {
		run, err := runUpdate(t, &UpdateMongoDBCommand{},
			"--service-id=1b6b5e9c-1c3a-4e5f-9a3a-1c3a4e5f9a3a", "--metrics-mode=auto", "--query-source=mongolog")
		require.NoError(t, err)

		assert.JSONEq(t, `{"mongodb": {
			"metrics_mode": "METRICS_MODE_UNSPECIFIED",
			"qan_mongodb_profiler": false,
			"qan_mongodb_mongolog": true
		}}`, run.body)
	})

	t.Run("Address", func(t *testing.T) {
		run, err := runUpdate(t, &UpdateMongoDBCommand{}, "mongodb_srv_1", "10.0.0.1:27018", "--connection-timeout=2s")
		require.NoError(t, err)

		assert.JSONEq(t, `{"mongodb": {"address": "10.0.0.1", "port": 27018, "connection_timeout": "2s"}}`, run.body)
	})

	t.Run("CredentialsSource", func(t *testing.T) {
		credentials := filepath.Join(t.TempDir(), "credentials.json")
		require.NoError(t, os.WriteFile(credentials, []byte(`{"username": "pmm", "password": "s3cret"}`), 0o600))

		run, err := runUpdate(t, &UpdateMongoDBCommand{}, "mongodb_srv_1", "--credentials-source="+credentials)
		require.NoError(t, err)

		assert.JSONEq(t, `{"mongodb": {"username": "pmm", "password": "s3cret"}}`, run.body)

		// The password is shown as "***" before and after, but it is reported as changed.
		result := run.res.(*updateServiceResult)
		assert.Contains(t, result.Changes, settingChange{Flag: "--password", Before: secretValue, After: secretValue})
	})

	t.Run("DryRun", func(t *testing.T) {
		run, err := runUpdate(t, &UpdateMongoDBCommand{}, "mongodb_srv_1", "--max-collections-limit=0", "--dry-run")
		require.NoError(t, err)

		assert.JSONEq(t, `{"dry_run": true, "mongodb": {"collections_limit": 0}}`, run.body)
		assert.Contains(t, run.res.String(), "MongoDB Service not changed (dry run).")
		assert.Contains(t, run.res.String(), "Settings the update would change:")
	})

	t.Run("RejectsMovingTheService", func(t *testing.T) {
		_, err := runUpdate(t, &UpdateMongoDBCommand{}, "mongodb_srv_1", "--node-id=other-node")
		require.EqualError(t, err, "--node-id cannot be changed by update; remove the Service and add it again")
	})
}

func TestUpdateMySQL(t *testing.T) {
	t.Run("Tablestats", func(t *testing.T) {
		for _, tc := range []struct {
			args []string
			body string
		}{
			{[]string{"--disable-tablestats"}, `{"mysql": {"tablestats_group_table_limit": -1}}`},
			{[]string{"--no-disable-tablestats"}, `{"mysql": {"tablestats_group_table_limit": 0}}`},
			{[]string{"--disable-tablestats-limit=500"}, `{"mysql": {"tablestats_group_table_limit": 500}}`},
		} {
			run, err := runUpdate(t, &UpdateMySQLCommand{}, append([]string{"mysql_srv_1"}, tc.args...)...)
			require.NoError(t, err)
			assert.JSONEq(t, tc.body, run.body, tc.args)
		}

		_, err := runUpdate(t, &UpdateMySQLCommand{}, "mysql_srv_1", "--disable-tablestats", "--disable-tablestats-limit=10")
		require.EqualError(t, err, "both --disable-tablestats and --disable-tablestats-limit are passed")
	})

	t.Run("QANSettings", func(t *testing.T) {
		run, err := runUpdate(t, &UpdateMySQLCommand{},
			"mysql_srv_1", "--query-source=perfschema", "--size-slow-logs=1GiB", "--comments-parsing=on", "--extra-dsn-params=allowCleartextPasswords=1")
		require.NoError(t, err)

		assert.JSONEq(t, `{"mysql": {
			"qan_mysql_perfschema": true,
			"qan_mysql_slowlog": false,
			"max_slowlog_file_size": "1073741824",
			"disable_comments_parsing": false,
			"extra_dsn_params": {"values": {"allowCleartextPasswords": "1"}}
		}}`, run.body)
	})
}

func TestUpdatePostgreSQL(t *testing.T) {
	run, err := runUpdate(t, &UpdatePostgreSQLCommand{}, "postgresql_srv_1", "--query-source=pgstatements", "--database=app")
	require.NoError(t, err)
	assert.JSONEq(t, `{"postgresql": {
		"qan_postgresql_pgstatements_agent": true,
		"qan_postgresql_pgstatmonitor_agent": false,
		"database": "app"
	}}`, run.body)

	_, err = runUpdate(t, &UpdatePostgreSQLCommand{}, "postgresql_srv_1", "--query-source=pgstatement")
	require.EqualError(t, err, "--query-source must be one of: pgstatements, pgstatmonitor, none")
}

func TestUpdateValkey(t *testing.T) {
	_, err := runUpdate(t, &UpdateValkeyCommand{}, "valkey_srv_1", "--disable-collectors=x")
	require.EqualError(t, err, "--disable-collectors is not supported for Valkey")
}

func TestUpdateExternal(t *testing.T) {
	run, err := runUpdate(t, &UpdateExternalCommand{},
		"--service-name=external_srv_1", "--listen-port=9105", "--metrics-path=metrics", "--group=exporters")
	require.NoError(t, err)
	assert.JSONEq(t, `{"external": {"listen_port": 9105, "metrics_path": "/metrics", "group": "exporters"}}`, run.body)

	_, err = runUpdate(t, &UpdateExternalServerlessCommand{}, "--external-name=external_srv_1", "--url=http://10.0.0.1:9100/metrics")
	require.EqualError(t, err, "--url cannot be changed by update; remove the Service and add it again")
}

func TestUpdateHAProxy(t *testing.T) {
	run, err := runUpdate(t, &UpdateHAProxyCommand{}, "haproxy_srv_1", "--scheme=https", "--no-tls-skip-verify")
	require.NoError(t, err)
	assert.Equal(t, "/v1/management/services/haproxy_srv_1", run.path)
	assert.JSONEq(t, `{"haproxy": {"scheme": "https", "tls_skip_verify": false}}`, run.body)
}
