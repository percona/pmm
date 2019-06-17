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
)

func TestMongoRun(t *testing.T) {
	// setup
	l := logrus.WithField("component", "mongo-builtin-agent")
	p := &Params{DSN: "mongodb://root:root-password@127.0.0.1:27017/admin", AgentID: "/agent_id/test"}
	m, err := New(p, l)
	if err != nil {
		t.Fatal(err)
	}

	// run agent
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	go m.Run(ctx)

	// collect changes (only check statuses of agent)
	actualStatues := make([]inventorypb.AgentStatus, 0)
	for c := range m.Changes() {
		if c.Status != inventorypb.AgentStatus_AGENT_STATUS_INVALID {
			actualStatues = append(actualStatues, c.Status)
		}
	}

	// waiting agent for sendStopStatus
	<-ctx.Done()

	// check actual statuses with real lifecycle
	expectedStatuses := []inventorypb.AgentStatus{
		inventorypb.AgentStatus_STARTING,
		inventorypb.AgentStatus_RUNNING,
		inventorypb.AgentStatus_STOPPING,
		inventorypb.AgentStatus_DONE,
	}

	assert.Equal(t, expectedStatuses, actualStatues)
}
