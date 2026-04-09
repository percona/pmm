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
	"fmt"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

func getSensitiveMessages(tb testing.TB, n int32) proto.Message {
	tb.Helper()

	agentProcs := make(map[string]*agentv1.SetStateRequest_AgentProcess)
	biAgents := make(map[string]*agentv1.SetStateRequest_BuiltinAgent)
	for i := range n {
		biAgents[inventoryv1.AgentType(i+1).String()] = &agentv1.SetStateRequest_BuiltinAgent{
			Type: inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT,
			Dsn:  fmt.Sprintf("mysql://admin-user-%d:admin-passwd-%d@localhost:3306/admin?param=value", n, n),
			Env: map[string]string{
				"USERNAME": "agent",
				"PASSWORD": fmt.Sprintf("mysql://admin-user-%d:admin-passwd-%d@localhost:3306/admin?param=value", n, n),
			},
			MaxQueryLength:         1024,
			DisableCommentsParsing: false,
			DisableQueryExamples:   false,
			MaxQueryLogSize:        10 * 1024 * 1024,
			Tls:                    true,
			TlsSkipVerify:          true,
			ServiceId:              fmt.Sprintf("svc-%d", n),
			ServiceName:            fmt.Sprintf("service-%d", n),
			RtaOptions: &inventoryv1.RTAOptions{
				CollectInterval: durationpb.New(15 * time.Second),
			},
		}

		agentProcs[inventoryv1.AgentType(i+1).String()] = &agentv1.SetStateRequest_AgentProcess{
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
				fmt.Sprintf("mysql://admin-user-%d:admin-passwd-%d@localhost:3306/admin?param=value", n, n),
				fmt.Sprintf("postgres://admin-user-%d:admin-passwd-%d@localhost:5432/admin?param=value", n, n),
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
	}
	return &agentv1.ServerMessage{
		Payload: &agentv1.ServerMessage_SetState{
			SetState: &agentv1.SetStateRequest{
				AgentProcesses: agentProcs,
				BuiltinAgents:  biAgents,
			},
		},
	}
}

func BenchmarkFormatWithRedactionSetStateRequest(b *testing.B) {
	msg := getSensitiveMessages(b, 5)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = prototext.Format(RedactMessage(msg))
	}
}

func BenchmarkFormatWithoutRedactionSetStateRequest(b *testing.B) {
	msg := getSensitiveMessages(b, 5)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = prototext.Format(msg)
	}
}