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
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
)

type agentInfo struct {
	channel *Channel
	id      string
	kick    chan struct{}
}

// Registry keeps track of all connected pmm-agents.
type Registry struct {
	db         *reform.DB
	prometheus prometheus
	qanClient  qanClient

	rw     sync.RWMutex
	agents map[string]*agentInfo // id -> info

	sharedMetrics *sharedChannelMetrics
	mConnects     prom.Counter
	mDisconnects  *prom.CounterVec
	mRoundTrip    prom.Summary
	mClockDrift   prom.Summary
}

// NewRegistry creates a new registry with given database connection.
func NewRegistry(db *reform.DB, prometheus prometheus, qanClient qanClient) *Registry {
	r := &Registry{
		db:         db,
		prometheus: prometheus,
		qanClient:  qanClient,

		agents: make(map[string]*agentInfo),

		sharedMetrics: newSharedMetrics(),
		mConnects: prom.NewCounter(prom.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "connects_total",
			Help:      "A total number of pmm-agent connects.",
		}),
		mDisconnects: prom.NewCounterVec(prom.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "disconnects_total",
			Help:      "A total number of pmm-agent disconnects.",
		}, []string{"reason"}),
		mRoundTrip: prom.NewSummary(prom.SummaryOpts{
			Namespace:  prometheusNamespace,
			Subsystem:  prometheusSubsystem,
			Name:       "round_trip_seconds",
			Help:       "Round-trip time.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
		mClockDrift: prom.NewSummary(prom.SummaryOpts{
			Namespace:  prometheusNamespace,
			Subsystem:  prometheusSubsystem,
			Name:       "clock_drift_seconds",
			Help:       "Clock drift.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
	}

	// initialize metrics with labels
	r.mDisconnects.WithLabelValues("unknown")

	return r
}

// IsConnected returns true if pmm-agent with given ID is currently connected, false otherwise.
func (r *Registry) IsConnected(pmmAgentID string) bool {
	r.rw.RLock()
	agent := r.agents[pmmAgentID]
	r.rw.RUnlock()
	return agent != nil
}

// Run takes over pmm-agent gRPC stream and runs it until completion.
func (r *Registry) Run(stream agentpb.Agent_ConnectServer) error {
	r.mConnects.Inc()
	disconnectReason := "unknown"
	defer func() {
		r.mDisconnects.WithLabelValues(disconnectReason).Inc()
	}()

	ctx := stream.Context()
	l := logger.Get(ctx)
	agent, err := r.register(stream)
	if err != nil {
		disconnectReason = "auth"
		return err
	}
	defer func() {
		l.Infof("Disconnecting client: %s.", disconnectReason)
	}()

	// send first SetStateRequest concurrently with ping from agent
	go r.SendSetStateRequest(ctx, agent.id)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.ping(ctx, agent)

		case <-agent.kick:
			l.Warn("Kicked.")
			disconnectReason = "kicked"
			err = status.Errorf(codes.Aborted, "Another pmm-agent with ID %q connected to the server.", agent.id)
			return err

		case msg := <-agent.channel.Requests():
			if msg == nil {
				disconnectReason = "done"
				return agent.channel.Wait()
			}

			switch req := msg.Payload.(type) {
			case *agentpb.AgentMessage_Ping:
				agent.channel.SendResponse(&agentpb.ServerMessage{
					Id: msg.Id,
					Payload: &agentpb.ServerMessage_Pong{
						Pong: &agentpb.Pong{
							CurrentTime: ptypes.TimestampNow(),
						},
					},
				})

			case *agentpb.AgentMessage_StateChanged:
				if err := r.stateChanged(ctx, req.StateChanged); err != nil {
					l.Errorf("%+v", err)
				}

				agent.channel.SendResponse(&agentpb.ServerMessage{
					Id: msg.Id,
					Payload: &agentpb.ServerMessage_StateChanged{
						StateChanged: new(agentpb.StateChangedResponse),
					},
				})

			case *agentpb.AgentMessage_QanCollect:
				if err := r.qanCollect(ctx, req.QanCollect, agent.id); err != nil {
					l.Errorf("%+v", err)
				}

				agent.channel.SendResponse(&agentpb.ServerMessage{
					Id: msg.Id,
					Payload: &agentpb.ServerMessage_QanCollect{
						QanCollect: new(agentpb.QANCollectResponse),
					},
				})

			default:
				l.Warnf("Unexpected request payload: %s.", msg)
				disconnectReason = "unimplemented"
				return status.Error(codes.Unimplemented, "Unexpected request payload.")
			}
		}
	}
}

