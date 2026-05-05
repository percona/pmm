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

var rtaData = rtav1.QueryData{
	ServiceId:              "serviceID",
	ServiceName:            "serviceName",
	QueryId:                "static-query-1",
	QueryText:              `{ find: "mycollection", filter: { status: "active" } }`,
	QueryExecutionDuration: durationpb.New(time.Duration(15)),
	QueryCollectTime:       timestamppb.Now(),
	QueryRawJson:           `{ find: "mycollection", filter: { status: "active" } }`,
	ClientAddress:          "127.0.0.1:5060",
	Payload: &rtav1.QueryData_MongoDbPayload{
		MongoDbPayload: &rtav1.QueryMongoDBData{
			DbInstanceAddress:  "c4486b1ebd30:27017",
			DatabaseName:       "mydb",
			ClientAppName:      "myapp",
			Collection:         "mycollection",
			Operation:          "find",
			OperationStartTime: timestamppb.Now(),
			Username:           "test-user",
			PlanSummary:        "COLLSCAN",
		},
	},
}

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
			"my***REDACTED***ue",
			"po***REDACTED***ue",
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
			input: &rtaData,
			want:  &rtaData,
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, MaskDSN(tt.input), "maskDSN() should return the expected redacted DSN")
		})
	}
}
