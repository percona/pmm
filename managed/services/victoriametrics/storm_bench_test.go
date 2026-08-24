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

// Reconnect-storm harness for PMM-15228.
//
// Reproduces the DB-bound half of StateUpdater.sendSetStateRequest: the same query
// sequence, the same per-call transaction opened by BuildScrapeConfigForVMAgent, and
// the same 5s stateChangeTimeout. The final agent.channel.SendAndWaitResponse is not
// reproduced -- channel is a concrete *channel.Channel and needs a live gRPC stream.
//
// Run:
//
//	PMM_STORM_TEST=1 go test ./managed/services/victoriametrics/ -run TestSetStateStorm -v -timeout 30m

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	config "github.com/percona/promconfig"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/models"
)

const (
	// Mirrors stateChangeTimeout in managed/services/agents/state.go.
	stormStateChangeTimeout = 5 * time.Second
	// Mirrors updateBatchDelay in managed/services/agents/state.go.
	stormUpdateBatchDelay = time.Second
	stormDBName           = "pmm-managed-storm"
)

// stormFixed enables the PMM-15228 mitigations in the mirrored code paths:
// per-call lookup caches and jittered retry backoff. Set STORM_FIXED=1.
var stormFixed = os.Getenv("STORM_FIXED") != ""

func stormEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func stormEnvInt(t *testing.T, key string, def int) int {
	t.Helper()
	v, err := strconv.Atoi(stormEnv(key, strconv.Itoa(def)))
	require.NoError(t, err)
	return v
}

// stormResult holds one sweep row.
type stormResult struct {
	pool          int
	attempts      int64
	failures      int64
	p50, p95, p99 time.Duration
	waitCount     int64
	waitDuration  time.Duration
	inUse, idle   int
}

func TestSetStateStorm(t *testing.T) {
	if os.Getenv("PMM_STORM_TEST") == "" {
		t.Skip("set PMM_STORM_TEST=1 to run the reconnect-storm harness")
	}

	var (
		addr     = stormEnv("STORM_PG_ADDR", "127.0.0.1:55432")
		nAgents  = stormEnvInt(t, "STORM_AGENTS", 300)
		duration = time.Duration(stormEnvInt(t, "STORM_SECONDS", 30)) * time.Second
		pools    = stormEnv("STORM_POOLS", "50,100,200,300")
	)

	sqlDB := stormSetupDB(t, addr)
	defer sqlDB.Close()

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	agentIDs := stormSeed(t, db, nAgents)
	t.Logf("seeded %d pmm-agents (%d rows in agents table)", len(agentIDs), stormCountAgents(t, db))

	poolSizes := stormParsePools(t, pools)
	results := make([]stormResult, 0, len(poolSizes))
	for _, p := range poolSizes {
		results = append(results, stormRun(t, sqlDB, db, agentIDs, p, duration))
	}

	t.Log("")
	t.Log("=== PMM-15228 reconnect storm: pool sweep ===")
	t.Logf("agents=%d  duration=%s  stateChangeTimeout=%s", nAgents, duration, stormStateChangeTimeout)
	t.Logf("%-6s %9s %9s %8s %10s %10s %11s %14s", "pool", "attempts", "failures", "fail%", "p50", "p95", "poolWaits", "poolWaitTotal")
	for _, r := range results {
		failPct := 0.0
		if r.attempts > 0 {
			failPct = float64(r.failures) / float64(r.attempts) * 100
		}
		t.Logf("%-6d %9d %9d %7.1f%% %10s %10s %11d %14s",
			r.pool, r.attempts, r.failures, failPct,
			r.p50.Round(time.Millisecond), r.p95.Round(time.Millisecond),
			r.waitCount, r.waitDuration.Round(time.Millisecond))
	}
}

