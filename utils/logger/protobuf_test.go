// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

var startTime = timestamppb.Now()

func getRtaQueryDataMessage(t *testing.T) []proto.Message {
	t.Helper()

	dataOrig := rtav1.QueryData{
		ServiceId:              "serviceID",
		ServiceName:            "serviceName",
		QueryId:                "static-query-1",
		QueryText:              `{ find: "mycollection", filter: { status: "active" } }`,
		QueryExecutionDuration: durationpb.New(time.Duration(15)),
		QueryCollectTime:       startTime,
		QueryRawJson:           `{ find: "mycollection", filter: { status: "active" } }`,
		ClientAddress:          "127.0.0.1:5060",
		Payload: &rtav1.QueryData_MongoDbPayload{
			MongoDbPayload: &rtav1.QueryMongoDBData{
				DbInstanceAddress:  "c4486b1ebd30:27017",
				DatabaseName:       "mydb",
				ClientAppName:      "myapp",
				Collection:         "mycollection",
				Operation:          "find",
				OperationStartTime: startTime,
				Username:           "test-user",
				PlanSummary:        "COLLSCAN",
			},
		},
	}

	dataRedacted := rtav1.QueryData{
		ServiceId:              "serviceID",
		ServiceName:            "serviceName",
		QueryId:                "static-query-1",
		QueryText:              `{ find: "mycollection", filter: { status: "active" } }`,
		QueryExecutionDuration: durationpb.New(time.Duration(15)),
		QueryCollectTime:       startTime,
		QueryRawJson:           `{ find: "mycollection", filter: { status: "active" } }`,
		ClientAddress:          "127.0.0.1:5060",
		Payload: &rtav1.QueryData_MongoDbPayload{
			MongoDbPayload: &rtav1.QueryMongoDBData{
				DbInstanceAddress:  "c4486b1ebd30:27017",
				DatabaseName:       "mydb",
				ClientAppName:      "myapp",
				Collection:         "mycollection",
				Operation:          "find",
				OperationStartTime: startTime,
				Username:           "***REDACTED***",
				PlanSummary:        "COLLSCAN",
			},
		},
	}

	return []proto.Message{&dataOrig, &dataRedacted}
}

