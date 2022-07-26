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
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
)

func TestConnectionChecker(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		req         *agentpb.CheckConnectionRequest
		expectedErr string
		panic       bool
	}{
		{
			name: "MySQL",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=1s",
				Type:    inventorypb.ServiceType_MYSQL_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
		},
		{
			name: "MySQL wrong params",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "pmm-agent:pmm-agent-wrong-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=1s",
				Type:    inventorypb.ServiceType_MYSQL_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `Error 1045: Access denied for user 'pmm-agent'@'.+' \(using password: YES\)`,
		},
		{
			name: "MySQL timeout",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=10s",
				Type:    inventorypb.ServiceType_MYSQL_SERVICE,
				Timeout: durationpb.New(time.Nanosecond),
			},
			expectedErr: `context deadline exceeded`,
		},

		{
			name: "MongoDB with no auth",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "mongodb://127.0.0.1:27019/admin?connectTimeoutMS=1000",
				Type:    inventorypb.ServiceType_MONGODB_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
		},
		{
			name: "MongoDB with no auth with params",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "mongodb://root:root-password@127.0.0.1:27019/admin?connectTimeoutMS=1000",
				Type:    inventorypb.ServiceType_MONGODB_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `.*auth error: (sasl conversation error: )?unable to authenticate using mechanism "[\w-]+": ` +
				`\(AuthenticationFailed\) Authentication failed.`,
		},
		{
			name: "MongoDB",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "mongodb://root:root-password@127.0.0.1:27017/admin?connectTimeoutMS=1000",
				Type:    inventorypb.ServiceType_MONGODB_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
		},
		{
			name: "MongoDB no params",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "mongodb://127.0.0.1:27017/admin?connectTimeoutMS=1000",
				Type:    inventorypb.ServiceType_MONGODB_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `\(Unauthorized\) (?:command listDatabases requires authentication|` +
				`there are no users authenticated|` +
				`not authorized on admin to execute command \{ listDatabases\: 1 \})`,
		},
		{
			name: "MongoDB wrong params",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "mongodb://root:root-password-wrong@127.0.0.1:27017/admin?connectTimeoutMS=1000",
				Type:    inventorypb.ServiceType_MONGODB_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `.*auth error: (sasl conversation error: )?unable to authenticate using mechanism "[\w-]+": ` +
				`\(AuthenticationFailed\) Authentication failed.`,
		},
		{
			name: "MongoDB timeout",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "mongodb://root:root-password@127.0.0.1:27017/admin?connectTimeoutMS=10000",
				Type:    inventorypb.ServiceType_MONGODB_SERVICE,
				Timeout: durationpb.New(time.Nanosecond),
			},
			expectedErr: `server selection error: context deadline exceeded, current topology: \{ Type: Unknown, Servers: \[\{ Addr: 127.0.0.1:27017, Type: Unknown \}, \] \}`,
		},
		{
			name: "MongoDB no database",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "mongodb://root:root-password@127.0.0.1:27017?connectTimeoutMS=1000",
				Type:    inventorypb.ServiceType_MONGODB_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `error parsing uri: must have a / before the query \?`,
		},

		{
			name: "PostgreSQL",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "postgres://pmm-agent:pmm-agent-password@127.0.0.1:5432/postgres?connect_timeout=1&sslmode=disable",
				Type:    inventorypb.ServiceType_POSTGRESQL_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
		},
		{
			name: "PostgreSQL wrong params",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "postgres://pmm-agent:pmm-agent-wrong-password@127.0.0.1:5432/postgres?connect_timeout=1&sslmode=disable",
				Type:    inventorypb.ServiceType_POSTGRESQL_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `pq: password authentication failed for user "pmm-agent"`,
		},
		{
			name: "PostgreSQL timeout",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "postgres://pmm-agent:pmm-agent-password@127.0.0.1:5432/postgres?connect_timeout=10&sslmode=disable",
				Type:    inventorypb.ServiceType_POSTGRESQL_SERVICE,
				Timeout: durationpb.New(time.Nanosecond),
			},
			expectedErr: `context deadline exceeded`,
		},

		// Use MySQL for ProxySQL tests for now.
		// TODO https://jira.percona.com/browse/PMM-4930
		{
			name: "ProxySQL/MySQL",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=1s",
				Type:    inventorypb.ServiceType_PROXYSQL_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
		},
		{
			name: "ProxySQL/MySQL wrong params",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "pmm-agent:pmm-agent-wrong-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=1s",
				Type:    inventorypb.ServiceType_PROXYSQL_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `Error 1045: Access denied for user 'pmm-agent'@'.+' \(using password: YES\)`,
		},
		{
			name: "ProxySQL/MySQL timeout",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=10s",
				Type:    inventorypb.ServiceType_PROXYSQL_SERVICE,
				Timeout: durationpb.New(time.Nanosecond),
			},
			expectedErr: `context deadline exceeded`,
		},
		{
			name: "Invalid service type",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=10s",
				Type:    inventorypb.ServiceType_SERVICE_TYPE_INVALID,
				Timeout: durationpb.New(time.Nanosecond),
			},
			expectedErr: `unknown service type: SERVICE_TYPE_INVALID`,
			panic:       true,
		},
		{
			name: "Unknown service type",
			req: &agentpb.CheckConnectionRequest{
				Dsn:     "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=10s",
				Type:    inventorypb.ServiceType(12345),
				Timeout: durationpb.New(time.Nanosecond),
			},
			expectedErr: `unknown service type: 12345`,
			panic:       true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			temp, err := os.MkdirTemp("", "pmm-agent-")
			require.NoError(t, err)

			c := New(&config.Paths{
				TempDir: temp,
			})

			if tt.panic {
				require.PanicsWithValue(t, tt.expectedErr, func() {
					c.Check(context.Background(), tt.req, 0)
				})
				return
			}

			resp := c.Check(context.Background(), tt.req, 0)
			require.NotNil(t, resp)
			if tt.expectedErr == "" {
				assert.Empty(t, resp.Error)
			} else {
				require.NotEmpty(t, resp.Error)
				assert.Regexp(t, `^`+tt.expectedErr+`$`, resp.Error)
			}
		})
	}

	t.Run("TableCount", func(t *testing.T) {
		temp, err := os.MkdirTemp("", "pmm-agent-")
		require.NoError(t, err)

		c := New(&config.Paths{
			TempDir: temp,
		})
		resp := c.Check(context.Background(), &agentpb.CheckConnectionRequest{
			Dsn:  "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=1s",
			Type: inventorypb.ServiceType_MYSQL_SERVICE,
		}, 0)
		require.NotNil(t, resp)
		assert.InDelta(t, 250, resp.Stats.TableCount, 150)
	})

	t.Run("MongoDBWithSSL", func(t *testing.T) {
		mongoDBDSNWithSSL, mongoDBTextFiles := tests.GetTestMongoDBWithSSLDSN(t, "../")
		temp, err := os.MkdirTemp("", "pmm-agent-")
		require.NoError(t, err)

		c := New(&config.Paths{
			TempDir: temp,
		})
		resp := c.Check(context.Background(), &agentpb.CheckConnectionRequest{
			Dsn:       mongoDBDSNWithSSL,
			Type:      inventorypb.ServiceType_MONGODB_SERVICE,
			Timeout:   durationpb.New(30 * time.Second),
			TextFiles: mongoDBTextFiles,
		}, rand.Uint32()) //nolint:gosec
		require.NotNil(t, resp)
		assert.Empty(t, resp.Error)
	})
}
