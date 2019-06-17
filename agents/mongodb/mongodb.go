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

// Package mongodb runs built-in QAN Agent for MongoDB profiler.
package mongodb

import (
	"context"

	"github.com/percona/pmgo"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/qanpb"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm-agent/agents/mongodb/internal/profiler"
	"github.com/percona/pmm-agent/agents/mongodb/internal/report"
)

// MongoDB extracts performance data from Mongo op log.
type MongoDB struct {
	agentID string
	l       *logrus.Entry
	changes chan Change

	dialInfo *pmgo.DialInfo
	dialer   pmgo.Dialer
}

// Params represent Agent parameters.
type Params struct {
	DSN     string
	AgentID string
}

// Change represents Agent status change _or_ QAN collect request.
type Change struct {
	Status  inventorypb.AgentStatus
	Request *qanpb.CollectRequest
}

// New creates new MongoDB QAN service.
func New(params *Params, l *logrus.Entry) (*MongoDB, error) {
	// if dsn is incorrect we should exit immediately as this is not gonna correct itself
	dialInfo, err := pmgo.ParseURL(params.DSN)
	if err != nil {
		return nil, err
	}

	return newMongo(dialInfo, l, params), nil
}

func newMongo(dialInfo *pmgo.DialInfo, l *logrus.Entry, params *Params) *MongoDB {
	return &MongoDB{
		agentID:  params.AgentID,
		dialInfo: dialInfo,
		dialer:   pmgo.NewDialer(),

		l:       l,
		changes: make(chan Change, 10),
	}
}

// Run extracts performance data and sends it to the channel until ctx is canceled.
func (m *MongoDB) Run(ctx context.Context) {
	var prof Profiler

	defer func() {
		prof.Stop() //nolint:errcheck
		prof = nil
		m.changes <- Change{Status: inventorypb.AgentStatus_DONE}
		close(m.changes)
	}()

	m.changes <- Change{Status: inventorypb.AgentStatus_STARTING}

	prof = profiler.New(m.dialInfo, m.dialer, m.l, m, m.agentID)
	if err := prof.Start(); err != nil {
		m.changes <- Change{Status: inventorypb.AgentStatus_STOPPING}
		return
	}

	m.changes <- Change{Status: inventorypb.AgentStatus_RUNNING}

	<-ctx.Done()
	m.changes <- Change{Status: inventorypb.AgentStatus_STOPPING}
	return
}

// Changes returns channel that should be read until it is closed.
func (m *MongoDB) Changes() <-chan Change {
	return m.changes
}

// Write writes MetricsBuckets to pmm-managed
func (m *MongoDB) Write(r *report.Report) error {
	m.changes <- Change{Request: &qanpb.CollectRequest{MetricsBucket: r.Buckets}}
	return nil
}

type Profiler interface {
	Start() error
	Stop() error
	Status() map[string]string
}