//nolint:gosec
func getSetStateRequestMessage(t *testing.T) []proto.Message {
	t.Helper()

	agentProcsOrig := make(map[string]*agentv1.SetStateRequest_AgentProcess)
	agentProcsOrig[inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER.String()] = &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"postgres_exporter",
			"--web.listen-address=:9187",
			"--web.telemetry-path=/metrics",
			"--log.level=info",
			"--web.disable-exporter-metrics=true",
			"--web.disable-admin-api=true",
		},
		Env: []string{
			"mysql://admin-user:admin-passwd@localhost:3306/admin?param=value",
			"postgres://user:password@localhost:5432/dbname?param=value",
		},
		TextFiles: map[string]string{
			"/etc/agent/config.yaml": "agent_config: value\npassword: mysql://admin-user:admin-passwd@localhost:3306/admin?param=value",
		},
		RedactWords: []string{
			"mysql://admin-user:admin-passwd@localhost:3306/admin?param=value",
			"postgres://user:password@localhost:5432/dbname?param=value",
		},
		EnvVariableNames: []string{
			"ENV_VAR1",
			"ENV_VAR2",
		},
	}

	biAgentsOrig := make(map[string]*agentv1.SetStateRequest_BuiltinAgent)
	biAgentsOrig[inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT.String()] = &agentv1.SetStateRequest_BuiltinAgent{
		Type: inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT,
		Dsn:  "mysql://admin-user:admin-passwd@localhost:3306/admin?param=value",
		Env: map[string]string{
			"ENV_VAR1": "mysql://admin-user:admin-passwd@localhost:3306/admin?param=value",
			"ENV_VAR2": "postgres://user:password@localhost:5432/dbname?param=value",
		},
		MaxQueryLength:         1024,
		DisableCommentsParsing: false,
		DisableQueryExamples:   false,
		MaxQueryLogSize:        10 * 1024 * 1024,
		TextFiles: &agentv1.TextFiles{
			Files: map[string]string{
				"/etc/agent/config.yaml": "agent_config: value\npassword: mysql://admin-user:admin-passwd@localhost:3306/admin?param=value",
			},
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
		},
		Tls:           true,
		TlsSkipVerify: true,
		ServiceId:     "svc-1",
		ServiceName:   "service-1",
		RtaOptions: &inventoryv1.RTAOptions{
			CollectInterval: durationpb.New(15 * time.Second),
		},
	}
	msgOrig := &agentv1.ServerMessage{
		Payload: &agentv1.ServerMessage_SetState{
			SetState: &agentv1.SetStateRequest{
				AgentProcesses: agentProcsOrig,
				BuiltinAgents:  biAgentsOrig,
			},
		},
	}
	// REDACTED versions of the messages
	AgentProcsRedacted := make(map[string]*agentv1.SetStateRequest_AgentProcess)
	AgentProcsRedacted[inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER.String()] = &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"postgres_exporter",
			"--web.listen-address=:9187",
			"--web.telemetry-path=/metrics",
			"--log.level=info",
			"--web.disable-exporter-metrics=true",
			"--web.disable-admin-api=true",
		},
		Env: []string{
			"mysql://***REDACTED***:***REDACTED***@localhost:3306/admin?param=value",
			"postgres://***REDACTED***:***REDACTED***@localhost:5432/dbname?param=value",
		},
		TextFiles: map[string]string{
			"/etc/agent/config.yaml": "ag***REDACTED***ue",
		},
		RedactWords: []string{
			"***REDACTED***",
			"***REDACTED***",
		},
		EnvVariableNames: []string{
			"ENV_VAR1",
			"ENV_VAR2",
		},
	}

	biAgentsRedacted := make(map[string]*agentv1.SetStateRequest_BuiltinAgent)
	biAgentsRedacted[inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT.String()] = &agentv1.SetStateRequest_BuiltinAgent{
		Type: inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT,
		Dsn:  "mysql://***REDACTED***:***REDACTED***@localhost:3306/admin?param=value",
		Env: map[string]string{
			"ENV_VAR1": "mysql://***REDACTED***:***REDACTED***@localhost:3306/admin?param=value",
			"ENV_VAR2": "postgres://***REDACTED***:***REDACTED***@localhost:5432/dbname?param=value",
		},
		MaxQueryLength:         1024,
		DisableCommentsParsing: false,
		DisableQueryExamples:   false,
		MaxQueryLogSize:        10 * 1024 * 1024,
		TextFiles: &agentv1.TextFiles{
			Files: map[string]string{
				"/etc/agent/config.yaml": "ag***REDACTED***ue",
			},
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
		},
		Tls:           true,
		TlsSkipVerify: true,
		ServiceId:     "svc-1",
		ServiceName:   "service-1",
		RtaOptions: &inventoryv1.RTAOptions{
			CollectInterval: durationpb.New(15 * time.Second),
		},
	}
	msgRedacted := &agentv1.ServerMessage{
		Payload: &agentv1.ServerMessage_SetState{
			SetState: &agentv1.SetStateRequest{
				AgentProcesses: AgentProcsRedacted,
				BuiltinAgents:  biAgentsRedacted,
			},
		},
	}
	return []proto.Message{msgOrig, msgRedacted}
}

func getStartActionRequestPtpgSummaryParams(t *testing.T) []proto.Message {
	t.Helper()

	startActionOrig := &agentv1.StartActionRequest{
		ActionId: "action-1",
		Timeout:  durationpb.New(30 * time.Second),
		Params: &agentv1.StartActionRequest_PtPgSummaryParams{
			PtPgSummaryParams: &agentv1.StartActionRequest_PTPgSummaryParams{
				Host:     "localhost",
				Port:     5432,
				Username: "test-user",
				Password: "test-password",
			},
		},
	}

	startActionRedacted := &agentv1.StartActionRequest{
		ActionId: "action-1",
		Timeout:  durationpb.New(30 * time.Second),
		Params: &agentv1.StartActionRequest_PtPgSummaryParams{
			PtPgSummaryParams: &agentv1.StartActionRequest_PTPgSummaryParams{
				Host:     "localhost",
				Port:     5432,
				Username: maskedString,
				Password: maskedString,
			},
		},
	}
	return []proto.Message{startActionOrig, startActionRedacted}
}

func getStartActionRequestPTMongoDBSummaryParams(t *testing.T) []proto.Message {
	t.Helper()

	startActionOrig := &agentv1.StartActionRequest{
		ActionId: "action-1",
		Timeout:  durationpb.New(30 * time.Second),
		Params: &agentv1.StartActionRequest_PtMongodbSummaryParams{
			PtMongodbSummaryParams: &agentv1.StartActionRequest_PTMongoDBSummaryParams{
				Host:     "localhost",
				Port:     5432,
				Username: "test-user",
				Password: "test-password",
			},
		},
	}

	startActionRedacted := &agentv1.StartActionRequest{
		ActionId: "action-1",
		Timeout:  durationpb.New(30 * time.Second),
		Params: &agentv1.StartActionRequest_PtMongodbSummaryParams{
			PtMongodbSummaryParams: &agentv1.StartActionRequest_PTMongoDBSummaryParams{
				Host:     "localhost",
				Port:     5432,
				Username: maskedString,
				Password: maskedString,
			},
		},
	}
	return []proto.Message{startActionOrig, startActionRedacted}
}

