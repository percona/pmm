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

package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMongoRun(t *testing.T) {
	params := &Params{
		DSN:     "mongodb://root:root-password@127.0.0.1:27017/admin",
		AgentID: "/agent_id/test",
	}
	m, err := New(params, logrus.WithField("test", t.Name()))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go m.Run(ctx)

	// collect only status changes, skip QAN data
	var actual []inventorypb.AgentStatus
	for c := range m.Changes() {
		if c.Status != inventorypb.AgentStatus_AGENT_STATUS_INVALID {
			actual = append(actual, c.Status)
		}
	}

	expected := []inventorypb.AgentStatus{
		inventorypb.AgentStatus_STARTING,
		inventorypb.AgentStatus_RUNNING,
		inventorypb.AgentStatus_STOPPING,
		inventorypb.AgentStatus_DONE,
	}
	assert.Equal(t, expected, actual)
}
