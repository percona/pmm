// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package connectionchecker

import (
	"testing"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionChecker_Check(t *testing.T) {
	tests := []struct {
		name     string
		msg      *agentpb.CheckConnectionRequest
		expected string
	}{
		{
			name: "MySQL",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:  "root:root-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=1s",
				Type: inventorypb.ServiceType_MYSQL_SERVICE,
			},
		},
		{
			name: "MySQL wrong params",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:  "pmm-agent:pmm-agent-wrong-password@tcp(127.0.0.1:3306)/?clientFoundRows=true&parseTime=true&timeout=1s",
				Type: inventorypb.ServiceType_MYSQL_SERVICE,
			},
			expected: `Error 1045: Access denied for user 'pmm-agent'@'.+' \(using password: YES\)`,
		},
		{
			name: "PostgreSQL",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:  "postgres://pmm-agent:pmm-agent-password@127.0.0.1:15432/postgres?connect_timeout=1&sslmode=disable",
				Type: inventorypb.ServiceType_POSTGRESQL_SERVICE,
			},
		},
		{
			name: "PostgreSQL wrong params",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:  "postgres://pmm-agent:pmm-agent-wrong-password@127.0.0.1:15432/postgres?connect_timeout=1&sslmode=disable",
				Type: inventorypb.ServiceType_POSTGRESQL_SERVICE,
			},
			expected: `pq: password authentication failed for user "pmm-agent"`,
		},
		{
			name: "MongoDB",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:  "mongodb://root:root-password@127.0.0.1:27017/admin",
				Type: inventorypb.ServiceType_MONGODB_SERVICE,
			},
		},
		{
			name: "MongoDB wrong params",
			msg: &agentpb.CheckConnectionRequest{
				Dsn:  "mongodb://root:root-password-wrong@127.0.0.1:27017/admin",
				Type: inventorypb.ServiceType_MONGODB_SERVICE,
			},
			expected: `Authentication failed.`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			err := c.Check(tt.msg)
			if tt.expected == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Regexp(t, tt.expected, err.Error())
			}
		})
	}
}

func TestConnectionChecker_MongoDSNWithoutSlashShouldProduceError(t *testing.T) {
	c := New()
	err := c.Check(&agentpb.CheckConnectionRequest{
		Dsn:  "mongodb://root:root-password@127.0.0.1:27017?connectTimeoutMS=1000",
		Type: inventorypb.ServiceType_MONGODB_SERVICE,
	})

	assert.Error(t, err)
	assert.EqualError(t, err, "error parsing uri: must have a / before the query ?")
}
