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

package realtimeanalytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
	"github.com/percona/pmm/managed/models"
)

// A collect interval is only persisted when the request carried one -
// RTAOptionsFromRequest leaves it nil otherwise, which is what
// `pmm-admin inventory add agent rta-mysql-agent` without --collect-interval
// produces - so the session mapping must not dereference it blindly.
func TestConvertAgentToSession(t *testing.T) {
	t.Parallel()

	service := &models.Service{
		ServiceID:   "service-id",
		ServiceName: "service-name",
		ServiceType: models.MySQLServiceType,
		Cluster:     "cluster",
	}

	t.Run("WithCollectInterval", func(t *testing.T) {
		t.Parallel()

		agent := &models.Agent{
			AgentID:    "agent-id",
			AgentType:  models.RTAMySQLAgentType,
			RTAOptions: models.RTAOptions{CollectInterval: new(5 * time.Second)},
		}

		session := (&Service{}).convertAgentToSession(agent, service)

		require.NotNil(t, session.CollectInterval)
		assert.Equal(t, 5*time.Second, session.CollectInterval.AsDuration())
	})

	t.Run("WithoutCollectInterval", func(t *testing.T) {
		t.Parallel()

		agent := &models.Agent{
			AgentID:   "agent-id",
			AgentType: models.RTAMySQLAgentType,
		}

		var session *rtav1.Session
		require.NotPanics(t, func() {
			session = (&Service{}).convertAgentToSession(agent, service)
		})

		require.NotNil(t, session)
		assert.Nil(t, session.CollectInterval, "an interval that was never set must be omitted, not invented")
		assert.Equal(t, "service-id", session.ServiceId)
	})
}
