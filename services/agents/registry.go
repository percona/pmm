// pmm-managed
// Copyright (C) 2017 Percona LLC
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
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/go-sql-driver/mysql"
	"github.com/golang/protobuf/ptypes"
	api "github.com/percona/pmm/api/agent"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
)

const (
	// maximum time for connecting to the database
	sqlDialTimeout = 5 * time.Second
)

type agentInfo struct {
	channel *Channel
	id      string
	l       *logrus.Entry
	kick    chan struct{}
}

type Registry struct {
	db *reform.DB

	rw     sync.RWMutex
	agents map[string]*agentInfo

	sharedMetrics *sharedChannelMetrics
	mConnects     prometheus.Counter
	mDisconnects  *prometheus.CounterVec
	mLatency      prometheus.Summary
	mClockDrift   prometheus.Summary
}

func NewRegistry(db *reform.DB) *Registry {
	return &Registry{
		db:            db,
		agents:        make(map[string]*agentInfo),
		sharedMetrics: newSharedMetrics(),
		mConnects: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "connects_total",
			Help:      "A total number of pmm-agent connects.",
		}),
		mDisconnects: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "disconnects_total",
			Help:      "A total number of pmm-agent disconnects.",
		}, []string{"reason"}),
		mLatency: prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace:  prometheusNamespace,
			Subsystem:  prometheusSubsystem,
			Name:       "latency_seconds",
			Help:       "Ping latency.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
		mClockDrift: prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace:  prometheusNamespace,
			Subsystem:  prometheusSubsystem,
			Name:       "clock_drift_seconds",
			Help:       "Clock drift.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
	}
}

func (r *Registry) Run(stream api.Agent_ConnectServer) error {
	r.mConnects.Inc()
	disconnectReason := "unknown"
	defer func() {
		r.mDisconnects.WithLabelValues(disconnectReason).Inc()
	}()

	agent, err := r.register(stream)
	if err != nil {
		disconnectReason = "auth"
		return err
	}

	r.ping(agent)
	r.SendSetStateRequest(stream.Context(), agent.id)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.ping(agent)

		case <-agent.kick:
			agent.l.Warn("Kicked.")
			disconnectReason = "kicked"
			return nil

		case msg := <-agent.channel.Requests():
			if msg == nil {
				disconnectReason = "done"
				return agent.channel.Wait()
			}

			switch req := msg.Payload.(type) {
			case *api.AgentMessage_QanData:
				// TODO
				agent.channel.SendResponse(&api.ServerMessage{
					Id: msg.Id,
					Payload: &api.ServerMessage_QanData{
						QanData: new(api.QANDataResponse),
					},
				})
			default:
				agent.l.Warnf("Unexpected request: %s.", req)
				disconnectReason = "bad_request"
				return nil
			}
		}
	}
}

func (r *Registry) register(stream api.Agent_ConnectServer) (*agentInfo, error) {
	l := logger.Get(stream.Context())
	md := api.GetAgentConnectMetadata(stream.Context())
	if err := authenticate(&md, r.db.Querier); err != nil {
		l.Warnf("Failed to authenticate connected pmm-agent %+v.", md)
		return nil, err
	}
	l.Infof("Connected pmm-agent: %+v.", md)

	r.rw.Lock()
	defer r.rw.Unlock()

	if agent := r.agents[md.ID]; agent != nil {
		close(agent.kick)
	}

	agent := &agentInfo{
		channel: NewChannel(stream, l.WithField("component", "channel"), r.sharedMetrics),
		id:      md.ID,
		l:       l,
		kick:    make(chan struct{}),
	}
	r.agents[md.ID] = agent
	return agent, nil
}

func authenticate(md *api.AgentConnectMetadata, q *reform.Querier) error {
	if md.ID == "" {
		return status.Error(codes.Unauthenticated, "Empty Agent ID.")
	}

	row := &models.AgentRow{ID: md.ID}
	if err := q.Reload(row); err != nil {
		if err == reform.ErrNoRows {
			return status.Errorf(codes.Unauthenticated, "No Agent with ID %q.", md.ID)
		}
		return errors.Wrap(err, "failed to find agent")
	}

	if row.Type != models.PMMAgentType {
		return status.Errorf(codes.Unauthenticated, "No pmm-agent with ID %q.", md.ID)
	}

	row.Version = &md.Version
	if err := q.Update(row); err != nil {
		return errors.Wrap(err, "failed to update agent")
	}
	return nil
}

// Kick disconnects pmm-agent with given ID.
func (r *Registry) Kick(ctx context.Context, id string) {
	r.rw.Lock()
	defer r.rw.Unlock()

	l := logger.Get(ctx)
	agent := r.agents[id]
	if agent == nil {
		l.Infof("pmm-agent with ID %q is not connected.", id)
		return
	}
	l.Infof("pmm-agent with ID %q is connected, kicking.", id)
	delete(r.agents, id)
	close(agent.kick)
}

func (r *Registry) ping(agent *agentInfo) {
	start := time.Now()
	res := agent.channel.SendRequest(&api.ServerMessage_Ping{
		Ping: new(api.PingRequest),
	})
	if res == nil {
		return
	}
	latency := time.Since(start) / 2
	t, err := ptypes.Timestamp(res.(*api.AgentMessage_Ping).Ping.CurrentTime)
	if err != nil {
		agent.l.Errorf("Failed to decode PingResponse.current_time: %s.", err)
		return
	}
	clockDrift := t.Sub(start.Add(latency))
	agent.l.Infof("Latency: %s. Clock drift: %s.", latency, clockDrift)
	r.mLatency.Observe(latency.Seconds())
	r.mClockDrift.Observe(clockDrift.Seconds())
}

