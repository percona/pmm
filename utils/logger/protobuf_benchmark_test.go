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
