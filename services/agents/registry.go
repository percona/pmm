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
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/agents/channel"
	"github.com/percona/pmm-managed/utils/logger"
)

const (
	prometheusNamespace = "pmm_managed"
	prometheusSubsystem = "agents"
)

type agentInfo struct {
	channel *channel.Channel
	id      string
	kick    chan struct{}
}

// Registry keeps track of all connected pmm-agents.
//
// TODO Split into several types?
type Registry struct {
	db         *reform.DB
	prometheus prometheusService
	qanClient  qanClient

	rw     sync.RWMutex
	agents map[string]*agentInfo // id -> info

	sharedMetrics *channel.SharedChannelMetrics
	mConnects     prom.Counter
	mDisconnects  *prom.CounterVec
	mRoundTrip    prom.Summary
	mClockDrift   prom.Summary
}

// NewRegistry creates a new registry with given database connection.
func NewRegistry(db *reform.DB, prometheus prometheusService, qanClient qanClient) *Registry {
	r := &Registry{
		db:         db,
		prometheus: prometheus,
		qanClient:  qanClient,

		agents: make(map[string]*agentInfo),

		sharedMetrics: channel.NewSharedMetrics(),
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

		case req := <-agent.channel.Requests():
			if req == nil {
				disconnectReason = "done"
				return agent.channel.Wait()
			}

			switch p := req.Payload.(type) {
			case *agentpb.Ping:
				agent.channel.SendResponse(&channel.ServerResponse{
					ID: req.ID,
					Payload: &agentpb.Pong{
						CurrentTime: ptypes.TimestampNow(),
					},
				})

			case *agentpb.StateChangedRequest:
				if err := r.stateChanged(ctx, p); err != nil {
					l.Errorf("%+v", err)
				}

				agent.channel.SendResponse(&channel.ServerResponse{
					ID:      req.ID,
					Payload: new(agentpb.StateChangedResponse),
				})

			case *agentpb.QANCollectRequest:
				if err := r.qanClient.Collect(ctx, p.Message); err != nil {
					l.Errorf("%+v", err)
				}

				agent.channel.SendResponse(&channel.ServerResponse{
					ID:      req.ID,
					Payload: new(agentpb.QANCollectResponse),
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

				agent.channel.SendResponse(&channel.ServerResponse{
					ID:      req.ID,
					Payload: new(agentpb.ActionResultResponse),
				})

			case nil:
				l.Warnf("Unexpected request: %v.", req)
				disconnectReason = "unimplemented"
				return status.Error(codes.Unimplemented, "Unexpected request payload.")
			}
		}
	}
}

func (r *Registry) register(stream agentpb.Agent_ConnectServer) (*agentInfo, error) {
	ctx := stream.Context()
	l := logger.Get(ctx)
	agentMD, err := agentpb.ReceiveAgentConnectMetadata(stream)
	if err != nil {
		return nil, err
	}
	runsOnNodeID, err := authenticate(agentMD, r.db.Querier)
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
		close(agent.kick)
	}

	agent := &agentInfo{
		channel: channel.New(stream, r.sharedMetrics),
		id:      agentMD.ID,
		kick:    make(chan struct{}),
	}
	r.agents[agentMD.ID] = agent
	return agent, nil
}

func authenticate(md *agentpb.AgentConnectMetadata, q *reform.Querier) (string, error) { //nolint:unused
	if md.ID == "" {
		return "", status.Error(codes.Unauthenticated, "Empty Agent ID.")
	}

	agent, err := models.AgentFindByID(q, md.ID)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", status.Errorf(codes.Unauthenticated, "No Agent with ID %q.", md.ID)
		}
		return "", errors.Wrap(err, "failed to find agent")
	}

	if agent.AgentType != models.PMMAgentType {
		return "", status.Errorf(codes.Unauthenticated, "No pmm-agent with ID %q.", md.ID)
	}

	if pointer.GetString(agent.RunsOnNodeID) == "" {
		return "", status.Errorf(codes.Unauthenticated, "Can't get 'runs_on_node_id' for pmm-agent with ID %q.", md.ID)
	}

	agent.Version = &md.Version
	if err := q.Update(agent); err != nil {
		return "", errors.Wrap(err, "failed to update agent")
	}

	return pointer.GetString(agent.RunsOnNodeID), nil
}

