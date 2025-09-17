// Copyright 2023 Percona LLC
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

package serviceinfobroker

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/utils/tests"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

func TestServiceInfoBroker(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		req         *agentv1.ServiceInfoRequest
		expectedErr string
		panic       bool
	}{
		{
			name: "MySQL",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=1s",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_MYSQL_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
		},
		{
			name: "MySQL wrong params",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "pmm-agent:pmm-agent-wrong-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=1s",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_MYSQL_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `Error 1045 \(28000\): Access denied for user 'pmm-agent'@'.+' \(using password: YES\)`,
		},
		{
			name: "MySQL timeout",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=10s",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_MYSQL_SERVICE,
				Timeout: durationpb.New(time.Nanosecond),
			},
			expectedErr: `context deadline exceeded`,
		},

		{
			name: "MongoDB with no auth",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "mongodb://127.0.0.1:27019/admin?connectTimeoutMS=1000",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
		},
		{
			name: "MongoDB with no auth with params",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "mongodb://root:root-password@127.0.0.1:27019/admin?connectTimeoutMS=1000",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `.*auth error: (sasl conversation error: )?unable to authenticate using mechanism "[\w-]+": ` +
				`\(AuthenticationFailed\) Authentication failed.`,
		},
		{
			name: "MongoDB",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "mongodb://root:root-password@127.0.0.1:27017/admin?connectTimeoutMS=1000",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
		},
		{
			name: "MongoDB no params",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "mongodb://127.0.0.1:27017/admin?connectTimeoutMS=1000",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
		},
		{
			name: "MongoDB wrong params",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "mongodb://root:root-password-wrong@127.0.0.1:27017/admin?connectTimeoutMS=1000",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `.*auth error: (sasl conversation error: )?unable to authenticate using mechanism "[\w-]+": ` +
				`\(AuthenticationFailed\) Authentication failed.`,
		},
		{
			name: "MongoDB timeout",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "mongodb://root:root-password@127.0.0.1:27017/admin?connectTimeoutMS=10000",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE,
				Timeout: durationpb.New(time.Nanosecond),
			},
			expectedErr: `.*context deadline exceeded.*`,
		},
		{
			name: "MongoDB no database",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "mongodb://root:root-password@127.0.0.1:27017?connectTimeoutMS=1000",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `error parsing uri: must have a / before the query \?`,
		},

		{
			name: "PostgreSQL",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "postgres://pmm-agent:pmm-agent-password@127.0.0.1:5432/postgres?connect_timeout=1&sslmode=disable",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_POSTGRESQL_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
		},
		{
			name: "PostgreSQL wrong params",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "postgres://pmm-agent:pmm-agent-wrong-password@127.0.0.1:5432/postgres?connect_timeout=1&sslmode=disable",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_POSTGRESQL_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `pq: password authentication failed for user "pmm-agent"`,
		},
		{
			name: "PostgreSQL timeout",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "postgres://pmm-agent:pmm-agent-password@127.0.0.1:5432/postgres?connect_timeout=10&sslmode=disable",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_POSTGRESQL_SERVICE,
				Timeout: durationpb.New(time.Nanosecond),
			},
			expectedErr: `context deadline exceeded`,
		},

		// Use MySQL for ProxySQL tests for now.
		// TODO https://jira.percona.com/browse/PMM-4930
		// NOTE the above will also fix the error `Error 1193 (HY000): Unknown system variable 'admin-version'`
		{
			name: "ProxySQL/MySQL",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=1s",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_PROXYSQL_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `Error 1193 \(HY000\): Unknown system variable 'admin-version'`,
		},
		{
			name: "ProxySQL/MySQL wrong params",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "pmm-agent:pmm-agent-wrong-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=1s",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_PROXYSQL_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `Error 1045 \(28000\): Access denied for user 'pmm-agent'@'.+' \(using password: YES\)`,
		},
		{
			name: "ProxySQL/MySQL timeout",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=10s",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_PROXYSQL_SERVICE,
				Timeout: durationpb.New(time.Nanosecond),
			},
			expectedErr: `context deadline exceeded`,
		},
		{
			name: "Invalid service type",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=10s",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_UNSPECIFIED,
				Timeout: durationpb.New(time.Nanosecond),
			},
			expectedErr: `unknown service type: SERVICE_TYPE_UNSPECIFIED`,
			panic:       true,
		},
		{
			name: "Unknown service type",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=10s",
				Type:    inventoryv1.ServiceType(12345),
				Timeout: durationpb.New(time.Nanosecond),
			},
			expectedErr: `unknown service type: 12345`,
			panic:       true,
		},
		{
			name: "Valkey",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "redis://default:pmm-agent_password@127.0.0.1:6379",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_VALKEY_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
		},
		{
			name: "Valkey wrong params",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "redis://default:pmm-agent_wrong_password@127.0.0.1:6379",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_VALKEY_SERVICE,
				Timeout: durationpb.New(3 * time.Second),
			},
			expectedErr: `WRONGPASS invalid username-password pair or user is disabled.`,
		},
		{
			name: "Valkey timeout",
			req: &agentv1.ServiceInfoRequest{
				Dsn:     "redis://default:pmm-agent_password@127.0.0.1:6379",
				Type:    inventoryv1.ServiceType_SERVICE_TYPE_VALKEY_SERVICE,
				Timeout: durationpb.New(time.Nanosecond),
			},
			expectedErr: `dial tcp 127.0.0.1:6379: i/o timeout`,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfgStorage := config.NewStorage(&config.Config{
				Paths: config.Paths{TempDir: t.TempDir()},
			})
			c := New(cfgStorage)

			if tt.panic {
				require.PanicsWithValue(t, tt.expectedErr, func() {
					c.GetInfoFromService(context.Background(), tt.req, 0)
				})
				return
			}

			resp := c.GetInfoFromService(context.Background(), tt.req, 0)
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
		cfgStorage := config.NewStorage(&config.Config{
			Paths: config.Paths{TempDir: t.TempDir()},
		})
		c := New(cfgStorage)
		resp := c.GetInfoFromService(context.Background(), &agentv1.ServiceInfoRequest{
			Dsn:  "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=1s",
			Type: inventoryv1.ServiceType_SERVICE_TYPE_MYSQL_SERVICE,
		}, 0)
		require.NotNil(t, resp)
		assert.InDelta(t, 250, resp.TableCount, 150)
	})

	t.Run("PostgreSQLOptions", func(t *testing.T) {
		cfgStorage := config.NewStorage(&config.Config{
			Paths: config.Paths{TempDir: t.TempDir()},
		})
		c := New(cfgStorage)
		resp := c.GetInfoFromService(context.Background(), &agentv1.ServiceInfoRequest{
			Dsn:  tests.GetTestPostgreSQLDSN(t),
			Type: inventoryv1.ServiceType_SERVICE_TYPE_POSTGRESQL_SERVICE,
		}, 0)
		require.NotNil(t, resp)
		assert.Equal(t, []string{"postgres", "pmm-agent"}, resp.DatabaseList)
		assert.Equal(t, "", *resp.PgsmVersion)
	})

	t.Run("MongoDBWithSSL", func(t *testing.T) {
		mongoDBDSNWithSSL, mongoDBTextFiles := tests.GetTestMongoDBWithSSLDSN(t, "../")

		cfgStorage := config.NewStorage(&config.Config{
			Paths: config.Paths{TempDir: t.TempDir()},
		})

		c := New(cfgStorage)
		resp := c.GetInfoFromService(context.Background(), &agentv1.ServiceInfoRequest{
			Dsn:       mongoDBDSNWithSSL,
			Type:      inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE,
			Timeout:   durationpb.New(30 * time.Second),
			TextFiles: mongoDBTextFiles,
		}, rand.Uint32()) //nolint:gosec
		require.NotNil(t, resp)
		assert.Empty(t, resp.Error)
	})
}

