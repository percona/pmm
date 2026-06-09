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

package agents

import (
	"context"
	"fmt"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents/channel"
	"github.com/percona/pmm/utils/logger"
)

const defaultAgentPingInterval = 10 * time.Second

// Handler handles agent requests.
type Handler struct {
	db          *reform.DB
	r           *Registry
	vmdb        prometheusService
	qanClient   qanClient
	state       *StateUpdater
	jobsService jobsService
}

// NewHandler creates new agents handler.
func NewHandler(db *reform.DB, qanClient qanClient, vmdb prometheusService, registry *Registry, state *StateUpdater,
	jobsService jobsService,
) *Handler {
	h := &Handler{
		db:          db,
		r:           registry,
		vmdb:        vmdb,
		qanClient:   qanClient,
		state:       state,
		jobsService: jobsService,
	}
	return h
}

// Run takes over pmm-agent gRPC stream and runs it until completion.
func (h *Handler) Run(stream agentv1.AgentService_ConnectServer) error { //nolint:gocognit
	disconnectReason := "unknown"

	ctx := stream.Context()
	l := logger.Get(ctx)
	agent, err := h.r.register(stream)
	if err != nil {
		disconnectReason = "auth"
		return err
	}
	defer func() {
		l.Infof("Disconnecting client: %s.", disconnectReason)
	}()

	// run pmm-agent state update loop for the current agent.
	go h.state.runStateChangeHandler(ctx, agent)

	h.state.RequestStateUpdate(ctx, agent.id)

	ticker := time.NewTicker(defaultAgentPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := h.r.ping(ctx, agent)
			if err != nil {
				l.Errorf("agent %s ping: %v", agent.id, err)
			}

		// see unregister and Kick methods
		case <-agent.kickChan:
			// already unregistered, no need to call unregister method
			l.Warn("Kicked.")
			disconnectReason = "kicked"
			err = status.Errorf(codes.Aborted, "Kicked.")
			return err

		case req := <-agent.channel.Requests():
			if req == nil {
				disconnectReason = "done"
				err = agent.channel.Wait()
				h.r.unregister(ctx, agent.id, disconnectReason)
				if err != nil {
					l.Error(errors.WithStack(err))
				}
				return nil
			}

			switch p := req.Payload.(type) {
			case *agentv1.Ping:
				agent.channel.Send(&channel.ServerResponse{
					ID: req.ID,
					Payload: &agentv1.Pong{
						CurrentTime: timestamppb.Now(),
					},
				})

			case *agentv1.StateChangedRequest:
				pprof.Do(ctx, pprof.Labels("request", "StateChangedRequest"), func(ctx context.Context) {
					err := h.stateChanged(ctx, p)
					if err != nil {
						l.Errorf("%+v", err)
					}

					agent.channel.Send(&channel.ServerResponse{
						ID:      req.ID,
						Payload: &agentv1.StateChangedResponse{},
					})
				})

			case *agentv1.QANCollectRequest:
				pprof.Do(ctx, pprof.Labels("request", "QANCollectRequest"), func(ctx context.Context) {
					err := h.qanClient.Collect(ctx, p.MetricsBucket)
					if err != nil {
						l.Errorf("%+v", err)
					}

					agent.channel.Send(&channel.ServerResponse{
						ID:      req.ID,
						Payload: &agentv1.QANCollectResponse{},
					})
				})

			case *agentv1.ActionResultRequest:
				// TODO: PMM-3978: In the future we need to merge action parts before we send it to the storage.
				err := models.ChangeActionResult(h.db.Querier, p.ActionId, agent.id, p.Error, string(p.Output), p.Done)
				if err != nil {
					l.Warnf("Failed to change action: %+v", err)
				}

				if !p.Done && p.Error != "" {
					l.Warnf("Action was done with an error: %v.", p.Error)
				}

				agent.channel.Send(&channel.ServerResponse{
					ID:      req.ID,
					Payload: &agentv1.ActionResultResponse{},
				})

			case *agentv1.JobResult:
				h.jobsService.handleJobResult(ctx, l, p)
			case *agentv1.JobProgress:
				h.jobsService.handleJobProgress(ctx, p)
			case nil:
				l.Errorf("Unexpected request: %+v.", req)
			}
		}
	}
}