// TestStormScrapeConfigGolden dumps the generated scrape config for a seeded agent so the
// cached and uncached implementations can be diffed:
//
//	PMM_STORM_TEST=1 STORM_GOLDEN=/tmp/new.yaml go test ... -run TestStormScrapeConfigGolden -count=1
//	git stash push managed/services/victoriametrics/prometheus.go
//	PMM_STORM_TEST=1 STORM_GOLDEN=/tmp/old.yaml go test ... -run TestStormScrapeConfigGolden -count=1
//	git stash pop && diff /tmp/old.yaml /tmp/new.yaml
func TestStormScrapeConfigGolden(t *testing.T) {
	if os.Getenv("PMM_STORM_TEST") == "" {
		t.Skip("set PMM_STORM_TEST=1 to run the reconnect-storm harness")
	}
	out := os.Getenv("STORM_GOLDEN")
	if out == "" {
		t.Skip("set STORM_GOLDEN=<path> to dump the generated scrape config")
	}

	sqlDB := stormSetupDB(t, stormEnv("STORM_PG_ADDR", "127.0.0.1:55432"))
	defer sqlDB.Close()

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	ids := stormSeed(t, db, 3)

	var buf []byte
	for _, id := range ids {
		cfg, err := stormBuildScrapeConfig(t.Context(), db, id)
		require.NoError(t, err)
		buf = append(buf, []byte("--- agent config ---\n")...)
		buf = append(buf, cfg...)
	}
	require.NoError(t, os.WriteFile(out, buf, 0o600))
	t.Logf("wrote %d bytes to %s", len(buf), out)
}

func stormParsePools(t *testing.T, s string) []int {
	t.Helper()
	parts := splitComma(s)
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		v, err := strconv.Atoi(part)
		require.NoError(t, err)
		out = append(out, v)
	}
	return out
}

func splitComma(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// stormSetupDB recreates a throwaway database and runs migrations.
func stormSetupDB(t *testing.T, addr string) *sql.DB {
	t.Helper()
	ctx := t.Context()

	admin, err := models.OpenDB(models.SetupDBParams{Address: addr, Username: "postgres"})
	require.NoError(t, err)
	_, err = admin.ExecContext(ctx, `DROP DATABASE IF EXISTS "`+stormDBName+`"`)
	require.NoError(t, err)
	_, err = admin.ExecContext(ctx, `CREATE DATABASE "`+stormDBName+`"`)
	require.NoError(t, err)
	require.NoError(t, admin.Close())

	params := models.SetupDBParams{
		Address:       addr,
		Name:          stormDBName,
		Username:      "postgres",
		SetupFixtures: models.SetupFixtures,
	}
	sqlDB, err := models.OpenDB(params)
	require.NoError(t, err)
	_, err = models.SetupDB(ctx, sqlDB, params)
	require.NoError(t, err)
	return sqlDB
}

// stormSeed creates nAgents pmm-agents, each with a node, a postgres service and the
// exporter/QAN/vmagent rows a push-mode agent carries.
func stormSeed(t *testing.T, db *reform.DB, nAgents int) []string {
	t.Helper()
	ids := make([]string, 0, nAgents)

	err := db.InTransactionContext(t.Context(), nil, func(tx *reform.TX) error {
		q := tx.Querier
		for i := range nAgents {
			node, err := models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
				NodeName: fmt.Sprintf("storm-node-%d", i),
				Address:  fmt.Sprintf("10.0.%d.%d", i/250, i%250+1),
			})
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

			pmmAgent, err := models.CreatePMMAgent(q, node.NodeID, nil)
			if err != nil {
				return fmt.Errorf("pmm-agent %d: %w", i, err)
			}
			version := "3.9.1"
			pmmAgent.Version = &version
			err = q.Update(pmmAgent)
			if err != nil {
				return fmt.Errorf("pmm-agent version %d: %w", i, err)
			}

			_, err = models.CreateAgent(q, models.VMAgentType, &models.CreateAgentParams{
				PMMAgentID: pmmAgent.AgentID,
				NodeID:     node.NodeID,
			})
			if err != nil {
				return fmt.Errorf("vmagent %d: %w", i, err)
			}

			_, err = models.CreateAgent(q, models.NodeExporterType, &models.CreateAgentParams{
				PMMAgentID: pmmAgent.AgentID,
				NodeID:     node.NodeID,
			})
			if err != nil {
				return fmt.Errorf("node_exporter %d: %w", i, err)
			}

			port := uint16(5432)
			addr := "127.0.0.1"
			svc, err := models.AddNewService(q, models.PostgreSQLServiceType, &models.AddDBMSServiceParams{
				ServiceName: fmt.Sprintf("storm-pg-%d", i),
				NodeID:      node.NodeID,
				Address:     &addr,
				Port:        &port,
			})
			if err != nil {
				return fmt.Errorf("service %d: %w", i, err)
			}

			for _, at := range []models.AgentType{
				models.PostgresExporterType,
				models.QANPostgreSQLPgStatementsAgentType,
			} {
				_, err = models.CreateAgent(q, at, &models.CreateAgentParams{
					PMMAgentID: pmmAgent.AgentID,
					ServiceID:  svc.ServiceID,
					Username:   "pmm",
					Password:   "pmm",
				})
				if err != nil {
					return fmt.Errorf("agent %s %d: %w", at, i, err)
				}
			}

			ids = append(ids, pmmAgent.AgentID)
		}
		return nil
	})
	require.NoError(t, err)

	// FindAgentsForScrapeConfig requires push_metrics and a listen_port; a real agent
	// sets these when it reports status, which does not happen here. push_metrics lives
	// inside the exporter_options JSONB (see pushMetricsTrue in agent_helpers.go).
	_, err = db.Exec(`
		UPDATE agents
		SET listen_port = 42001,
		    exporter_options = jsonb_set(COALESCE(exporter_options, '{}'::jsonb), '{push_metrics}', 'true')
		WHERE agent_type <> 'pmm-agent'`)
	require.NoError(t, err)

	return ids
}