func TestExtractValkeyVersion(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name            string
		input           string
		expectedVersion string
		expectedError   string
	}

	cases := []testCase{
		{
			name: "Valid Valkey version",
			input: `# Server
redis_version:7.2.4
server_name:valkey
valkey_version:8.1.1
valkey_release_stage:ga
redis_git_sha1:00000000
redis_git_dirty:0
redis_build_id:5beb99de11516a6b
server_mode:standalone
os:Linux 6.13.7-orbstack-00283-g9d1400e7e9c6 aarch64
arch_bits:64
monotonic_clock:POSIX clock_gettime
multiplexing_api:epoll
gcc_version:12.2.0
process_id:1
process_supervised:no
run_id:db51448e49fb73ce02ccbab88ff56f6eddef6a90
tcp_port:6379
`,
			expectedVersion: "8.1.1",
		},
		{
			name: "No Valkey version, but Redis version present",
			input: `# Server
redis_version:7.2.4
server_name:valkey
valkey_release_stage:ga
redis_git_sha1:00000000
redis_git_dirty:0
redis_build_id:5beb99de11516a6b
server_mode:standalone
os:Linux 6.13.7-orbstack-00283-g9d1400e7e9c6 aarch64
arch_bits:64
monotonic_clock:POSIX clock_gettime
multiplexing_api:epoll
gcc_version:12.2.0
process_id:1
process_supervised:no
run_id:db51448e49fb73ce02ccbab88ff56f6eddef6a90
tcp_port:6379
`,
			expectedVersion: "7.2.4",
		},
		{
			name:          "Empty INFO string",
			input:         "",
			expectedError: "failed to get Valkey version",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			extractedVersion, err := extractValkeyVersion(tc.input)
			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedVersion, extractedVersion)
			}
		})
	}
}
