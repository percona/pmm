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
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	_ "github.com/lib/pq" // register postgres driver
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/agents"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

const changesBufferSize = 10

// PostgreSQLRTA extracts Real-Time Analytics data from PostgreSQL.
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

// New creates new PostgreSQLRTA service.
func New(params *Params, l *logrus.Entry) (*PostgreSQLRTA, error) {
	if params.DSN == "" {
		return nil, fmt.Errorf("empty DSN")
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

// Run extracts currently running DB sessions from PostgreSQL until ctx is canceled.
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
	defer db.Close() //nolint:errcheck

	db.SetMaxOpenConns(2)
	db.SetConnMaxLifetime(5 * time.Minute) //nolint:mnd

	if err = db.PingContext(ctx); err != nil {
		p.l.Errorf("Can't connect to PostgreSQL for RTA: %v", err)
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
				bucket, collectErr := p.collect(curCtx, db)
				if collectErr != nil {
					if collectErr == errMissingPgReadAllStats {
						p.l.Error(collectErr.Error())
						p.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_WAITING}
						return
					}
					p.l.Warnf("pg_stat_activity collection failed: %v", collectErr)
					return
				}

				select {
				case <-curCtx.Done():
					return
				default:
					if len(bucket) != 0 {
						p.changes <- agents.Change{RTAQueriesBucket: bucket}
					}
				}
			}(ctx)
		}
	}
}

func (p *PostgreSQLRTA) collect(ctx context.Context, db *sql.DB) ([]*rtav1.QueryData, error) {
	results, err := collectSessions(ctx, db)
	if err != nil {
		return nil, err
	}

	collectTime := timestamppb.New(time.Now())
	for _, query := range results {
		query.ServiceId = p.serviceID
		query.ServiceName = p.serviceName
		query.QueryCollectTime = collectTime
	}

	return results, nil
}

// Changes returns channel that should be read until it is closed.
func (p *PostgreSQLRTA) Changes() <-chan agents.Change {
	return p.changes
}

// Describe implements prometheus.Collector.
func (p *PostgreSQLRTA) Describe(_ chan<- *prometheus.Desc) {}

// Collect implement prometheus.Collector.
func (p *PostgreSQLRTA) Collect(_ chan<- prometheus.Metric) {}

var _ prometheus.Collector = (*PostgreSQLRTA)(nil)