func (r *Registry) SendSetStateRequest(ctx context.Context, id string) {
	l := logger.Get(ctx)

	r.rw.RLock()
	agent := r.agents[id]
	r.rw.RUnlock()
	if agent == nil {
		l.Infof("pmm-agent with ID %q is not currently connected, ignoring state change.", id)
		return
	}

	// We assume that all agents running on that Node except pmm-agent with given ID are subagents.
	// FIXME That is just plain wrong. We should filter by type, exclude external exporters, etc.

	pmmAgent := &models.AgentRow{ID: id}
	if err := r.db.Reload(pmmAgent); err != nil {
		l.Errorf("pmm-agent with ID %q not found: %s.", id, err)
		return
	}
	if pmmAgent.Type != models.PMMAgentType {
		l.Panicf("Agent with ID %q has invalid type %q.", id, pmmAgent.Type)
		return
	}
	structs, err := r.db.FindAllFrom(models.AgentRowTable, "runs_on_node_id", pmmAgent.RunsOnNodeID)
	if err != nil {
		l.Errorf("Failed to collect agents: %s.", err)
		return
	}

	processes := make(map[string]*api.SetStateRequest_AgentProcess, len(structs))
	for _, str := range structs {
		row := str.(*models.AgentRow)
		if row.Disabled {
			continue
		}

		switch row.Type {
		case models.PMMAgentType:
			continue
		case models.NodeExporterAgentType:
			processes[row.ID] = r.nodeExporterConfig(row)
		case models.MySQLdExporterAgentType:
			processes[row.ID] = r.mysqldExporterConfig(row)
		default:
			l.Panicf("unhandled AgentRow type %s", row.Type)
		}
	}

	res := agent.channel.SendRequest(&api.ServerMessage_State{
		State: &api.SetStateRequest{
			AgentProcesses: processes,
		},
	})
	agent.l.Infof("%s", res)
}

func (r *Registry) nodeExporterConfig(agent *models.AgentRow) *api.SetStateRequest_AgentProcess {
	collectors := []string{
		"diskstats",
		"filefd",
		"filesystem",
		"loadavg",
		"meminfo_numa",
		"meminfo",
		"netdev",
		"netstat",
		"stat",
		"textfile",
		"time",
		"uname",
		"vmstat",
	}
	return &api.SetStateRequest_AgentProcess{
		Type: api.Type_NODE_EXPORTER,
		Args: []string{
			fmt.Sprintf("-collectors.enabled=%s", strings.Join(collectors, ",")),
		},
	}
}

func (r *Registry) mysqldExporterConfig(agent *models.AgentRow) *api.SetStateRequest_AgentProcess {
	args := []string{
		"-collect.binlog_size",
		"-collect.global_status",
		"-collect.global_variables",
		"-collect.info_schema.innodb_metrics",
		"-collect.info_schema.processlist",
		"-collect.info_schema.query_response_time",
		"-collect.info_schema.userstats",
		"-collect.perf_schema.eventswaits",
		"-collect.perf_schema.file_events",
		"-collect.slave_status",
		"-web.listen-address=:{{ .listen_port }}",
	}
	// TODO Make it configurable. Play safe for now.
	// args = append(args, "-collect.auto_increment.columns")
	// args = append(args, "-collect.info_schema.tables")
	// args = append(args, "-collect.info_schema.tablestats")
	// args = append(args, "-collect.perf_schema.indexiowaits")
	// args = append(args, "-collect.perf_schema.tableiowaits")
	// args = append(args, "-collect.perf_schema.tablelocks")
	sort.Strings(args)

	// TODO TLSConfig: "true", https://jira.percona.com/browse/PMM-1727
	// TODO Other parameters?
	cfg := mysql.NewConfig()
	cfg.User = pointer.GetString(agent.ServiceUsername)
	cfg.Passwd = pointer.GetString(agent.ServicePassword)
	cfg.Net = "tcp"
	// TODO cfg.Addr = net.JoinHostPort(*service.Address, strconv.Itoa(int(*service.Port)))
	cfg.Timeout = sqlDialTimeout
	dsn := cfg.FormatDSN()

	return &api.SetStateRequest_AgentProcess{
		Type: api.Type_MYSQLD_EXPORTER,
		Args: args,
		Env: []string{
			fmt.Sprintf("DATA_SOURCE_NAME=%s", dsn),
		},
	}
}

// Describe implements prometheus.Collector.
func (r *Registry) Describe(ch chan<- *prometheus.Desc) {
	r.sharedMetrics.Describe(ch)
	r.mConnects.Describe(ch)
	r.mDisconnects.Describe(ch)
}

// Collect implement prometheus.Collector.
func (r *Registry) Collect(ch chan<- prometheus.Metric) {
	r.sharedMetrics.Collect(ch)
	r.mConnects.Collect(ch)
	r.mDisconnects.Collect(ch)
}

// check interfaces
var (
	_ prometheus.Collector = (*Registry)(nil)
)
