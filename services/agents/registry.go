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

// Package agents contains business logic of working with pmm-agent.
package agents

import (
	"context"
	"fmt"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/agents/channel"
	"github.com/percona/pmm-managed/utils/logger"
)

const (
	// constants for delayed batch updates
	updateBatchDelay   = time.Second
	stateChangeTimeout = 5 * time.Second

	prometheusNamespace = "pmm_managed"
	prometheusSubsystem = "agents"
)

var (
	checkExternalExporterConnectionPMMVersion = version.MustParse("1.14.99")

	defaultActionTimeout      = ptypes.DurationProto(10 * time.Second)
	defaultQueryActionTimeout = ptypes.DurationProto(15 * time.Second) // should be less than checks.resultTimeout
	defaultPtActionTimeout    = ptypes.DurationProto(30 * time.Second) // Percona-toolkit action timeout

	mSentDesc = prom.NewDesc(
		prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "messages_sent_total"),
		"A total number of messages sent to pmm-agent.",
		[]string{"agent_id"},
		nil,
	)
	mRecvDesc = prom.NewDesc(
		prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "messages_received_total"),
		"A total number of messages received from pmm-agent.",
		[]string{"agent_id"},
		nil,
	)
	mResponsesDesc = prom.NewDesc(
		prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "messages_response_queue_length"),
		"The current length of the response queue.",
		[]string{"agent_id"},
		nil,
	)
	mRequestsDesc = prom.NewDesc(
		prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "messages_request_queue_length"),
		"The current length of the request queue.",
		[]string{"agent_id"},
		nil,
	)
)

type pmmAgentInfo struct {
	channel         *channel.Channel
	id              string
	stateChangeChan chan struct{}
	kick            chan struct{}
}

// Registry keeps track of all connected pmm-agents.
//
// TODO Split into several types https://jira.percona.com/browse/PMM-4932
type Registry struct {
	db        *reform.DB
	vmdb      prometheusService
	qanClient qanClient

	rw     sync.RWMutex
	agents map[string]*pmmAgentInfo // id -> info

	roster *roster

	mAgents      prom.GaugeFunc
	mConnects    prom.Counter
	mDisconnects *prom.CounterVec
	mRoundTrip   prom.Summary
	mClockDrift  prom.Summary
}

// NewRegistry creates a new registry with given database connection.
func NewRegistry(db *reform.DB, qanClient qanClient, vmdb prometheusService) *Registry {
	agents := make(map[string]*pmmAgentInfo)
	r := &Registry{
		db:        db,
		vmdb:      vmdb,
		qanClient: qanClient,

		agents: agents,

		roster: newRoster(),

		mAgents: prom.NewGaugeFunc(prom.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "connected",
			Help:      "The current number of connected pmm-agents.",
		}, func() float64 {
			return float64(len(agents))
		}),
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
	_, err := r.get(pmmAgentID)
	return err == nil
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

	// run pmm-agent state update loop for the current agent.
	go r.runStateChangeHandler(ctx, agent)

	r.RequestStateUpdate(ctx, agent.id)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.ping(ctx, agent)

		// see unregister and Kick methods
		case <-agent.kick:
			// already unregistered, no need to call unregister method
			l.Warn("Kicked.")
			disconnectReason = "kicked"
			err = status.Errorf(codes.Aborted, "Kicked.")
			return err

		case req := <-agent.channel.Requests():
			if req == nil {
				disconnectReason = "done"
				err = agent.channel.Wait()
				r.unregister(agent.id)
				if err != nil {
					l.Error(errors.WithStack(err))
				}
				return r.updateAgentStatusForChildren(ctx, agent.id, inventorypb.AgentStatus_DONE, 0)
			}

			switch p := req.Payload.(type) {
			case *agentpb.Ping:
				agent.channel.Send(&channel.ServerResponse{
					ID: req.ID,
					Payload: &agentpb.Pong{
						CurrentTime: ptypes.TimestampNow(),
					},
				})

			case *agentpb.StateChangedRequest:
				pprof.Do(ctx, pprof.Labels("request", "StateChangedRequest"), func(ctx context.Context) {
					if err := r.stateChanged(ctx, p); err != nil {
						l.Errorf("%+v", err)
					}

					agent.channel.Send(&channel.ServerResponse{
						ID:      req.ID,
						Payload: new(agentpb.StateChangedResponse),
					})
				})

			case *agentpb.QANCollectRequest:
				pprof.Do(ctx, pprof.Labels("request", "QANCollectRequest"), func(ctx context.Context) {
					if err := r.qanClient.Collect(ctx, p.MetricsBucket); err != nil {
						l.Errorf("%+v", err)
					}

					agent.channel.Send(&channel.ServerResponse{
						ID:      req.ID,
						Payload: new(agentpb.QANCollectResponse),
					})
				})

			case *agentpb.ActionResultRequest:
				// TODO: PMM-3978: In the future we need to merge action parts before send it to storage.
				err := models.ChangeActionResult(r.db.Querier, p.ActionId, agent.id, p.Error, string(p.Output), p.Done)
				if err != nil {
					l.Warnf("Failed to change action: %+v", err)
				}

				if !p.Done && p.Error != "" {
					l.Warnf("Action was done with an error: %v.", p.Error)
				}

				agent.channel.Send(&channel.ServerResponse{
					ID:      req.ID,
					Payload: new(agentpb.ActionResultResponse),
				})

			case *agentpb.JobResult:
				r.handleJobResult(l, p)
			case *agentpb.JobProgress:
				// TODO Handle job progress messages https://jira.percona.com/browse/PMM-7756

			case nil:
				l.Errorf("Unexpected request: %+v.", req)
			}
		}
	}
}