func (r *Registry) register(stream agentpb.Agent_ConnectServer) (*agentInfo, error) {
	ctx := stream.Context()
	l := logger.Get(ctx)
	md := agentpb.GetAgentConnectMetadata(ctx)
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
		channel: NewChannel(stream, r.sharedMetrics),
		id:      md.ID,
		kick:    make(chan struct{}),
	}
	r.agents[md.ID] = agent
	return agent, nil
}

func authenticate(md *agentpb.AgentConnectMetadata, q *reform.Querier) error {
	if md.ID == "" {
		return status.Error(codes.Unauthenticated, "Empty Agent ID.")
	}

	row := &models.Agent{AgentID: md.ID}
	if err := q.Reload(row); err != nil {
		if err == reform.ErrNoRows {
			return status.Errorf(codes.Unauthenticated, "No Agent with ID %q.", md.ID)
		}
		return errors.Wrap(err, "failed to find agent")
	}

	if row.AgentType != models.PMMAgentType {
		return status.Errorf(codes.Unauthenticated, "No pmm-agent with ID %q.", md.ID)
	}

	row.Version = &md.Version
	if err := q.Update(row); err != nil {
		return errors.Wrap(err, "failed to update agent")
	}
	return nil
}

// Kick disconnects pmm-agent with given ID.
func (r *Registry) Kick(ctx context.Context, pmmAgentID string) {
	// We do not check that pmmAgentID is in fact ID of existing pmm-agent because
	// it may be already deleted from the database, that's why we disconnect it.

	r.rw.Lock()
	defer r.rw.Unlock()

	l := logger.Get(ctx)
	agent := r.agents[pmmAgentID]
	if agent == nil {
		l.Infof("pmm-agent with ID %q is not connected.", pmmAgentID)
		return
	}
	l.Infof("pmm-agent with ID %q is connected, kicking.", pmmAgentID)
	delete(r.agents, pmmAgentID)
	close(agent.kick)
}

// ping sends Ping message to given Agent, waits for Pong and observes round-trip time and clock drift.
func (r *Registry) ping(ctx context.Context, agent *agentInfo) {
	l := logger.Get(ctx)
	start := time.Now()
	res := agent.channel.SendRequest(&agentpb.ServerMessage_Ping{
		Ping: new(agentpb.Ping),
	})
	if res == nil {
		return
	}
	roundtrip := time.Since(start)
	agentTime, err := ptypes.Timestamp(res.(*agentpb.AgentMessage_Pong).Pong.CurrentTime)
	if err != nil {
		l.Errorf("Failed to decode Pong.current_time: %s.", err)
		return
	}
	clockDrift := agentTime.Sub(start) - roundtrip/2
	if clockDrift < 0 {
		clockDrift = -clockDrift
	}
	l.Infof("Round-trip time: %s. Estimated clock drift: %s.", roundtrip, clockDrift)
	r.mRoundTrip.Observe(roundtrip.Seconds())
	r.mClockDrift.Observe(clockDrift.Seconds())
}

func (r *Registry) stateChanged(ctx context.Context, req *agentpb.StateChangedRequest) error {
	err := r.db.InTransaction(func(tx *reform.TX) error {
		agent := &models.Agent{AgentID: req.AgentId}
		if err := tx.Reload(agent); err != nil {
			return errors.Wrap(err, "failed to select Agent by ID")
		}

		agent.Status = req.Status.String()
		agent.ListenPort = pointer.ToUint16(uint16(req.ListenPort))
		return tx.Update(agent)
	})
	if err != nil {
		return err
	}

	return r.prometheus.UpdateConfiguration(ctx)
}

