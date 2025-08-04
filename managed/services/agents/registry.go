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

// Package agents contains business logic of working with pmm-agent.
package agents

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	prom "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents/channel"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/version"
)

const (
	prometheusNamespace = "pmm_managed"
	prometheusSubsystem = "agents"
)

var (
	mSentDesc = prom.NewDesc(
		prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "messages_sent_total"),
		"A total number of messages sent to pmm-agent.",
		[]string{"agent_id"},
		nil)
	mRecvDesc = prom.NewDesc(
		prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "messages_received_total"),
		"A total number of messages received from pmm-agent.",
		[]string{"agent_id"},
		nil)
	mResponsesDesc = prom.NewDesc(
		prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "messages_response_queue_length"),
		"The current length of the response queue.",
		[]string{"agent_id"},
		nil)
	mRequestsDesc = prom.NewDesc(
		prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "messages_request_queue_length"),
		"The current length of the request queue.",
		[]string{"agent_id"},
		nil)
)

type pmmAgentInfo struct {
	channel         *channel.Channel
	id              string
	stateChangeChan chan struct{}
	kickChan        chan struct{}
}

// Registry keeps track of all connected pmm-agents.
type Registry struct {
	db *reform.DB

	rw     sync.RWMutex
	agents map[string]*pmmAgentInfo // id -> info

	roster *roster

	mConnects    prom.Counter
	mDisconnects *prom.CounterVec
	mRoundTrip   prom.Summary
	mClockDrift  prom.Summary
	mAgents      prom.GaugeFunc

	isExternalVM bool
}

// NewRegistry creates a new registry with given database connection.
func NewRegistry(db *reform.DB, externalVMChecker victoriaMetricsParams) *Registry {
	agents := make(map[string]*pmmAgentInfo)
	r := &Registry{
		db: db,

		agents: agents,

		roster: newRoster(db),

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

		isExternalVM: externalVMChecker.ExternalVM(),
	}

	r.mAgents = prom.NewGaugeFunc(prom.GaugeOpts{
		Namespace: prometheusNamespace,
		Subsystem: prometheusSubsystem,
		Name:      "connected",
		Help:      "The current number of connected pmm-agents.",
	}, func() float64 {
		r.rw.Lock()
		defer r.rw.Unlock()

		return float64(len(agents))
	})

	// initialize metrics with labels
	r.mDisconnects.WithLabelValues("unknown")

	return r
}

// IsConnected returns true if pmm-agent with given ID is currently connected, false otherwise.
func (r *Registry) IsConnected(pmmAgentID string) bool {
	_, err := r.get(pmmAgentID)
	return err == nil
}