func (r *Registry) handleJobResult(l *logrus.Entry, result *agentpb.JobResult) {
	if e := r.db.InTransaction(func(t *reform.TX) error {
		res, err := models.FindJobResultByID(t.Querier, result.JobId)
		if err != nil {
			return err
		}

		switch result := result.Result.(type) {
		case *agentpb.JobResult_Error_:
			if err := r.handleJobError(res); err != nil {
				l.Errorf("failed to handle job error: %s", err)
			}
			res.Error = result.Error.Message
		case *agentpb.JobResult_Echo_:
			if res.Type != models.Echo {
				return errors.Errorf("result type echo doesn't match job type %s", res.Type)
			}
			res.Result = &models.JobResultData{
				Echo: &models.EchoJobResult{
					Message: result.Echo.Message,
				},
			}
		case *agentpb.JobResult_MysqlBackup:
			if res.Type != models.MySQLBackupJob {
				return errors.Errorf("result type %s doesn't match job type %s", models.MySQLBackupJob, res.Type)
			}

			_, err := models.ChangeArtifact(t.Querier, res.Result.MySQLBackup.ArtifactID, models.ChangeArtifactParams{
				Status: models.SuccessBackupStatus.Pointer(),
			})
			if err != nil {
				return err
			}
		case *agentpb.JobResult_MongodbBackup:
			if res.Type != models.MongoDBBackupJob {
				return errors.Errorf("result type %s doesn't match job type %s", models.MongoDBBackupJob, res.Type)
			}

			_, err := models.ChangeArtifact(t.Querier, res.Result.MongoDBBackup.ArtifactID, models.ChangeArtifactParams{
				Status: models.SuccessBackupStatus.Pointer(),
			})
			if err != nil {
				return err
			}
		case *agentpb.JobResult_MysqlRestoreBackup:
			if res.Type != models.MySQLRestoreBackupJob {
				return errors.Errorf("result type %s doesn't match job type %s", models.MySQLRestoreBackupJob, res.Type)
			}

			_, err := models.ChangeRestoreHistoryItem(
				t.Querier,
				res.Result.MySQLRestoreBackup.RestoreID,
				models.ChangeRestoreHistoryItemParams{
					Status: models.SuccessRestoreStatus,
				})
			if err != nil {
				return err
			}

		case *agentpb.JobResult_MongodbRestoreBackup:
			if res.Type != models.MongoDBRestoreBackupJob {
				return errors.Errorf("result type %s doesn't match job type %s", models.MongoDBRestoreBackupJob, res.Type)
			}

			_, err := models.ChangeRestoreHistoryItem(
				t.Querier,
				res.Result.MongoDBRestoreBackup.RestoreID,
				models.ChangeRestoreHistoryItemParams{
					Status: models.SuccessRestoreStatus,
				})
			if err != nil {
				return err
			}
		default:
			return errors.Errorf("unexpected job result type: %T", result)
		}
		res.Done = true
		return t.Update(res)
	}); e != nil {
		l.Errorf("Failed to save job result: %+v", e)
	}
}

func (r *Registry) handleJobError(jobResult *models.JobResult) error {
	var err error
	switch jobResult.Type {
	case models.Echo:
		// nothing
	case models.MySQLBackupJob:
		_, err = models.ChangeArtifact(r.db.Querier, jobResult.Result.MySQLBackup.ArtifactID, models.ChangeArtifactParams{
			Status: models.ErrorBackupStatus.Pointer(),
		})
	case models.MongoDBBackupJob:
		_, err = models.ChangeArtifact(r.db.Querier, jobResult.Result.MongoDBBackup.ArtifactID, models.ChangeArtifactParams{
			Status: models.ErrorBackupStatus.Pointer(),
		})
	case models.MySQLRestoreBackupJob:
		_, err = models.ChangeRestoreHistoryItem(
			r.db.Querier,
			jobResult.Result.MySQLRestoreBackup.RestoreID,
			models.ChangeRestoreHistoryItemParams{
				Status: models.ErrorRestoreStatus,
			})
	case models.MongoDBRestoreBackupJob:
		_, err = models.ChangeRestoreHistoryItem(
			r.db.Querier,
			jobResult.Result.MongoDBRestoreBackup.RestoreID,
			models.ChangeRestoreHistoryItemParams{
				Status: models.ErrorRestoreStatus,
			})
	default:
		// Don't do anything without explicit handling
	}
	return err

}

func (r *Registry) register(stream agentpb.Agent_ConnectServer) (*pmmAgentInfo, error) {
	ctx := stream.Context()
	l := logger.Get(ctx)
	agentMD, err := agentpb.ReceiveAgentConnectMetadata(stream)
	if err != nil {
		return nil, err
	}
	var runsOnNodeID string
	err = r.db.InTransaction(func(tx *reform.TX) error {
		runsOnNodeID, err = authenticate(agentMD, tx.Querier)
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

	serverMD := agentpb.ServerConnectMetadata{
		AgentRunsOnNodeID: runsOnNodeID,
		ServerVersion:     version.Version,
	}
	l.Debugf("Sending metadata: %+v.", serverMD)
	if err = agentpb.SendServerConnectMetadata(stream, &serverMD); err != nil {
		return nil, err
	}

	r.rw.Lock()
	defer r.rw.Unlock()

	// do not use r.get() - r.rw is already locked
	if agent := r.agents[agentMD.ID]; agent != nil {
		// pmm-agent with the same ID can still be connected in two cases:
		//   1. Someone uses the same ID by mistake, glitch, or malicious intent.
		//   2. pmm-agent detects broken connection and reconnects,
		//      but pmm-managed still thinks that the previous connection is okay.
		// In both cases, kick it.
		l.Warnf("Another pmm-agent with ID %q is already connected.", agentMD.ID)
		r.Kick(ctx, agentMD.ID)
	}

	agent := &pmmAgentInfo{
		channel:         channel.New(stream),
		id:              agentMD.ID,
		stateChangeChan: make(chan struct{}, 1),
		kick:            make(chan struct{}),
	}
	r.agents[agentMD.ID] = agent
	return agent, nil
}

func authenticate(md *agentpb.AgentConnectMetadata, q *reform.Querier) (string, error) {
	if md.ID == "" {
		return "", status.Error(codes.PermissionDenied, "Empty Agent ID.")
	}

	agent, err := models.FindAgentByID(q, md.ID)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", status.Errorf(codes.PermissionDenied, "No Agent with ID %q.", md.ID)
		}
		return "", errors.Wrap(err, "failed to find agent")
	}

	if agent.AgentType != models.PMMAgentType {
		return "", status.Errorf(codes.PermissionDenied, "No pmm-agent with ID %q.", md.ID)
	}

	if pointer.GetString(agent.RunsOnNodeID) == "" {
		return "", status.Errorf(codes.PermissionDenied, "Can't get 'runs_on_node_id' for pmm-agent with ID %q.", md.ID)
	}

	agentVersion, err := version.Parse(md.Version)
	if err != nil {
		return "", status.Errorf(codes.InvalidArgument, "Can't parse 'version' for pmm-agent with ID %q.", md.ID)
	}

	if err := addOrRemoveVMAgent(q, md.ID, pointer.GetString(agent.RunsOnNodeID), agentVersion); err != nil {
		return "", err
	}

	agent.Version = &md.Version
	if err := q.Update(agent); err != nil {
		return "", errors.Wrap(err, "failed to update agent")
	}

	return pointer.GetString(agent.RunsOnNodeID), nil
}