func getStartActionRequestPTMySQLSummaryParams(t *testing.T) []proto.Message {
	t.Helper()

	startActionOrig := &agentv1.StartActionRequest{
		ActionId: "action-1",
		Timeout:  durationpb.New(30 * time.Second),
		Params: &agentv1.StartActionRequest_PtMysqlSummaryParams{
			PtMysqlSummaryParams: &agentv1.StartActionRequest_PTMySQLSummaryParams{
				Host:     "localhost",
				Port:     5432,
				Username: "test-user",
				Password: "test-password",
			},
		},
	}

	startActionRedacted := &agentv1.StartActionRequest{
		ActionId: "action-1",
		Timeout:  durationpb.New(30 * time.Second),
		Params: &agentv1.StartActionRequest_PtMysqlSummaryParams{
			PtMysqlSummaryParams: &agentv1.StartActionRequest_PTMySQLSummaryParams{
				Host:     "localhost",
				Port:     5432,
				Username: maskedString,
				Password: maskedString,
			},
		},
	}
	return []proto.Message{startActionOrig, startActionRedacted}
}

//nolint:gosec
func getServerMessageMessage(t *testing.T) []proto.Message {
	t.Helper()

	orig := &agentv1.ServerMessage{
		Id: 10,
		Payload: &agentv1.ServerMessage_ServiceInfo{
			ServiceInfo: &agentv1.ServiceInfoRequest{
				Type: inventoryv1.ServiceType_SERVICE_TYPE_POSTGRESQL_SERVICE,
				Dsn:  "postgres://user:password@pmm2-qa-postgresql.postgres.database.azure.com:5432/postgres?connect_timeout=1&sslmode=disable",
			},
		},
	}

	redacted := &agentv1.ServerMessage{
		Id: 10,
		Payload: &agentv1.ServerMessage_ServiceInfo{
			ServiceInfo: &agentv1.ServiceInfoRequest{
				Type: inventoryv1.ServiceType_SERVICE_TYPE_POSTGRESQL_SERVICE,
				Dsn:  "postgres://***REDACTED***:***REDACTED***@pmm2-qa-postgresql.postgres.database.azure.com:5432/postgres?connect_timeout=1&sslmode=disable",
			},
		},
	}
	return []proto.Message{orig, redacted}
}

func TestRedactMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input proto.Message
		want  proto.Message
	}{
		{
			name:  "nil message",
			input: nil,
			want:  nil,
		},
		{
			name:  "non-sensitive message",
			input: getRtaQueryDataMessage(t)[0],
			want:  getRtaQueryDataMessage(t)[1],
		},
		{
			name:  "sensitive message",
			input: getSetStateRequestMessage(t)[0],
			want:  getSetStateRequestMessage(t)[1],
		},
		{
			name:  "StartActionRequest_PTPgSummaryParams message with sensitive data",
			input: getStartActionRequestPtpgSummaryParams(t)[0],
			want:  getStartActionRequestPtpgSummaryParams(t)[1],
		},
		{
			name:  "StartActionRequest_PTMongoDBSummaryParams message with sensitive data",
			input: getStartActionRequestPTMongoDBSummaryParams(t)[0],
			want:  getStartActionRequestPTMongoDBSummaryParams(t)[1],
		},
		{
			name:  "StartActionRequest_PTMySQLSummaryParams message with sensitive data",
			input: getStartActionRequestPTMySQLSummaryParams(t)[0],
			want:  getStartActionRequestPTMySQLSummaryParams(t)[1],
		},
		{
			name:  "ServerMessage message with sensitive data",
			input: getServerMessageMessage(t)[0],
			want:  getServerMessageMessage(t)[1],
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			redactedMsg := RedactMessage(tt.input)
			if diff := cmp.Diff(tt.want, redactedMsg, protocmp.Transform()); diff != "" {
				t.Errorf("RedactMessage() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_maskString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "short string",
			input: "abc",
			want:  "***REDACTED***",
		},
		{
			name:  "exactly 4 characters",
			input: "abcd",
			want:  "***REDACTED***",
		},
		{
			name:  "long string",
			input: "mysecretpassword",
			want:  "my***REDACTED***rd",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, maskString(tt.input), "maskString() should return the expected redacted string")
		})
	}
}