// Kick disconnects pmm-agent with given ID.
func (r *Registry) Kick(ctx context.Context, pmmAgentID string) {
	// We do not check that pmmAgentID is in fact ID of existing pmm-agent because
	// it may be already deleted from the database, that's why we disconnect it.

	r.rw.Lock()
	defer r.rw.Unlock()

	// do not use r.get() - r.rw is already locked
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
	resp := agent.channel.SendRequest(new(agentpb.Ping))
	if resp == nil {
		return
	}
	roundtrip := time.Since(start)
	agentTime, err := ptypes.Timestamp(resp.(*agentpb.Pong).CurrentTime)
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

// SendSetStateRequest sends SetStateRequest to pmm-agent with given ID.
func (r *Registry) SendSetStateRequest(ctx context.Context, pmmAgentID string) {
	l := logger.Get(ctx)
	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > time.Second {
			l.Warnf("SendSetStateRequest took %s.", dur)
		}
	}()

	agent, err := r.get(pmmAgentID)
	if err != nil {
		l.Infof("SendSetStateRequest: %s.", err)
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
			nodes, err := models.FindNodesForAgentID(r.db.Querier, row.AgentID)
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

		case models.QANMySQLSlowlogAgentType:
			services, err := models.ServicesForAgent(r.db.Querier, row.AgentID)
			if err != nil {
				l.Error(err)
				return
			}
			if len(services) != 1 {
				l.Errorf("Expected exactly one Services, got %d.", len(services))
				return
			}
			builtinAgents[row.AgentID] = qanMySQLSlowlogAgentConfig(services[0], row)

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

		case models.PostgresExporterType:
			services, err := models.ServicesForAgent(r.db.Querier, row.AgentID)
			if err != nil {
				l.Error(err)
				return
			}
			if len(services) != 1 {
				l.Errorf("Expected exactly one Services, got %d.", len(services))
				return
			}
			agentProcesses[row.AgentID] = postgresExporterConfig(services[0], row)

		case models.QANMongoDBProfilerAgentType:
			services, err := models.ServicesForAgent(r.db.Querier, row.AgentID)
			if err != nil {
				l.Error(err)
				return
			}
			if len(services) != 1 {
				l.Errorf("Expected exactly one Services, got %d.", len(services))
				return
			}
			builtinAgents[row.AgentID] = qanMongoDBProfilerAgentConfig(services[0], row)

		case models.ProxySQLExporterType:
			services, err := models.ServicesForAgent(r.db.Querier, row.AgentID)
			if err != nil {
				l.Error(err)
				return
			}
			if len(services) != 1 {
				l.Errorf("Expected exactly one Service, got %d.", len(services))
				return
			}
			agentProcesses[row.AgentID] = proxysqlExporterConfig(services[0], row)

		case models.QANPostgreSQLPgStatementsAgentType:
			services, err := models.ServicesForAgent(r.db.Querier, row.AgentID)
			if err != nil {
				l.Error(err)
				return
			}
			if len(services) != 1 {
				l.Errorf("Expected exactly one Services, got %d.", len(services))
				return
			}
			builtinAgents[row.AgentID] = qanPostgreSQLPgStatementsAgentConfig(services[0], row)

		default:
			l.Panicf("unhandled Agent type %s", row.AgentType)
		}
	}

	state := &agentpb.SetStateRequest{
		AgentProcesses: agentProcesses,
		BuiltinAgents:  builtinAgents,
	}
	l.Infof("SendSetStateRequest: %+v.", state)
	resp := agent.channel.SendRequest(state)
	l.Infof("SetState response: %+v.", resp)
}

// CheckConnectionToService sends request to pmm-agent to check connection to service.
func (r *Registry) CheckConnectionToService(ctx context.Context, service *models.Service, agent *models.Agent) error {
	// TODO: extract to a separate struct to keep Single Responsibility principles.
	l := logger.Get(ctx)
	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > 2*time.Second {
			l.Warnf("CheckConnectionToService took %s.", dur)
		}
	}()

	pmmAgentID := pointer.GetString(agent.PMMAgentID)
	pmmAgent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}

	var request *agentpb.CheckConnectionRequest
	switch service.ServiceType {
	case models.MySQLServiceType:
		request = &agentpb.CheckConnectionRequest{
			Type: inventorypb.ServiceType_MYSQL_SERVICE,
			Dsn:  agent.DSN(service, time.Second, ""),
		}
	case models.PostgreSQLServiceType:
		request = &agentpb.CheckConnectionRequest{
			Type: inventorypb.ServiceType_POSTGRESQL_SERVICE,
			Dsn:  agent.DSN(service, time.Second, "postgres"),
		}
	case models.MongoDBServiceType:
		request = &agentpb.CheckConnectionRequest{
			Type: inventorypb.ServiceType_MONGODB_SERVICE,
			Dsn:  agent.DSN(service, time.Second, ""),
		}
	case models.ProxySQLServiceType:
		request = &agentpb.CheckConnectionRequest{
			Type: inventorypb.ServiceType_PROXYSQL_SERVICE,
			Dsn:  agent.DSN(service, time.Second, ""),
		}
	default:
		l.Panicf("unhandled Service type %s", service.ServiceType)
	}

	l.Infof("CheckConnectionRequest: %+v.", request)
	resp := pmmAgent.channel.SendRequest(request)
	l.Infof("CheckConnection response: %+v.", resp)
	checkConnectionResponse := resp.(*agentpb.CheckConnectionResponse)
	if checkConnectionResponse.Error != "" {
		return status.Error(codes.FailedPrecondition, checkConnectionResponse.Error)
	}
	return nil
}