// unregister removes pmm-agent with given ID from the registry.
func (r *Registry) unregister(pmmAgentID string) *pmmAgentInfo {
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

// addOrRemoveVMAgent - creates vmAgent agentType if pmm-agent's version supports it and agent not exists yet,
// otherwise ensures that vmAgent not exist for pmm-agent and pmm-agent's agents don't have push_metrics mode,
// removes it if needed.
func addOrRemoveVMAgent(q *reform.Querier, pmmAgentID, runsOnNodeID string, pmmAgentVersion *version.Parsed) error {
	if pmmAgentVersion.Less(models.PMMAgentWithPushMetricsSupport) {
		// ensure that vmagent not exists and agents dont have push_metrics.
		return removeVMAgentFromPMMAgent(q, pmmAgentID)
	}
	return addVMAgentToPMMAgent(q, pmmAgentID, runsOnNodeID)
}

func addVMAgentToPMMAgent(q *reform.Querier, pmmAgentID, runsOnNodeID string) error {
	// TODO remove it after fix
	// https://jira.percona.com/browse/PMM-4420
	if runsOnNodeID == "pmm-server" {
		return nil
	}
	vmAgentType := models.VMAgentType
	vmAgent, err := models.FindAgents(q, models.AgentFilters{PMMAgentID: pmmAgentID, AgentType: &vmAgentType})
	if err != nil {
		return status.Errorf(codes.Internal, "Can't get 'vmAgent' for pmm-agent with ID %q", pmmAgentID)
	}
	if len(vmAgent) == 0 {
		if _, err := models.CreateAgent(q, models.VMAgentType, &models.CreateAgentParams{
			PMMAgentID:  pmmAgentID,
			PushMetrics: true,
			NodeID:      runsOnNodeID,
		}); err != nil {
			return errors.Wrapf(err, "Can't create 'vmAgent' for pmm-agent with ID %q", pmmAgentID)
		}
	}
	return nil
}

func removeVMAgentFromPMMAgent(q *reform.Querier, pmmAgentID string) error {
	vmAgentType := models.VMAgentType
	vmAgent, err := models.FindAgents(q, models.AgentFilters{PMMAgentID: pmmAgentID, AgentType: &vmAgentType})
	if err != nil {
		return status.Errorf(codes.Internal, "Can't get 'vmAgent' for pmm-agent with ID %q", pmmAgentID)
	}
	if len(vmAgent) != 0 {
		for _, agent := range vmAgent {
			if _, err := models.RemoveAgent(q, agent.AgentID, models.RemoveRestrict); err != nil {
				return errors.Wrapf(err, "Can't remove 'vmAgent' for pmm-agent with ID %q", pmmAgentID)
			}
		}
	}
	agents, err := models.FindAgents(q, models.AgentFilters{PMMAgentID: pmmAgentID})
	if err != nil {
		return errors.Wrapf(err, "Can't find agents for pmm-agent with ID %q", pmmAgentID)
	}
	for _, agent := range agents {
		if agent.PushMetrics {
			logrus.Warnf("disabling push_metrics for agent with unsupported version ID %q with pmm-agent ID %q", agent.AgentID, pmmAgentID)
			agent.PushMetrics = false
			if err := q.Update(agent); err != nil {
				return errors.Wrapf(err, "Can't set push_metrics=false for agent %q at pmm-agent with ID %q", agent.AgentID, pmmAgentID)
			}
		}
	}
	return nil
}

// Kick unregisters and forcefully disconnects pmm-agent with given ID.
func (r *Registry) Kick(ctx context.Context, pmmAgentID string) {
	agent := r.unregister(pmmAgentID)
	if agent == nil {
		return
	}

	l := logger.Get(ctx)
	l.Debugf("pmm-agent with ID %q will be kicked in a moment.", pmmAgentID)

	// see Run method
	close(agent.kick)

	// Do not close agent.stateChangeChan to avoid breaking RequestStateUpdate;
	// closing agent.kick is enough to exit runStateChangeHandler goroutine.
}

// ping sends Ping message to given Agent, waits for Pong and observes round-trip time and clock drift.
func (r *Registry) ping(ctx context.Context, agent *pmmAgentInfo) error {
	l := logger.Get(ctx)
	start := time.Now()
	resp, err := agent.channel.SendAndWaitResponse(new(agentpb.Ping))
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	roundtrip := time.Since(start)
	agentTime, err := ptypes.Timestamp(resp.(*agentpb.Pong).CurrentTime)
	if err != nil {
		return errors.Wrap(err, "failed to decode Pong.current_time")
	}
	clockDrift := agentTime.Sub(start) - roundtrip/2
	if clockDrift < 0 {
		clockDrift = -clockDrift
	}
	l.Infof("Round-trip time: %s. Estimated clock drift: %s.", roundtrip, clockDrift)
	r.mRoundTrip.Observe(roundtrip.Seconds())
	r.mClockDrift.Observe(clockDrift.Seconds())
	return nil
}

func updateAgentStatus(ctx context.Context, q *reform.Querier, agentID string, status inventorypb.AgentStatus, listenPort uint32) error {
	l := logger.Get(ctx)
	l.Debugf("updateAgentStatus: %s %s %d", agentID, status, listenPort)

	agent := &models.Agent{AgentID: agentID}
	err := q.Reload(agent)

	// FIXME that requires more investigation: https://jira.percona.com/browse/PMM-4932
	if err == reform.ErrNoRows {
		l.Warnf("Failed to select Agent by ID for (%s, %s).", agentID, status)

		switch status {
		case inventorypb.AgentStatus_STOPPING, inventorypb.AgentStatus_DONE:
			return nil
		}
	}
	if err != nil {
		return errors.Wrap(err, "failed to select Agent by ID")
	}

	agent.Status = status.String()
	agent.ListenPort = pointer.ToUint16(uint16(listenPort))
	if err = q.Update(agent); err != nil {
		return errors.Wrap(err, "failed to update Agent")
	}
	return nil
}

func (r *Registry) stateChanged(ctx context.Context, req *agentpb.StateChangedRequest) error {
	e := r.db.InTransaction(func(tx *reform.TX) error {
		agentIDs := r.roster.get(req.AgentId)
		if agentIDs == nil {
			agentIDs = []string{req.AgentId}
		}

		for _, agentID := range agentIDs {
			if err := updateAgentStatus(ctx, tx.Querier, agentID, req.Status, req.ListenPort); err != nil {
				return err
			}
		}
		return nil
	})
	if e != nil {
		return e
	}
	r.vmdb.RequestConfigurationUpdate()
	agent, err := models.FindAgentByID(r.db.Querier, req.AgentId)
	if err != nil {
		return err
	}
	if agent.PMMAgentID == nil {
		return nil
	}
	r.RequestStateUpdate(ctx, *agent.PMMAgentID)
	return nil
}

// UpdateAgentsState sends SetStateRequest to all pmm-agents with push metrics agents.
func (r *Registry) UpdateAgentsState(ctx context.Context) error {
	pmmAgents, err := models.FindPMMAgentsIDsWithPushMetrics(r.db.Querier)
	if err != nil {
		return errors.Wrap(err, "cannot find pmmAgentsIDs for AgentsState update")
	}
	var wg sync.WaitGroup
	limiter := make(chan struct{}, 10)
	for _, pmmAgentID := range pmmAgents {
		wg.Add(1)
		limiter <- struct{}{}
		go func(pmmAgentID string) {
			defer wg.Done()
			r.RequestStateUpdate(ctx, pmmAgentID)
			<-limiter
		}(pmmAgentID)
	}
	wg.Wait()
	return nil
}

// runStateChangeHandler runs pmm-agent state update loop for given pmm-agent until ctx is canceled or agent is kicked.
func (r *Registry) runStateChangeHandler(ctx context.Context, agent *pmmAgentInfo) {
	l := logger.Get(ctx).WithField("agent_id", agent.id)

	l.Info("Starting runStateChangeHandler ...")
	defer l.Info("Done runStateChangeHandler.")

	// stateChangeChan, state update loop, and RequestStateUpdate method ensure that state
	// is reloaded when requested, but several requests are batched together to avoid too often reloads.
	// That allows the caller to just call RequestStateUpdate when it seems fit.
	if cap(agent.stateChangeChan) != 1 {
		panic("stateChangeChan should have capacity 1")
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-agent.kick:
			return

		case <-agent.stateChangeChan:
			// batch several update requests together by delaying the first one
			sleepCtx, sleepCancel := context.WithTimeout(ctx, updateBatchDelay)
			<-sleepCtx.Done()
			sleepCancel()

			if ctx.Err() != nil {
				return
			}

			nCtx, cancel := context.WithTimeout(ctx, stateChangeTimeout)
			err := r.sendSetStateRequest(nCtx, agent)
			if err != nil {
				l.Error(err)
				r.RequestStateUpdate(ctx, agent.id)
			}
			cancel()
		}
	}
}