func (h *Handler) stateChanged(ctx context.Context, req *agentv1.StateChangedRequest) error {
	var PMMAgentID string
	var portsChanged bool
	l := logger.Get(ctx).WithField("component", "agents/handler")

	errTX := h.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var agentIDs []string
		var err error
		sAgentID := strings.TrimPrefix(req.AgentId, "/agent_id/")
		PMMAgentID, agentIDs, err = h.r.roster.get(sAgentID)
		if err != nil {
			return err
		}

		for _, agentID := range agentIDs {
			// Check if port changed before updating
			if checkPortChanged(tx.Querier, agentID, req.ListenPort) {
				portsChanged = true
			}

			err := updateAgentStatus(
				ctx,
				tx.Querier,
				agentID,
				req.Status,
				req.ListenPort,
				pointer.ToStringOrNil(req.ProcessExecPath),
				pointer.ToStringOrNil(req.Version),
			)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if errTX != nil {
		return errTX
	}

	// For port changes, force immediate synchronous config update to prevent
	// VictoriaMetrics from scraping stale ports (PMM-14267)
	if portsChanged {
		l.Debug("Listen port changed, forcing immediate VictoriaMetrics configuration update")
		err := h.vmdb.ForceConfigurationUpdate(ctx)
		if err != nil {
			return fmt.Errorf("failed to force configuration update: %w", err)
		}
	} else {
		// Normal async update
		h.vmdb.RequestConfigurationUpdate()
	}

	agent, err := models.FindAgentByID(h.db.Querier, PMMAgentID)
	if err != nil {
		return err
	}
	if agent.PMMAgentID == nil {
		return nil
	}

	h.state.RequestStateUpdate(ctx, *agent.PMMAgentID)
	return nil
}

// checkPortChanged checks if the agent's listen port is changing.
func checkPortChanged(q *reform.Querier, agentID string, newPort uint32) bool {
	agent, err := models.FindAgentByID(q, agentID)
	if err != nil {
		// Can't determine, assume no change
		return false
	}
	oldPort := pointer.GetUint16(agent.ListenPort)
	newPort16 := uint16(newPort) //nolint:gosec
	// Port changed if old port exists and is different from new port
	return oldPort != 0 && oldPort != newPort16
}

func updateAgentStatus(
	ctx context.Context,
	q *reform.Querier,
	agentID string,
	status inventoryv1.AgentStatus,
	listenPort uint32,
	processExecPath *string,
	version *string,
) error {
	l := logger.Get(ctx).WithField("component", "agents/handler")
	l.Debugf("updateAgentStatus: %s %s %d", agentID, status, listenPort)

	agent, err := models.FindAgentByID(q, agentID)

	// agent can be already deleted, but we still can receive status message from pmm-agent.
	if errors.Is(err, reform.ErrNoRows) {
		if status == inventoryv1.AgentStatus_AGENT_STATUS_STOPPING || status == inventoryv1.AgentStatus_AGENT_STATUS_DONE {
			return nil
		}

		l.Warnf("Failed to select Agent by ID for (%s, %s).", agentID, status)
	}
	if err != nil {
		return fmt.Errorf("failed to select Agent by ID: %w", err)
	}

	if agent.Disabled {
		if status != inventoryv1.AgentStatus_AGENT_STATUS_DONE {
			l.Debugf("Agent %s is disabled, but status is %s. Setting status to DONE.", agentID, status)
		}
		status = inventoryv1.AgentStatus_AGENT_STATUS_DONE
	}

	agent.Status = status.String()
	agent.ProcessExecPath = processExecPath
	agent.ListenPort = new(uint16(listenPort)) //nolint:gosec // port is uint16
	if version != nil {
		agent.Version = version
	}

	err = models.UpdateAgent(q, agent)
	if err != nil {
		return fmt.Errorf("failed to update Agent: %w", err)
	}

	return nil
}