func stormCountAgents(t *testing.T, db *reform.DB) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM agents`).Scan(&n))
	return n
}

// stormSetStateDBWork mirrors the DB sequence of sendSetStateRequest for one pmm-agent.
// When fixed is true it mirrors the per-call lookup caches added to state.go.
func stormSetStateDBWork(ctx context.Context, db *reform.DB, pmmAgentID string, fixed bool) error {
	pmmAgent, err := models.FindAgentByID(db.Querier, pmmAgentID)
	if err != nil {
		return err
	}
	_, err = models.GetSettings(db.Querier)
	if err != nil {
		return err
	}
	agents, err := models.FindAgents(db.Querier, models.AgentFilters{PMMAgentID: pmmAgentID})
	if err != nil {
		return err
	}

	nodeCache := make(map[string]*models.Node)
	serviceCache := make(map[string]*models.Service)
	// The lookups are performed for their query cost and cache population; the rows
	// themselves are not needed here, so only the error is returned.
	getNode := func(id string) error {
		if fixed {
			if _, ok := nodeCache[id]; ok {
				return nil
			}
		}
		n, err := models.FindNodeByID(db.Querier, id)
		if err != nil {
			return err
		}
		nodeCache[id] = n
		return nil
	}
	getService := func(id string) error {
		if fixed {
			if _, ok := serviceCache[id]; ok {
				return nil
			}
		}
		s, err := models.FindServiceByID(db.Querier, id)
		if err != nil {
			return err
		}
		serviceCache[id] = s
		return nil
	}

	for _, row := range agents {
		switch row.AgentType {
		case models.PMMAgentType:
			continue
		case models.VMAgentType:
			// The expensive one: opens its own transaction (victoriametrics.go).
			_, err = stormBuildScrapeConfig(ctx, db, pmmAgentID)
			if err != nil {
				return fmt.Errorf("cannot get agent scrape config for agent %s: %w", pmmAgentID, err)
			}
		default:
			if row.ServiceID != nil {
				err = getService(*row.ServiceID)
				if err != nil {
					return err
				}
			}
			if row.NodeID != nil {
				err = getNode(*row.NodeID)
				if err != nil {
					return err
				}
			}
			// state.go re-fetched the same node on every row and dropped the error.
			if pmmAgent.RunsOnNodeID != nil {
				_ = getNode(*pmmAgent.RunsOnNodeID)
			}
		}
	}
	return nil
}

// stormRetryDelay mirrors retryDelay in managed/services/agents/state.go.
func stormRetryDelay(failures int) time.Duration {
	if failures < 1 {
		failures = 1
	}
	d := min(time.Second<<min(failures-1, 5), 30*time.Second)
	half := int64(d) / 2
	return time.Duration(half + rand.Int64N(half))
}

// stormBuildScrapeConfig replicates Service.BuildScrapeConfigForVMAgent.
func stormBuildScrapeConfig(ctx context.Context, db *reform.DB, pmmAgentID string) ([]byte, error) {
	var cfg config.Config
	e := db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		settings, err := models.GetSettings(tx)
		if err != nil {
			return err
		}
		res := settings.MetricsResolutions
		id := pmmAgentID
		return AddScrapeConfigs(logrus.WithField("component", "storm"), &cfg, tx.Querier, &res, &id, true, false)
	})
	if e != nil {
		return nil, e
	}
	return yaml.Marshal(cfg)
}

// stormRun drives nAgents workers in the same retry shape as runStateChangeHandler.
func stormRun(t *testing.T, sqlDB *sql.DB, db *reform.DB, agentIDs []string, pool int, duration time.Duration) stormResult {
	t.Helper()

	sqlDB.SetMaxOpenConns(pool)
	sqlDB.SetMaxIdleConns(pool)
	// Drain stats from the previous sweep row.
	before := sqlDB.Stats()

	ctx, cancel := context.WithTimeout(t.Context(), duration)
	defer cancel()

	var (
		attempts, failures atomic.Int64
		mu                 sync.Mutex
		lat                []time.Duration
		wg                 sync.WaitGroup
	)

	for _, id := range agentIDs {
		wg.Go(func() {
			consecutive := 0
			for ctx.Err() == nil {
				// Batch delay, as in runStateChangeHandler.
				select {
				case <-ctx.Done():
					return
				case <-time.After(stormUpdateBatchDelay):
				}

				nCtx, nCancel := context.WithTimeout(ctx, stormStateChangeTimeout)
				start := time.Now()
				err := stormSetStateDBWork(nCtx, db, id, stormFixed)
				dur := time.Since(start)
				nCancel()

				attempts.Add(1)
				if err != nil {
					failures.Add(1)
					consecutive++
					if stormFixed {
						// Jittered backoff before re-requesting.
						select {
						case <-ctx.Done():
							return
						case <-time.After(stormRetryDelay(consecutive)):
						}
					}
					continue // otherwise: immediate re-request, as stock state.go does
				}
				consecutive = 0
				mu.Lock()
				lat = append(lat, dur)
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	after := sqlDB.Stats()
	slices.Sort(lat)
	pick := func(q float64) time.Duration {
		if len(lat) == 0 {
			return 0
		}
		i := int(float64(len(lat)) * q)
		if i >= len(lat) {
			i = len(lat) - 1
		}
		return lat[i]
	}

	r := stormResult{
		pool:         pool,
		attempts:     attempts.Load(),
		failures:     failures.Load(),
		p50:          pick(0.50),
		p95:          pick(0.95),
		p99:          pick(0.99),
		waitCount:    after.WaitCount - before.WaitCount,
		waitDuration: after.WaitDuration - before.WaitDuration,
		inUse:        after.InUse,
		idle:         after.Idle,
	}
	t.Logf("pool=%-4d attempts=%-7d failures=%-7d p50=%-9s p95=%-9s poolWaits=%-7d poolWaitTotal=%s",
		r.pool, r.attempts, r.failures, r.p50.Round(time.Millisecond), r.p95.Round(time.Millisecond),
		r.waitCount, r.waitDuration.Round(time.Millisecond))
	return r
}
