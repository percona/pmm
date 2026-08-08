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

package agents

import (
	"context"
	"testing"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/logger"
)

type haServiceStub struct{}

func (haServiceStub) Params() *models.HAParams { return &models.HAParams{} }

func newTestRegistry() *Registry {
	return &Registry{
		agents:    make(map[string]*pmmAgentInfo),
		roster:    newRoster(nil),
		haService: haServiceStub{},
		mDisconnects: prom.NewCounterVec(prom.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "disconnects_total",
			Help:      "A total number of pmm-agent disconnects.",
		}, []string{"reason"}),
	}
}

func TestUnregister(t *testing.T) {
	t.Parallel()

	const agentID = "/agent_id/00000000-0000-4000-8000-000000000001"

	ctx := logger.SetEntry(context.Background(), logrus.WithField("test", t.Name()))

	t.Run("removes the current connection", func(t *testing.T) {
		t.Parallel()

		r := newTestRegistry()
		current := &pmmAgentInfo{id: agentID}
		r.agents[agentID] = current

		assert.Same(t, current, r.unregister(ctx, agentID, "done", current))
		assert.Empty(t, r.agents)
	})

	t.Run("removes any connection when none is given", func(t *testing.T) {
		t.Parallel()

		r := newTestRegistry()
		current := &pmmAgentInfo{id: agentID}
		r.agents[agentID] = current

		assert.Same(t, current, r.unregister(ctx, agentID, "kick", nil))
		assert.Empty(t, r.agents)
	})

	t.Run("keeps the connection that superseded a stale one", func(t *testing.T) {
		// A silently dropped connection can take minutes to die. By then the agent has
		// reconnected and registered again, and the stale handler must not evict it,
		// otherwise the agent stays connected while pmm-managed reports it as
		// disconnected forever. See PMM-15310.
		t.Parallel()

		r := newTestRegistry()
		stale := &pmmAgentInfo{id: agentID}
		current := &pmmAgentInfo{id: agentID}
		r.agents[agentID] = current

		assert.Nil(t, r.unregister(ctx, agentID, "done", stale))
		assert.Same(t, current, r.agents[agentID])
	})

	t.Run("does nothing for an unknown agent", func(t *testing.T) {
		t.Parallel()

		r := newTestRegistry()

		assert.Nil(t, r.unregister(ctx, agentID, "done", &pmmAgentInfo{id: agentID}))
		assert.Empty(t, r.agents)
	})
}