func (r *Registry) register(stream agentv1.AgentService_ConnectServer) (*pmmAgentInfo, error) {
	ctx := stream.Context()
	l := logger.Get(ctx)
	r.mConnects.Inc()

	agentMD, err := agentv1.ReceiveAgentConnectMetadata(stream)
	if err != nil {
		return nil, err
	}
	var node *models.Node
	err = r.db.InTransaction(func(tx *reform.TX) error {
		node, err = r.authenticate(agentMD, tx.Querier)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		l.Warnf("Failed to authenticate connected pmm-agent %+v.", agentMD)
		return nil, err
	}
	l.Infof("Connected pmm-agent: %+v.", agentMD)

	serverMD := agentv1.ServerConnectMetadata{
		AgentRunsOnNodeID: node.NodeID,
		NodeName:          node.NodeName,
		ServerVersion:     version.Version,
	}
	l.Debugf("Sending metadata: %+v.", serverMD)
	if err = agentv1.SendServerConnectMetadata(stream, &serverMD); err != nil {
		return nil, err
	}

	currentAgent, err := r.get(agentMD.ID)
	if err == nil {
		// pmm-agent with the same ID can still be connected in two cases:
		//   1. Someone uses the same ID by mistake, glitch, or malicious intent.
		//   2. pmm-agent detects broken connection and reconnects,
		//      but pmm-managed still thinks that the previous connection is okay.
		// If agent respond with pong new connection is not established,
		// so we return AlreadyExists error. Otherwise we kick the previous connection
		// and proceed with the new one.
		pong, err := r.ping(ctx, currentAgent)
		if pong {
			return nil, status.Errorf(codes.AlreadyExists, "pmm-agent with ID %q is already connected.", agentMD.ID)
		}

		l.Warningf("Failed to ping pmm-agent with ID %q: %w", agentMD.ID, err)
		r.Kick(ctx, agentMD.ID)
		l.Warningf("pmm-agent with ID %q is kicked.", agentMD.ID)
	}
	r.rw.Lock()
	defer r.rw.Unlock()

	agent := &pmmAgentInfo{
		channel:         channel.New(ctx, stream),
		id:              agentMD.ID,
		stateChangeChan: make(chan struct{}, 1),
		kickChan:        make(chan struct{}),
	}
	r.agents[agentMD.ID] = agent
	return agent, nil
}

func (r *Registry) authenticate(md *agentv1.AgentConnectMetadata, q *reform.Querier) (*models.Node, error) {
	if md.ID == "" {
		return nil, status.Error(codes.PermissionDenied, "Empty Agent ID.")
	}

	// Get agent ID
	agent, err := models.FindAgentByID(q, md.ID)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, status.Errorf(codes.PermissionDenied, "No Agent with ID %q.", md.ID)
		}
		return nil, fmt.Errorf("failed to find agent: %w", err)
	}

	if agent.AgentType != models.PMMAgentType {
		return nil, status.Errorf(codes.PermissionDenied, "No pmm-agent with ID %q.", md.ID)
	}

	runsOnNodeID := pointer.GetString(agent.RunsOnNodeID)
	if runsOnNodeID == "" {
		return nil, status.Errorf(codes.PermissionDenied, "Can't get 'runs_on_node_id' for pmm-agent with ID %q.", md.ID)
	}

	// Get agent version
	agentVersion, err := version.Parse(md.Version)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Can't parse 'version' for pmm-agent with ID %q.", md.ID)
	}

	if err := r.addOrRemoveVMAgent(q, md.ID, runsOnNodeID); err != nil {
		return nil, err
	}

	if err := r.addNomadAgentToPMMAgent(q, md.ID, runsOnNodeID, agentVersion); err != nil {
		return nil, err
	}

	agent.Version = &md.Version
	if err := q.Update(agent); err != nil {
		return nil, fmt.Errorf("failed to update agent: %w", err)
	}

	node, err := models.FindNodeByID(q, runsOnNodeID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Can't retrieve node ID for pmm-agent with ID %q.", md.ID)
	}

	return node, nil
}

// unregister removes pmm-agent with given ID from the registry.
func (r *Registry) unregister(pmmAgentID, disconnectReason string) *pmmAgentInfo {
	r.mDisconnects.WithLabelValues(disconnectReason).Inc()

	r.rw.Lock()
	defer r.rw.Unlock()

	// We do not check that pmmAgentID is in fact ID of existing pmm-agent because
	// it may be already deleted from the database, that's why we unregister it.

	agent := r.agents[pmmAgentID]
	if agent == nil {
		return nil
	}

	delete(r.agents, pmmAgentID)
	r.roster.clear(pmmAgentID)
	return agent
}

// ping sends Ping message to given Agent, waits for Pong and observes round-trip time and clock drift.
// Returns true if pong is received, false if there is no pong or error occurred.
func (r *Registry) ping(ctx context.Context, agent *pmmAgentInfo) (bool, error) {
	l := logger.Get(ctx)
	start := time.Now()
	resp, err := agent.channel.SendAndWaitResponse(&agentv1.Ping{})
	if err != nil {
		return false, err
	}
	if resp == nil {
		return false, errors.New("pong is not received, response is nil")
	}
	roundtrip := time.Since(start)
	agentTime := resp.(*agentv1.Pong).CurrentTime.AsTime() //nolint:forcetypeassert
	clockDrift := agentTime.Sub(start) - roundtrip/2
	if clockDrift < 0 {
		clockDrift = -clockDrift
	}
	l.Debugf("Round-trip time: %s. Estimated clock drift: %s.", roundtrip, clockDrift)
	r.mRoundTrip.Observe(roundtrip.Seconds())
	r.mClockDrift.Observe(clockDrift.Seconds())
	return true, nil
}

// addOrRemoveVMAgent - creates vmAgent agentType if pmm-agent's version supports it and agent not exists yet,
// otherwise ensures that vmAgent not exist for pmm-agent and pmm-agent's agents don't have push_metrics mode,
// removes it if needed.
func (r *Registry) addOrRemoveVMAgent(q *reform.Querier, pmmAgentID, runsOnNodeID string) error {
	return r.addVMAgentToPMMAgent(q, pmmAgentID, runsOnNodeID)
}