// SetAllAgentsStatusUnknown goes through all pmm-agents and sets status to UNKNOWN.
func (r *Registry) SetAllAgentsStatusUnknown(ctx context.Context) error {
	agentType := models.PMMAgentType
	agents, err := models.FindAgents(r.db.Querier, models.AgentFilters{AgentType: &agentType})
	if err != nil {
		return errors.Wrap(err, "failed to get pmm-agents")

	}
	for _, agent := range agents {
		if !r.IsConnected(agent.AgentID) {
			err = r.updateAgentStatusForChildren(ctx, agent.AgentID, inventorypb.AgentStatus_UNKNOWN, 0)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Registry) updateAgentStatusForChildren(ctx context.Context, agentID string, status inventorypb.AgentStatus, listenPort uint32) error {
	return r.db.InTransaction(func(t *reform.TX) error {
		agents, err := models.FindAgents(t.Querier, models.AgentFilters{
			PMMAgentID: agentID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to get pmm-agent's child agents")
		}
		for _, agent := range agents {
			if err := updateAgentStatus(ctx, t.Querier, agent.AgentID, status, listenPort); err != nil {
				return errors.Wrap(err, "failed to update agent's status")
			}
		}
		return nil
	})
}

// RequestStateUpdate requests state update on pmm-agent with given ID. It sets
// the status to done if the agent is not connected.
func (r *Registry) RequestStateUpdate(ctx context.Context, pmmAgentID string) {
	l := logger.Get(ctx)

	agent, err := r.get(pmmAgentID)
	if err != nil {
		l.Infof("RequestStateUpdate: %s.", err)
		return
	}

	select {
	case agent.stateChangeChan <- struct{}{}:
	default:
	}
}

// sendSetStateRequest sends SetStateRequest to given pmm-agent.
func (r *Registry) sendSetStateRequest(ctx context.Context, agent *pmmAgentInfo) error {
	l := logger.Get(ctx)
	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > time.Second {
			l.Warnf("sendSetStateRequest took %s.", dur)
		}
	}()
	pmmAgent, err := models.FindAgentByID(r.db.Querier, agent.id)
	if err != nil {
		return errors.Wrap(err, "failed to get PMM Agent")
	}
	pmmAgentVersion, err := version.Parse(*pmmAgent.Version)
	if err != nil {
		return errors.Wrapf(err, "failed to parse PMM agent version %q", *pmmAgent.Version)
	}

	agents, err := models.FindAgents(r.db.Querier, models.AgentFilters{PMMAgentID: agent.id})
	if err != nil {
		return errors.Wrap(err, "failed to collect agents")
	}

	redactMode := redactSecrets
	if l.Logger.GetLevel() >= logrus.DebugLevel {
		redactMode = exposeSecrets
	}

	rdsExporters := make(map[*models.Node]*models.Agent)
	agentProcesses := make(map[string]*agentpb.SetStateRequest_AgentProcess)
	builtinAgents := make(map[string]*agentpb.SetStateRequest_BuiltinAgent)
	for _, row := range agents {
		if row.Disabled {
			continue
		}

		// in order of AgentType consts
		switch row.AgentType {
		case models.PMMAgentType:
			continue
		case models.VMAgentType:
			scrapeCfg, err := r.vmdb.BuildScrapeConfigForVMAgent(agent.id)
			if err != nil {
				return errors.Wrapf(err, "cannot get agent scrape config for agent: %s", agent.id)
			}
			agentProcesses[row.AgentID] = vmAgentConfig(string(scrapeCfg))

		case models.NodeExporterType:
			node, err := models.FindNodeByID(r.db.Querier, pointer.GetString(row.NodeID))
			if err != nil {
				return err
			}
			agentProcesses[row.AgentID] = nodeExporterConfig(node, row)

		case models.RDSExporterType:
			node, err := models.FindNodeByID(r.db.Querier, pointer.GetString(row.NodeID))
			if err != nil {
				return err
			}
			rdsExporters[node] = row
		case models.ExternalExporterType:
			// ignore

		case models.AzureDatabaseExporterType:
			service, err := models.FindServiceByID(r.db.Querier, pointer.GetString(row.ServiceID))
			if err != nil {
				return err
			}
			config, err := azureDatabaseExporterConfig(row, service, redactMode)
			if err != nil {
				return err
			}
			agentProcesses[row.AgentID] = config

		// Agents with exactly one Service
		case models.MySQLdExporterType, models.MongoDBExporterType, models.PostgresExporterType, models.ProxySQLExporterType,
			models.QANMySQLPerfSchemaAgentType, models.QANMySQLSlowlogAgentType, models.QANMongoDBProfilerAgentType, models.QANPostgreSQLPgStatementsAgentType,
			models.QANPostgreSQLPgStatMonitorAgentType:

			service, err := models.FindServiceByID(r.db.Querier, pointer.GetString(row.ServiceID))
			if err != nil {
				return err
			}

			switch row.AgentType {
			case models.MySQLdExporterType:
				agentProcesses[row.AgentID] = mysqldExporterConfig(service, row, redactMode)
			case models.MongoDBExporterType:
				agentProcesses[row.AgentID] = mongodbExporterConfig(service, row, redactMode, pmmAgentVersion)
			case models.PostgresExporterType:
				agentProcesses[row.AgentID] = postgresExporterConfig(service, row, redactMode, pmmAgentVersion)
			case models.ProxySQLExporterType:
				agentProcesses[row.AgentID] = proxysqlExporterConfig(service, row, redactMode)
			case models.QANMySQLPerfSchemaAgentType:
				builtinAgents[row.AgentID] = qanMySQLPerfSchemaAgentConfig(service, row)
			case models.QANMySQLSlowlogAgentType:
				builtinAgents[row.AgentID] = qanMySQLSlowlogAgentConfig(service, row)
			case models.QANMongoDBProfilerAgentType:
				builtinAgents[row.AgentID] = qanMongoDBProfilerAgentConfig(service, row)
			case models.QANPostgreSQLPgStatementsAgentType:
				builtinAgents[row.AgentID] = qanPostgreSQLPgStatementsAgentConfig(service, row)
			case models.QANPostgreSQLPgStatMonitorAgentType:
				builtinAgents[row.AgentID] = qanPostgreSQLPgStatMonitorAgentConfig(service, row)
			}

		default:
			return errors.Errorf("unhandled Agent type %s", row.AgentType)
		}
	}

	if len(rdsExporters) > 0 {
		rdsExporterIDs := make([]string, 0, len(rdsExporters))
		for _, rdsExporter := range rdsExporters {
			rdsExporterIDs = append(rdsExporterIDs, rdsExporter.AgentID)
		}
		sort.Strings(rdsExporterIDs)

		groupID := r.roster.add(agent.id, rdsGroup, rdsExporterIDs)
		c, err := rdsExporterConfig(rdsExporters, redactMode)
		if err != nil {
			return err
		}
		agentProcesses[groupID] = c
	}
	state := &agentpb.SetStateRequest{
		AgentProcesses: agentProcesses,
		BuiltinAgents:  builtinAgents,
	}
	l.Debugf("sendSetStateRequest:\n%s", proto.MarshalTextString(state))
	resp, err := agent.channel.SendAndWaitResponse(state)
	if err != nil {
		return err
	}
	l.Infof("SetState response: %+v.", resp)
	return nil
}

func (r *Registry) isExternalExporterConnectionCheckSupported(q *reform.Querier, pmmAgentID string) (bool, error) {
	pmmAgent, err := models.FindAgentByID(r.db.Querier, pmmAgentID)
	if err != nil {
		return false, fmt.Errorf("failed to get PMM Agent: %s.", err)
	}
	pmmAgentVersion, err := version.Parse(*pmmAgent.Version)
	if err != nil {
		return false, fmt.Errorf("failed to parse PMM agent version %q: %s", *pmmAgent.Version, err)
	}

	if pmmAgentVersion.Less(checkExternalExporterConnectionPMMVersion) {
		return false, nil
	}
	return true, nil
}

// CheckConnectionToService sends request to pmm-agent to check connection to service.
func (r *Registry) CheckConnectionToService(ctx context.Context, q *reform.Querier, service *models.Service, agent *models.Agent) error {
	// TODO: extract to a separate struct to keep Single Responsibility principles: https://jira.percona.com/browse/PMM-4932
	l := logger.Get(ctx)
	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > 4*time.Second {
			l.Warnf("CheckConnectionToService took %s.", dur)
		}
	}()

	pmmAgentID := pointer.GetString(agent.PMMAgentID)
	if !agent.PushMetrics && (service.ServiceType == models.ExternalServiceType || service.ServiceType == models.HAProxyServiceType) {
		pmmAgentID = models.PMMServerAgentID
	}

	// Skip check connection to external exporter with old pmm-agent.
	if service.ServiceType == models.ExternalServiceType || service.ServiceType == models.HAProxyServiceType {
		isCheckConnSupported, err := r.isExternalExporterConnectionCheckSupported(q, pmmAgentID)
		if err != nil {
			return err
		}

		if !isCheckConnSupported {
			return nil
		}
	}

	pmmAgent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}

	var request *agentpb.CheckConnectionRequest
	switch service.ServiceType {
	case models.MySQLServiceType:
		tdp := agent.TemplateDelimiters(service)
		request = &agentpb.CheckConnectionRequest{
			Type:    inventorypb.ServiceType_MYSQL_SERVICE,
			Dsn:     agent.DSN(service, 2*time.Second, "", nil),
			Timeout: ptypes.DurationProto(3 * time.Second),
			TextFiles: &agentpb.TextFiles{
				Files:              agent.Files(),
				TemplateLeftDelim:  tdp.Left,
				TemplateRightDelim: tdp.Right,
			},
			TlsSkipVerify: agent.TLSSkipVerify,
		}
	case models.PostgreSQLServiceType:
		request = &agentpb.CheckConnectionRequest{
			Type:    inventorypb.ServiceType_POSTGRESQL_SERVICE,
			Dsn:     agent.DSN(service, 2*time.Second, "postgres", nil),
			Timeout: ptypes.DurationProto(3 * time.Second),
		}
	case models.MongoDBServiceType:
		tdp := agent.TemplateDelimiters(service)
		request = &agentpb.CheckConnectionRequest{
			Type:    inventorypb.ServiceType_MONGODB_SERVICE,
			Dsn:     agent.DSN(service, 2*time.Second, "", nil),
			Timeout: ptypes.DurationProto(3 * time.Second),
			TextFiles: &agentpb.TextFiles{
				Files:              agent.Files(),
				TemplateLeftDelim:  tdp.Left,
				TemplateRightDelim: tdp.Right,
			},
		}
	case models.ProxySQLServiceType:
		request = &agentpb.CheckConnectionRequest{
			Type:    inventorypb.ServiceType_PROXYSQL_SERVICE,
			Dsn:     agent.DSN(service, 2*time.Second, "", nil),
			Timeout: ptypes.DurationProto(3 * time.Second),
		}
	case models.ExternalServiceType:
		exporterURL, err := agent.ExporterURL(q)
		if err != nil {
			return err
		}

		request = &agentpb.CheckConnectionRequest{
			Type:    inventorypb.ServiceType_EXTERNAL_SERVICE,
			Dsn:     exporterURL,
			Timeout: ptypes.DurationProto(3 * time.Second),
		}
	case models.HAProxyServiceType:
		exporterURL, err := agent.ExporterURL(q)
		if err != nil {
			return err
		}

		request = &agentpb.CheckConnectionRequest{
			Type:    inventorypb.ServiceType_HAPROXY_SERVICE,
			Dsn:     exporterURL,
			Timeout: ptypes.DurationProto(3 * time.Second),
		}
	default:
		return errors.Errorf("unhandled Service type %s", service.ServiceType)
	}

	var sanitizedDSN string
	for _, word := range redactWords(agent) {
		sanitizedDSN = strings.ReplaceAll(request.Dsn, word, "****")
	}
	l.Infof("CheckConnectionRequest: type: %s, DSN: %s timeout: %s.", request.Type, sanitizedDSN, request.Timeout)
	resp, err := pmmAgent.channel.SendAndWaitResponse(request)
	if err != nil {
		return err
	}
	l.Infof("CheckConnection response: %+v.", resp)

	switch service.ServiceType {
	case models.MySQLServiceType:
		tableCount := resp.(*agentpb.CheckConnectionResponse).GetStats().GetTableCount()
		agent.TableCount = &tableCount
		l.Debugf("Updating table count: %d.", tableCount)
		if err = q.Update(agent); err != nil {
			return errors.Wrap(err, "failed to update table count")
		}
	case models.ExternalServiceType, models.HAProxyServiceType:
	case models.PostgreSQLServiceType:
	case models.MongoDBServiceType:
	case models.ProxySQLServiceType:
		// nothing yet

	default:
		return errors.Errorf("unhandled Service type %s", service.ServiceType)
	}

	msg := resp.(*agentpb.CheckConnectionResponse).Error
	switch msg {
	case "":
		return nil
	case context.Canceled.Error(), context.DeadlineExceeded.Error():
		msg = fmt.Sprintf("timeout (%s)", msg)
	}
	return status.Error(codes.FailedPrecondition, fmt.Sprintf("Connection check failed: %s.", msg))
}