func (r *Registry) get(pmmAgentID string) (*agentInfo, error) {
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

// StartPTSummaryAction starts pt-summary action on pmm-agent.
// TODO: Extract it from here. Where...?
func (r *Registry) StartPTSummaryAction(ctx context.Context, id, pmmAgentID string, args []string) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_PtSummaryParams{
			PtSummaryParams: &agentpb.StartActionRequest_ProcessParams{
				Args: args,
			},
		},
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}

	agent.channel.SendRequest(aRequest)
	return nil
}

// StartPTMySQLSummaryAction starts pt-mysql-summary action on pmm-agent.
// TODO: Extract it from here. Where...?
func (r *Registry) StartPTMySQLSummaryAction(ctx context.Context, id, pmmAgentID string, args []string) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_PtMysqlSummaryParams{
			PtMysqlSummaryParams: &agentpb.StartActionRequest_ProcessParams{
				Args: args,
			},
		},
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}

	agent.channel.SendRequest(aRequest)
	return nil
}

// StartMySQLExplainAction starts MySQL EXPLAIN Action on pmm-agent.
func (r *Registry) StartMySQLExplainAction(ctx context.Context, id, pmmAgentID, dsn, query string, format agentpb.MysqlExplainOutputFormat) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MysqlExplainParams{
			MysqlExplainParams: &agentpb.StartActionRequest_MySQLExplainParams{
				Dsn:          dsn,
				Query:        query,
				OutputFormat: format,
			},
		},
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}

	agent.channel.SendRequest(aRequest)
	return nil
}

// StartMySQLShowCreateTableAction starts mysql-show-create-table action on pmm-agent.
// TODO: Extract it from here. Where...?
func (r *Registry) StartMySQLShowCreateTableAction(ctx context.Context, id, pmmAgentID, dsn, table string) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MysqlShowCreateTableParams{
			MysqlShowCreateTableParams: &agentpb.StartActionRequest_MySQLShowCreateTableParams{
				Dsn:   dsn,
				Table: table,
			},
		},
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}

	agent.channel.SendRequest(aRequest)
	return nil
}

// StartMySQLShowTableStatusAction starts mysql-show-table-status action on pmm-agent.
// TODO: Extract it from here. Where...?
func (r *Registry) StartMySQLShowTableStatusAction(ctx context.Context, id, pmmAgentID, dsn, table string) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MysqlShowTableStatusParams{
			MysqlShowTableStatusParams: &agentpb.StartActionRequest_MySQLShowTableStatusParams{
				Dsn:   dsn,
				Table: table,
			},
		},
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}

	agent.channel.SendRequest(aRequest)
	return nil
}

// StartMySQLShowIndexAction starts mysql-show-index action on pmm-agent.
func (r *Registry) StartMySQLShowIndexAction(ctx context.Context, id, pmmAgentID, dsn, table string) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MysqlShowIndexParams{
			MysqlShowIndexParams: &agentpb.StartActionRequest_MySQLShowIndexParams{
				Dsn:   dsn,
				Table: table,
			},
		},
	}

	agent, err := r.get(pmmAgentID)
	if err != nil {
		return err
	}

	agent.channel.SendRequest(aRequest)
	return nil
}

// StopAction stops action with given given id.
// TODO: Extract it from here. Where...?
func (r *Registry) StopAction(ctx context.Context, actionID string) error {
	agent, err := r.get(actionID)
	if err != nil {
		return err
	}

	agent.channel.SendRequest(&agentpb.StopActionRequest{ActionId: actionID})
	return nil
}

// check interfaces
var (
	_ prom.Collector = (*Registry)(nil)
)