func (r *Registry) qanCollect(ctx context.Context, req *agentpb.QANCollectRequest, agentID string) error {
	// TODO we probably should not do that for every message
	agent := &models.Agent{AgentID: agentID}
	if err := r.db.Reload(agent); err != nil {
		return errors.Wrap(err, "failed to select Agent by ID")
	}

	return r.qanClient.Collect(ctx, req.Message, agent)
}

// SendSetStateRequest sends SetStateRequest to pmm-agent with given ID.
func (r *Registry) SendSetStateRequest(ctx context.Context, pmmAgentID string) {
	l := logger.Get(ctx)

	r.rw.RLock()
	agent := r.agents[pmmAgentID]
	r.rw.RUnlock()
	if agent == nil {
		l.Infof("SendSetStateRequest: pmm-agent with ID %q is not currently connected.", pmmAgentID)
		return
	}

	agents, err := models.AgentsRunningByPMMAgent(r.db.Querier, pmmAgentID)
	if err != nil {
		l.Errorf("Failed to collect agents: %s.", err)
		return
	}

	agentProcesses := make(map[string]*agentpb.SetStateRequest_AgentProcess)
	builtinAgents := make(map[string]*agentpb.SetStateRequest_BuiltinAgent)
	for _, row := range agents {
		if row.Disabled {
			continue
		}

		switch row.AgentType {
		case models.PMMAgentType:
			continue

		case models.NodeExporterType:
			nodes, err := models.NodesForAgent(r.db.Querier, row.AgentID)
			if err != nil {
				l.Error(err)
				return
			}
			if len(nodes) != 1 {
				l.Errorf("Expected exactly one Node, got %d.", len(nodes))
				return
			}
			agentProcesses[row.AgentID] = nodeExporterConfig(nodes[0], row)

		case models.MySQLdExporterType:
			services, err := models.ServicesForAgent(r.db.Querier, row.AgentID)
			if err != nil {
				l.Error(err)
				return
			}
			if len(services) != 1 {
				l.Errorf("Expected exactly one Service, got %d.", len(services))
				return
			}
			agentProcesses[row.AgentID] = mysqldExporterConfig(services[0], row)

		case models.QANMySQLPerfSchemaAgentType:
			services, err := models.ServicesForAgent(r.db.Querier, row.AgentID)
			if err != nil {
				l.Error(err)
				return
			}
			if len(services) != 1 {
				l.Errorf("Expected exactly one Services, got %d.", len(services))
				return
			}
			builtinAgents[row.AgentID] = qanMySQLPerfSchemaAgentConfig(services[0], row)

		case models.MongoDBExporterType:
			services, err := models.ServicesForAgent(r.db.Querier, row.AgentID)
			if err != nil {
				l.Error(err)
				return
			}
			if len(services) != 1 {
				l.Errorf("Expected exactly one Services, got %d.", len(services))
				return
			}
			agentProcesses[row.AgentID] = mongodbExporterConfig(services[0], row)

		default:
			l.Panicf("unhandled Agent type %s", row.AgentType)
		}
	}

	state := &agentpb.SetStateRequest{
		AgentProcesses: agentProcesses,
		BuiltinAgents:  builtinAgents,
	}
	l.Infof("SendSetStateRequest: %+v.", state)
	res := agent.channel.SendRequest(&agentpb.ServerMessage_SetState{
		SetState: state,
	})
	l.Infof("SetState response: %+v.", res)
}

// Describe implements prometheus.Collector.
func (r *Registry) Describe(ch chan<- *prom.Desc) {
	r.sharedMetrics.Describe(ch)
	r.mConnects.Describe(ch)
	r.mDisconnects.Describe(ch)
	r.mRoundTrip.Describe(ch)
	r.mClockDrift.Describe(ch)
}

// Collect implement prometheus.Collector.
func (r *Registry) Collect(ch chan<- prom.Metric) {
	r.sharedMetrics.Collect(ch)
	r.mConnects.Collect(ch)
	r.mDisconnects.Collect(ch)
	r.mRoundTrip.Collect(ch)
	r.mClockDrift.Collect(ch)
}

// check interfaces
var (
	_ prom.Collector = (*Registry)(nil)
)
