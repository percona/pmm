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

package connectionchecker

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionChecker(t *testing.T) {
	tests := []struct {
		name     string
		msg      *agentpb.CheckConnectionRequest
		expected string
	}{
		{
			name: "MySQL",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:     "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=1s",
				Type:    inventorypb.ServiceType_MYSQL_SERVICE,
				Timeout: ptypes.DurationProto(3 * time.Second),
			},
		}, {
			name: "MySQL wrong params",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:     "pmm-agent:pmm-agent-wrong-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=1s",
				Type:    inventorypb.ServiceType_MYSQL_SERVICE,
				Timeout: ptypes.DurationProto(3 * time.Second),
			},
			expected: `Error 1045: Access denied for user 'pmm-agent'@'.+' \(using password: YES\)`,
		}, {
			name: "MySQL timeout",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:     "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=10s",
				Type:    inventorypb.ServiceType_MYSQL_SERVICE,
				Timeout: ptypes.DurationProto(time.Nanosecond),
			},
			expected: `context deadline exceeded`,
		},

		{
			name: "PostgreSQL",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:     "postgres://pmm-agent:pmm-agent-password@127.0.0.1:15432/postgres?connect_timeout=1&sslmode=disable",
				Type:    inventorypb.ServiceType_POSTGRESQL_SERVICE,
				Timeout: ptypes.DurationProto(3 * time.Second),
			},
		}, {
			name: "PostgreSQL wrong params",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:     "postgres://pmm-agent:pmm-agent-wrong-password@127.0.0.1:15432/postgres?connect_timeout=1&sslmode=disable",
				Type:    inventorypb.ServiceType_POSTGRESQL_SERVICE,
				Timeout: ptypes.DurationProto(3 * time.Second),
			},
			expected: `pq: password authentication failed for user "pmm-agent"`,
		}, {
			name: "PostgreSQL timeout",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:     "postgres://pmm-agent:pmm-agent-password@127.0.0.1:15432/postgres?connect_timeout=10&sslmode=disable",
				Type:    inventorypb.ServiceType_POSTGRESQL_SERVICE,
				Timeout: ptypes.DurationProto(time.Nanosecond),
			},
			expected: `context deadline exceeded`,
		},

		{
			name: "MongoDB",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:     "mongodb://root:root-password@127.0.0.1:27017/admin?connectTimeoutMS=1000",
				Type:    inventorypb.ServiceType_MONGODB_SERVICE,
				Timeout: ptypes.DurationProto(3 * time.Second),
			},
		}, {
			name: "MongoDB wrong params",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:     "mongodb://root:root-password-wrong@127.0.0.1:27017/admin?connectTimeoutMS=1000",
				Type:    inventorypb.ServiceType_MONGODB_SERVICE,
				Timeout: ptypes.DurationProto(3 * time.Second),
			},
			expected: `auth error: sasl conversation error: unable to authenticate using mechanism "SCRAM-SHA-(1|256)": ` +
				`\(AuthenticationFailed\) Authentication failed.`,
		}, {
			name: "MongoDB timeout",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:     "mongodb://root:root-password@127.0.0.1:27017/admin?connectTimeoutMS=10000",
				Type:    inventorypb.ServiceType_MONGODB_SERVICE,
				Timeout: ptypes.DurationProto(time.Nanosecond),
			},
			expected: `context deadline exceeded`,
		}, {
			name: "MongoDB no database",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:     "mongodb://root:root-password@127.0.0.1:27017?connectTimeoutMS=1000",
				Type:    inventorypb.ServiceType_MONGODB_SERVICE,
				Timeout: ptypes.DurationProto(3 * time.Second),
			},
			expected: `error parsing uri: must have a / before the query \?`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(context.Background())
			err := c.Check(tt.msg)
			if tt.expected == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Regexp(t, `^`+tt.expected+`$`, err.Error())
			}
		})
	}
}