func (r *Registry) get(pmmAgentID string) (*pmmAgentInfo, error) {
	r.rw.RLock()
	pmmAgent := r.agents[pmmAgentID]
	r.rw.RUnlock()
	if pmmAgent == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "pmm-agent with ID %q is not currently connected", pmmAgentID)
	}
	return pmmAgent, nil
}

// Describe implements prometheus.Collector.
func (r *Registry) Describe(ch chan<- *prom.Desc) {
	ch <- mSentDesc
	ch <- mRecvDesc
	ch <- mResponsesDesc
	ch <- mRequestsDesc

	r.mAgents.Describe(ch)
	r.mConnects.Describe(ch)
	r.mDisconnects.Describe(ch)
	r.mRoundTrip.Describe(ch)
	r.mClockDrift.Describe(ch)
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

// StartMySQLExplainAction starts MySQL EXPLAIN Action on pmm-agent.
// TODO: Extract it from here: https://jira.percona.com/browse/PMM-4932
func (r *Registry) StartMySQLExplainAction(ctx context.Context, id, pmmAgentID, dsn, query string, format agentpb.MysqlExplainOutputFormat, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error {
	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}

	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MysqlExplainParams{
			MysqlExplainParams: &agentpb.StartActionRequest_MySQLExplainParams{
				Dsn:          dsn,
				Query:        query,
				OutputFormat: format,
				TlsFiles: &agentpb.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
				TlsSkipVerify: tlsSkipVerify,
			},
		},
		Timeout: defaultActionTimeout,
	}

	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMySQLShowCreateTableAction starts mysql-show-create-table action on pmm-agent.
// TODO: Extract it from here: https://jira.percona.com/browse/PMM-4932
func (r *Registry) StartMySQLShowCreateTableAction(ctx context.Context, id, pmmAgentID, dsn, table string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MysqlShowCreateTableParams{
			MysqlShowCreateTableParams: &agentpb.StartActionRequest_MySQLShowCreateTableParams{
				Dsn:   dsn,
				Table: table,
				TlsFiles: &agentpb.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
				TlsSkipVerify: tlsSkipVerify,
			},
		},
		Timeout: defaultActionTimeout,
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMySQLShowTableStatusAction starts mysql-show-table-status action on pmm-agent.
// TODO: Extract it from here: https://jira.percona.com/browse/PMM-4932
func (r *Registry) StartMySQLShowTableStatusAction(ctx context.Context, id, pmmAgentID, dsn, table string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MysqlShowTableStatusParams{
			MysqlShowTableStatusParams: &agentpb.StartActionRequest_MySQLShowTableStatusParams{
				Dsn:   dsn,
				Table: table,
				TlsFiles: &agentpb.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
				TlsSkipVerify: tlsSkipVerify,
			},
		},
		Timeout: defaultActionTimeout,
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMySQLShowIndexAction starts mysql-show-index action on pmm-agent.
// TODO: Extract it from here: https://jira.percona.com/browse/PMM-4932
func (r *Registry) StartMySQLShowIndexAction(ctx context.Context, id, pmmAgentID, dsn, table string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MysqlShowIndexParams{
			MysqlShowIndexParams: &agentpb.StartActionRequest_MySQLShowIndexParams{
				Dsn:   dsn,
				Table: table,
				TlsFiles: &agentpb.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
				TlsSkipVerify: tlsSkipVerify,
			},
		},
		Timeout: defaultActionTimeout,
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartPostgreSQLShowCreateTableAction starts postgresql-show-create-table action on pmm-agent.
// TODO: Extract it from here: https://jira.percona.com/browse/PMM-4932
func (r *Registry) StartPostgreSQLShowCreateTableAction(ctx context.Context, id, pmmAgentID, dsn, table string) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_PostgresqlShowCreateTableParams{
			PostgresqlShowCreateTableParams: &agentpb.StartActionRequest_PostgreSQLShowCreateTableParams{
				Dsn:   dsn,
				Table: table,
			},
		},
		Timeout: defaultActionTimeout,
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartPostgreSQLShowIndexAction starts postgresql-show-index action on pmm-agent.
// TODO: Extract it from here: https://jira.percona.com/browse/PMM-4932
func (r *Registry) StartPostgreSQLShowIndexAction(ctx context.Context, id, pmmAgentID, dsn, table string) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_PostgresqlShowIndexParams{
			PostgresqlShowIndexParams: &agentpb.StartActionRequest_PostgreSQLShowIndexParams{
				Dsn:   dsn,
				Table: table,
			},
		},
		Timeout: defaultActionTimeout,
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMongoDBExplainAction starts MongoDB query explain action on pmm-agent.
func (r *Registry) StartMongoDBExplainAction(ctx context.Context, id, pmmAgentID, dsn, query string, files map[string]string, tdp *models.DelimiterPair) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MongodbExplainParams{
			MongodbExplainParams: &agentpb.StartActionRequest_MongoDBExplainParams{
				Dsn:   dsn,
				Query: query,
				TextFiles: &agentpb.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
			},
		},
		Timeout: defaultActionTimeout,
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMySQLQueryShowAction starts MySQL SHOW query action on pmm-agent.
func (r *Registry) StartMySQLQueryShowAction(ctx context.Context, id, pmmAgentID, dsn, query string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MysqlQueryShowParams{
			MysqlQueryShowParams: &agentpb.StartActionRequest_MySQLQueryShowParams{
				Dsn:   dsn,
				Query: query,
				TlsFiles: &agentpb.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
				TlsSkipVerify: tlsSkipVerify,
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMySQLQuerySelectAction starts MySQL SELECT query action on pmm-agent.
func (r *Registry) StartMySQLQuerySelectAction(ctx context.Context, id, pmmAgentID, dsn, query string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MysqlQuerySelectParams{
			MysqlQuerySelectParams: &agentpb.StartActionRequest_MySQLQuerySelectParams{
				Dsn:   dsn,
				Query: query,
				TlsFiles: &agentpb.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
				TlsSkipVerify: tlsSkipVerify,
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartPostgreSQLQueryShowAction starts PostgreSQL SHOW query action on pmm-agent.
func (r *Registry) StartPostgreSQLQueryShowAction(ctx context.Context, id, pmmAgentID, dsn string) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_PostgresqlQueryShowParams{
			PostgresqlQueryShowParams: &agentpb.StartActionRequest_PostgreSQLQueryShowParams{
				Dsn: dsn,
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartPostgreSQLQuerySelectAction starts PostgreSQL SELECT query action on pmm-agent.
func (r *Registry) StartPostgreSQLQuerySelectAction(ctx context.Context, id, pmmAgentID, dsn, query string) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_PostgresqlQuerySelectParams{
			PostgresqlQuerySelectParams: &agentpb.StartActionRequest_PostgreSQLQuerySelectParams{
				Dsn:   dsn,
				Query: query,
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMongoDBQueryGetParameterAction starts MongoDB getParameter query action on pmm-agent.
func (r *Registry) StartMongoDBQueryGetParameterAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MongodbQueryGetparameterParams{
			MongodbQueryGetparameterParams: &agentpb.StartActionRequest_MongoDBQueryGetParameterParams{
				Dsn: dsn,
				TextFiles: &agentpb.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMongoDBQueryBuildInfoAction starts MongoDB buildInfo query action on pmm-agent.
func (r *Registry) StartMongoDBQueryBuildInfoAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MongodbQueryBuildinfoParams{
			MongodbQueryBuildinfoParams: &agentpb.StartActionRequest_MongoDBQueryBuildInfoParams{
				Dsn: dsn,
				TextFiles: &agentpb.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMongoDBQueryGetCmdLineOptsAction starts MongoDB getCmdLineOpts query action on pmm-agent.
func (r *Registry) StartMongoDBQueryGetCmdLineOptsAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MongodbQueryGetcmdlineoptsParams{
			MongodbQueryGetcmdlineoptsParams: &agentpb.StartActionRequest_MongoDBQueryGetCmdLineOptsParams{
				Dsn: dsn,
				TextFiles: &agentpb.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartPTSummaryAction starts pt-summary action on pmm-agent.
func (r *Registry) StartPTSummaryAction(ctx context.Context, id, pmmAgentID string) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		// Requires params to be passed, even empty, othervise request's marshal fail.
		Params: &agentpb.StartActionRequest_PtSummaryParams{
			PtSummaryParams: &agentpb.StartActionRequest_PTSummaryParams{},
		},
		Timeout: defaultPtActionTimeout,
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartPTPgSummaryAction starts pt-pg-summary action on the pmm-agent.
// The function returns nil if ok, otherwise an error code
func (r *Registry) StartPTPgSummaryAction(ctx context.Context, id, pmmAgentID, address string, port uint16, username, password string) error {
	actionRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_PtPgSummaryParams{
			PtPgSummaryParams: &agentpb.StartActionRequest_PTPgSummaryParams{
				Host:     address,
				Port:     uint32(port),
				Username: username,
				Password: password,
			},
		},
		Timeout: defaultPtActionTimeout,
	}

	pmmAgent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = pmmAgent.channel.SendAndWaitResponse(actionRequest)
	return err
}

// StartPTMongoDBSummaryAction starts pt-mongodb-summary action on the pmm-agent.
// The function returns nil if ok, otherwise an error code
func (r *Registry) StartPTMongoDBSummaryAction(ctx context.Context, id, pmmAgentID, address string, port uint16, username, password string) error {
	// Action request data that'll be sent to agent
	actionRequest := &agentpb.StartActionRequest{
		ActionId: id,
		// Proper params that'll will be passed to the command on the agent's side, even empty, othervise request's marshal fail.
		Params: &agentpb.StartActionRequest_PtMongodbSummaryParams{
			PtMongodbSummaryParams: &agentpb.StartActionRequest_PTMongoDBSummaryParams{
				Host:     address,
				Port:     uint32(port),
				Username: username,
				Password: password,
			},
		},
		Timeout: defaultPtActionTimeout,
	}

	// Agent which the action request will be sent to, got by the provided ID
	pmmAgent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = pmmAgent.channel.SendAndWaitResponse(actionRequest)
	return err
}

// StartPTMySQLSummaryAction starts pt-mysql-summary action on the pmm-agent.
// The pt-mysql-summary's execution may require some of the following params: host, port, socket, username, password.
func (r *Registry) StartPTMySQLSummaryAction(ctx context.Context, id, pmmAgentID, address string, port uint16, socket, username, password string) error {
	actionRequest := &agentpb.StartActionRequest{
		ActionId: id,
		// Proper params that'll will be passed to the command on the agent's side.
		Params: &agentpb.StartActionRequest_PtMysqlSummaryParams{
			PtMysqlSummaryParams: &agentpb.StartActionRequest_PTMySQLSummaryParams{
				Host:     address,
				Port:     uint32(port),
				Socket:   socket,
				Username: username,
				Password: password,
			},
		},
		Timeout: defaultPtActionTimeout,
	}

	pmmAgent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = pmmAgent.channel.SendAndWaitResponse(actionRequest)
	return err
}

// StopAction stops action with given given id.
// TODO: Extract it from here: https://jira.percona.com/browse/PMM-4932
func (r *Registry) StopAction(ctx context.Context, actionID string) error {
	// TODO Seems that we have a bug here, we passing actionID to the method that expects pmmAgentID
	agent, err := r.get(actionID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(&agentpb.StopActionRequest{ActionId: actionID})
	return err
}

// check interfaces
var (
	_ prom.Collector = (*Registry)(nil)
)
