// Copyright (C) 2023 Percona LLC
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

// Package realtimeanalytics runs built-in Real-Time Analytics Agent for PostgreSQL.
package realtimeanalytics

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/lib/pq" // register SQL driver
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/agent/agents"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

const (
	changesBufferSize = 10
	agentAppName      = "pmm-rta-postgresql-agent"
)

// ErrInsufficientPrivileges is returned when the monitoring user lacks pg_read_all_stats.
var ErrInsufficientPrivileges = errors.New("monitoring user lacks pg_read_all_stats privilege")

// PostgreSQLRTA extracts Real-Time Analytics data from PostgreSQL pg_stat_activity.
type PostgreSQLRTA struct {
	agentID         string
	serviceID       string
	serviceName     string
	l               *logrus.Entry
	changes         chan agents.Change
	dsn             string
	collectInterval time.Duration
}

// Params represent Agent parameters.
type Params struct {
	AgentID         string
	DSN             string
	ServiceID       string
	ServiceName     string
	CollectInterval time.Duration
}

// New creates a new PostgreSQLRTA agent.
func New(params *Params, l *logrus.Entry) (*PostgreSQLRTA, error) {
	if params.DSN == "" {
		return nil, errors.New("empty DSN")
	}

	return &PostgreSQLRTA{
		agentID:         params.AgentID,
		serviceID:       params.ServiceID,
		serviceName:     params.ServiceName,
		dsn:             params.DSN,
		collectInterval: params.CollectInterval,
		l:               l,
		changes:         make(chan agents.Change, changesBufferSize),
	}, nil
}

// Run polls PostgreSQL and sends session data until ctx is canceled.
func (p *PostgreSQLRTA) Run(ctx context.Context) {
	p.l.Info("Starting PostgreSQL RTA agent")

	p.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}

	defer func() {
		p.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE}
		close(p.changes)
	}()

	db, err := sql.Open("postgres", p.dsn)
	if err != nil {
		p.l.Errorf("Can't run Real-Time Analytics agent, reason: %v", err)
		p.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
		return
	}

	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	defer func() {
		_ = db.Close()
	}()

	collector, err := newCollector(ctx, db, p.agentID, p.l)
	if err != nil {
		p.l.Errorf("Can't initialize PostgreSQL RTA collector: %v", err)
		p.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
		return
	}

	p.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING}

	ticker := time.NewTicker(p.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.l.Info("Stopping PostgreSQL RTA agent")
			p.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
			return
		case <-ticker.C:
			go func(curCtx context.Context) {
				queries, collectErr := collector.collectSessions(curCtx)
				if collectErr != nil {
					if errors.Is(collectErr, ErrInsufficientPrivileges) {
						p.l.Errorf("PostgreSQL RTA permission error: %v", collectErr)
					} else {
						p.l.Warnf("PostgreSQL session collection failed: %v", collectErr)
					}
					return
				}

				select {
				case <-curCtx.Done():
					return
				default:
					if len(queries) != 0 {
						for i := range queries {
							queries[i].ServiceId = p.serviceID
							queries[i].ServiceName = p.serviceName
						}
						p.changes <- agents.Change{RTAQueriesBucket: queries}
					}
				}
			}(ctx)
		}
	}
}

// Changes returns channel that should be read until it is closed.
func (p *PostgreSQLRTA) Changes() <-chan agents.Change {
	return p.changes
}

// Describe implements prometheus.Collector.
func (p *PostgreSQLRTA) Describe(_ chan<- *prometheus.Desc) {}

// Collect implements prometheus.Collector.
func (p *PostgreSQLRTA) Collect(_ chan<- prometheus.Metric) {}

var _ prometheus.Collector = (*PostgreSQLRTA)(nil)
