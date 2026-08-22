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

package victoriametrics

import (
	"fmt"
	"testing"
	"time"

	config "github.com/percona/promconfig"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

// queryCounter counts the queries reform executes.
type queryCounter struct {
	queries int
}

func (c *queryCounter) Before(_ string, _ []any) {}

func (c *queryCounter) After(_ string, _ []any, _ time.Duration, _ error) {
	c.queries++
}

// addScrapeConfigsFixtures creates one Node with a pmm-agent running the given number of
// MySQL services, each monitored by its own mysqld_exporter.
func addScrapeConfigsFixtures(t *testing.T, q *reform.Querier, pmmAgentID string, services int) {
	t.Helper()

	nodeID := "/node_id/" + pmmAgentID
	structs := make([]reform.Struct, 0, 2+2*services)
	structs = append(
		structs,
		&models.Node{
			NodeID:   nodeID,
			NodeType: models.GenericNodeType,
			NodeName: "node-" + pmmAgentID,
			Address:  "1.2.3.4",
		},
		&models.Agent{
			AgentID:      pmmAgentID,
			AgentType:    models.PMMAgentType,
			RunsOnNodeID: &nodeID,
			Version:      new("3.9.0"),
		},
	)

	for i := range services {
		serviceID := fmt.Sprintf("/service_id/%s/%d", pmmAgentID, i)
		structs = append(
			structs,
			&models.Service{
				ServiceID:   serviceID,
				ServiceType: models.MySQLServiceType,
				ServiceName: fmt.Sprintf("mysql-%s-%d", pmmAgentID, i),
				NodeID:      nodeID,
				Address:     new("5.6.7.8"),
				Port:        new(uint16(3306)),
			},
			&models.Agent{
				AgentID:    fmt.Sprintf("/agent_id/%s/%d", pmmAgentID, i),
				AgentType:  models.MySQLdExporterType,
				PMMAgentID: &pmmAgentID,
				ServiceID:  &serviceID,
				ListenPort: new(uint16(12345)),
			},
		)
	}

	for _, str := range structs {
		require.NoError(t, q.Insert(str))
	}
}

// TestAddScrapeConfigsQueryCount pins the number of queries AddScrapeConfigs runs: it must
// not grow with the number of monitored services. Every exporter used to cost a Service, a
// Node and a pmm-agent lookup, which is what made a scrape config rebuild during a reconnect
// storm hold a connection for hundreds of round trips (PMM-15228).
func TestAddScrapeConfigsQueryCount(t *testing.T) {
	counter := &queryCounter{}
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, counter)

	resolutions := models.MetricsResolutions{HR: 5 * time.Second, MR: 10 * time.Second, LR: time.Minute}
	l := logrus.WithField("component", "victoriametrics-test")

	queries := make(map[int]int)
	for _, services := range []int{1, 10} {
		pmmAgentID := fmt.Sprintf("/agent_id/pmm-agent-%d", services)
		addScrapeConfigsFixtures(t, db.Querier, pmmAgentID, services)

		var cfg config.Config
		counter.queries = 0
		require.NoError(t, AddScrapeConfigs(l, &cfg, db.Querier, &resolutions, &pmmAgentID, false, false))
		queries[services] = counter.queries

		// sanity check: the exporters are actually in the generated config
		assert.Len(t, cfg.ScrapeConfigs, services*3)
	}

	t.Logf("queries per number of services: %v", queries)
	assert.Equal(t, queries[1], queries[10],
		"the number of queries must not depend on the number of monitored services: %v", queries)
	assert.LessOrEqual(t, queries[10], 5, "expected one query per entity kind: %v", queries)
}