//nolint:gosec
func Test_maskDSN(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "PostgreSQL DSN",
			input: "postgres://user:password@localhost:5432/dbname?param=value",
			want:  "postgres://***REDACTED***:***REDACTED***@localhost:5432/dbname?param=value",
		},
		{
			name:  "MySQL DSN",
			input: "mysql://user:password@localhost:3306/dbname?param=value",
			want:  "mysql://***REDACTED***:***REDACTED***@localhost:3306/dbname?param=value",
		},
		{
			name:  "MongoDB DSN",
			input: "mongodb://user:password@host.docker.internal:27017/?connectTimeoutMS=2000&direct",
			want:  "mongodb://***REDACTED***:***REDACTED***@host.docker.internal:27017/?connectTimeoutMS=2000&direct",
		},
		{
			name:  "DSN without credentials",
			input: "postgres://localhost:5432/dbname?param=value",
			want:  "postgres://localhost:5432/dbname?param=value",
		},
		{
			name:  "DSN without scheme",
			input: "user:password@localhost:3306/dbname?param=value",
			want:  "***REDACTED***:***REDACTED***@localhost:3306/dbname?param=value",
		},
		{
			name:  "Invalid DSN",
			input: "not-a-valid-dsn",
			want:  "not-a-valid-dsn",
		},
		{
			name:  "DSN without scheme and several @ characters-1",
			input: "us@er:password@tcp(pmm2-qa-mysql.mysql.database.azure.com:3306/dbname?param=value",
			want:  "***REDACTED***:***REDACTED***@tcp(pmm2-qa-mysql.mysql.database.azure.com:3306/dbname?param=value",
		},
		{
			name:  "DSN without scheme and several @ characters-2",
			input: "user:p@ssword@tcp(pmm2-qa-mysql.mysql.database.azure.com:3306/dbname?param=value",
			want:  "***REDACTED***:***REDACTED***@tcp(pmm2-qa-mysql.mysql.database.azure.com:3306/dbname?param=value",
		},
		{
			name:  "DSN with scheme and several @ characters-1",
			input: "mysql://us@er:password@tcp(pmm2-qa-mysql.mysql.database.azure.com:3306/dbname?param=value",
			want:  "mysql://***REDACTED***:***REDACTED***@tcp(pmm2-qa-mysql.mysql.database.azure.com:3306/dbname?param=value",
		},
		{
			name:  "DSN with scheme and several @ characters-2",
			input: "mysql://user:p@ssword@tcp(pmm2-qa-mysql.mysql.database.azure.com:3306/dbname?param=value",
			want:  "mysql://***REDACTED***:***REDACTED***@tcp(pmm2-qa-mysql.mysql.database.azure.com:3306/dbname?param=value",
		},
		{
			name:  "DSN with scheme and several @ characters-3",
			input: "mysql://us@r:p@ssword@tcp(pmm2-qa-mysql.mysql.database.azure.com:3306/dbname?param=value",
			want:  "mysql://***REDACTED***:***REDACTED***@tcp(pmm2-qa-mysql.mysql.database.azure.com:3306/dbname?param=value",
		},
		{
			name:  "DSN sxheme_ip without credentials",
			input: "scheme://127.0.0.1/foo/bar?key=value",
			want:  "scheme://127.0.0.1/foo/bar?key=value",
		},
		{
			name:  "DSN scheme_path without credentials",
			input: "scheme:///var/local/run/memcached.socket?weight=25",
			want:  "scheme:///var/local/run/memcached.socket?weight=25",
		},
		{
			name:  "DSN scheme_path with credentials",
			input: "scheme://user:pass@/var/local/run/memcached.socket?weight=25",
			want:  "scheme://***REDACTED***:***REDACTED***@/var/local/run/memcached.socket?weight=25",
		},
		{
			name:  "DSN scheme_path with credentials with several @ characters-1",
			input: "scheme://us@r:pass@/var/local/run/memcached.socket?weight=25",
			want:  "scheme://***REDACTED***:***REDACTED***@/var/local/run/memcached.socket?weight=25",
		},
		{
			name:  "DSN scheme_path with credentials with several @ characters-2",
			input: "scheme://user:p@ss@/var/local/run/memcached.socket?weight=25",
			want:  "scheme://***REDACTED***:***REDACTED***@/var/local/run/memcached.socket?weight=25",
		},
		{
			name:  "DSN scheme_path with credentials with several @ characters-3",
			input: "scheme://us@r:p@ss@/var/local/run/memcached.socket?weight=25",
			want:  "scheme://***REDACTED***:***REDACTED***@/var/local/run/memcached.socket?weight=25",
		},
		{
			name:  "DSN scheme_path without credentials",
			input: "scheme://a",
			want:  "scheme://a",
		},
		{
			name:  "DSN server:port without credentials",
			input: "server:80",
			want:  "server:80",
		},
		{
			name:  "escaped DSN",
			input: "postgres://pmm2qa:7w%5ELXF8%5EqUaPNfD@pmm2-qa-postgresql.postgres.database.azure.com:5432/postgres?connect_timeout=1&sslmode=disable",
			want:  "postgres://***REDACTED***:***REDACTED***@pmm2-qa-postgresql.postgres.database.azure.com:5432/postgres?connect_timeout=1&sslmode=disable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, MaskDSN(tt.input), "maskDSN() should return the expected redacted DSN")
		})
	}
}
