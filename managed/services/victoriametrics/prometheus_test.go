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
	"github.com/percona/pmm/utils/sqlmetrics"
)

const nodeAddress = "1.2.3.4"

// addScrapeConfigsFixtures creates one Node with a pmm-agent running the given number of MySQL
// services, each monitored by its own mysqld_exporter. Every service gets a distinct address and
// every exporter a distinct listen port, so a lookup that returns the wrong row shows up in the
// generated scrape configs.
func addScrapeConfigsFixtures(t *testing.T, q *reform.Querier, pmmAgentID string, services int, pushMetrics bool) {
	t.Helper()

	nodeID := "/node_id/" + pmmAgentID
	structs := make([]reform.Struct, 0, 2+2*services)
	structs = append(
		structs,
		&models.Node{
			NodeID:   nodeID,
			NodeType: models.GenericNodeType,
			NodeName: "node-" + pmmAgentID,
			Address:  nodeAddress,
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
				ServiceName: serviceName(pmmAgentID, i),
				NodeID:      nodeID,
				Address:     new(fmt.Sprintf("10.0.0.%d", i+1)),
				Port:        new(uint16(3306 + i)),
			},
			&models.Agent{
				AgentID:         fmt.Sprintf("/agent_id/%s/%d", pmmAgentID, i),
				AgentType:       models.MySQLdExporterType,
				PMMAgentID:      &pmmAgentID,
				ServiceID:       &serviceID,
				ListenPort:      new(exporterPort(i)),
				ExporterOptions: models.ExporterOptions{PushMetrics: pushMetrics},
			},
		)
	}

	for _, str := range structs {
		require.NoError(t, q.Insert(str))
	}
}

func serviceName(pmmAgentID string, i int) string {
	return fmt.Sprintf("mysql-%s-%d", pmmAgentID, i)
}

func exporterPort(i int) uint16 {
	return uint16(12345 + i)
}

// scrapedTargets maps every emitted target to the service_name label it carries.
func scrapedTargets(cfg *config.Config) map[string]string {
	res := make(map[string]string)
	for _, scfg := range cfg.ScrapeConfigs {
		for _, group := range scfg.ServiceDiscoveryConfig.StaticConfigs {
			for _, target := range group.Targets {
				res[target] = group.Labels["service_name"]
			}
		}
	}

	return res
}

// TestAddScrapeConfigsQueryCount pins the number of queries AddScrapeConfigs runs: it must not
// grow with the number of monitored services. Every exporter used to cost a Service, a Node and a
// pmm-agent lookup, which is what made a scrape config rebuild during a reconnect storm hold a
// connection for hundreds of round trips (PMM-15228). The target and label assertions guard the
// lookup keys, since a cache returning the wrong row would keep the query count flat too.
func TestAddScrapeConfigsQueryCount(t *testing.T) {
	reformL := sqlmetrics.NewReform("test", "test", t.Logf)
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reformL)

	resolutions := models.MetricsResolutions{HR: 5 * time.Second, MR: 10 * time.Second, LR: time.Minute}
	l := logrus.WithField("component", "victoriametrics-test")

	// BuildScrapeConfigForVMAgent - the path a pmm-agent state update takes - passes
	// pushMetrics=true, which resolves the host differently, so cover both.
	for _, pushMetrics := range []bool{false, true} {
		t.Run(fmt.Sprintf("push_metrics=%v", pushMetrics), func(t *testing.T) {
			queries := make(map[int]int)
			for _, services := range []int{1, 10} {
				pmmAgentID := fmt.Sprintf("/agent_id/pmm-agent-%v-%d", pushMetrics, services)
				addScrapeConfigsFixtures(t, db.Querier, pmmAgentID, services, pushMetrics)

				var cfg config.Config
				reformL.Reset()
				require.NoError(t, AddScrapeConfigs(l, &cfg, db.Querier, &resolutions, &pmmAgentID, pushMetrics, false))
				queries[services] = reformL.Requests()

				// Every service must appear with its own target and its own labels: a lookup
				// keyed by the wrong column would return another row and change these.
				host := nodeAddress
				if pushMetrics {
					host = models.LocalhostAddr
				}
				expected := make(map[string]string, services)
				for i := range services {
					expected[fmt.Sprintf("%s:%d", host, exporterPort(i))] = serviceName(pmmAgentID, i)
				}
				assert.Equal(t, expected, scrapedTargets(&cfg))
				assert.Len(t, cfg.ScrapeConfigs, services*3)
			}

			t.Logf("queries per number of services: %v", queries)
			assert.Equal(t, queries[1], queries[10],
				"the number of queries must not depend on the number of monitored services: %v", queries)
			assert.LessOrEqual(t, queries[10], 5, "expected one query per entity kind: %v", queries)
		})
	}
}