func (r *Registry) addVMAgentToPMMAgent(q *reform.Querier, pmmAgentID, runsOnNodeID string) error {
	if runsOnNodeID == "pmm-server" && !r.isExternalVM {
		return nil
	}
	vmAgentType := models.VMAgentType
	vmAgent, err := models.FindAgents(q, models.AgentFilters{PMMAgentID: pmmAgentID, AgentType: &vmAgentType})
	if err != nil {
		return status.Errorf(codes.Internal, "Can't get 'vmAgent' for pmm-agent with ID %q", pmmAgentID)
	}
	if len(vmAgent) == 0 {
		if _, err := models.CreateAgent(q, models.VMAgentType, &models.CreateAgentParams{
			PMMAgentID: pmmAgentID,
			NodeID:     runsOnNodeID,
			ExporterOptions: models.ExporterOptions{
				PushMetrics: true,
			},
		}); err != nil {
			return fmt.Errorf("can't create 'vmAgent' for pmm-agent with ID %q: %w", pmmAgentID, err)
		}
	}
	return nil
}

func (r *Registry) addNomadAgentToPMMAgent(q *reform.Querier, pmmAgentID, runsOnNodeID string, pmmAgentVersion *version.Parsed) error {
	if !pmmAgentVersion.IsFeatureSupported(version.NomadAgentSupportVersion) {
		return nil
	}
	nomadClient, err := models.FindAgents(q, models.AgentFilters{PMMAgentID: pmmAgentID, AgentType: pointer.To(models.NomadAgentType)})
	if err != nil {
		return status.Errorf(codes.Internal, "Can't get 'nomadClient' for pmm-agent with ID %q", pmmAgentID)
	}
	if len(nomadClient) == 0 {
		if _, err := models.CreateAgent(q, models.NomadAgentType, &models.CreateAgentParams{
			PMMAgentID: pmmAgentID,
			NodeID:     runsOnNodeID,
		}); err != nil {
			return fmt.Errorf("can't create 'nomadClient' for pmm-agent with ID %q: %w", pmmAgentID, err)
		}
	}
	return nil
}

// Kick unregisters and forcefully disconnects pmm-agent with given ID.
func (r *Registry) Kick(ctx context.Context, pmmAgentID string) {
	agent := r.unregister(pmmAgentID, "kick")
	if agent == nil {
		return
	}

	l := logger.Get(ctx)
	l.Debugf("pmm-agent with ID %s will be kicked in a moment.", pmmAgentID)

	// see Run method
	close(agent.kickChan)

	// Do not close agent.stateChangeChan to avoid breaking RequestStateUpdate;
	// closing agent.kickChan is enough to exit runStateChangeHandler goroutine.
}

func (r *Registry) get(pmmAgentID string) (*pmmAgentInfo, error) {
	r.rw.RLock()
	pmmAgent := r.agents[pmmAgentID]
	r.rw.RUnlock()
	if pmmAgent == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "pmm-agent with ID %s is not currently connected", pmmAgentID)
	}
	return pmmAgent, nil
}

// Describe implements prometheus.Collector.
func (r *Registry) Describe(ch chan<- *prom.Desc) {
	r.mConnects.Describe(ch)
	r.mDisconnects.Describe(ch)
	r.mRoundTrip.Describe(ch)
	r.mClockDrift.Describe(ch)
	r.mAgents.Describe(ch)
}

// Collect implement prometheus.Collector.
func (r *Registry) Collect(ch chan<- prom.Metric) {
	r.rw.RLock()

	for _, agent := range r.agents {
		m := agent.channel.Metrics()

		ch <- prom.MustNewConstMetric(mSentDesc, prom.CounterValue, m.Sent, agent.id)
		ch <- prom.MustNewConstMetric(mRecvDesc, prom.CounterValue, m.Recv, agent.id)
		ch <- prom.MustNewConstMetric(mResponsesDesc, prom.GaugeValue, m.Responses, agent.id)
		ch <- prom.MustNewConstMetric(mRequestsDesc, prom.GaugeValue, m.Requests, agent.id)
	}
	r.rw.RUnlock()

	r.mAgents.Collect(ch)
	r.mConnects.Collect(ch)
	r.mDisconnects.Collect(ch)
	r.mRoundTrip.Collect(ch)
	r.mClockDrift.Collect(ch)
}

// KickAll sends a signal to all registered agents in the registry to perform a kick action.
func (r *Registry) KickAll(ctx context.Context) {
	for _, agentInfo := range r.agents {
		r.Kick(ctx, agentInfo.id)
	}
}

// check interfaces.
var (
	_ prom.Collector = (*Registry)(nil)
)
