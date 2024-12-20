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
func (h *Handler) Run(stream agentv1.AgentService_ConnectServer) error {
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

	ticker := time.NewTicker(10 * time.Second)
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
				h.r.unregister(agent.id, disconnectReason)
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
					if err := h.stateChanged(ctx, p); err != nil {
						l.Errorf("%+v", err)
					}

					agent.channel.Send(&channel.ServerResponse{
						ID:      req.ID,
						Payload: &agentv1.StateChangedResponse{},
					})
				})

			case *agentv1.QANCollectRequest:
				pprof.Do(ctx, pprof.Labels("request", "QANCollectRequest"), func(ctx context.Context) {
					if err := h.qanClient.Collect(ctx, p.MetricsBucket); err != nil {
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

func (h *Handler) updateAgentStatusForChildren(ctx context.Context, agentID string, status inventoryv1.AgentStatus) error {
	return h.db.InTransaction(func(t *reform.TX) error {
		agents, err := models.FindAgents(t.Querier, models.AgentFilters{
			PMMAgentID: agentID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to get pmm-agent's child agents")
		}
		for _, agent := range agents {
			if err := updateAgentStatus(ctx, t.Querier, agent.AgentID, status, uint32(pointer.GetUint16(agent.ListenPort)), agent.ProcessExecPath, nil); err != nil {
				return errors.Wrap(err, "failed to update agent's status")
			}
		}
		return nil
	})
}

func (h *Handler) stateChanged(ctx context.Context, req *agentv1.StateChangedRequest) error {
	var PMMAgentID string

	errTX := h.db.InTransaction(func(tx *reform.TX) error {
		var agentIDs []string
		var err error
		req.AgentId = strings.TrimPrefix(req.AgentId, "/agent_id/")
		PMMAgentID, agentIDs, err = h.r.roster.get(req.AgentId)
		if err != nil {
			return err
		}

		for _, agentID := range agentIDs {
			err := updateAgentStatus(
				ctx,
				tx.Querier,
				agentID,
				req.Status,
				req.ListenPort,
				pointer.ToStringOrNil(req.ProcessExecPath),
				pointer.ToStringOrNil(req.Version))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if errTX != nil {
		return errTX
	}

	h.vmdb.RequestConfigurationUpdate()
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

// SetAllAgentsStatusUnknown goes through all pmm-agents and sets status to UNKNOWN.
func (h *Handler) SetAllAgentsStatusUnknown(ctx context.Context) error {
	agentType := models.PMMAgentType
	agents, err := models.FindAgents(h.db.Querier, models.AgentFilters{AgentType: &agentType})
	if err != nil {
		return errors.Wrap(err, "failed to get pmm-agents")
	}
	for _, agent := range agents {
		if !h.r.IsConnected(agent.AgentID) {
			err = h.updateAgentStatusForChildren(ctx, agent.AgentID, inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN)
			if err != nil {
				return err
			}
		}
	}
	return nil
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
	l := logger.Get(ctx)
	l.Debugf("updateAgentStatus: %s %s %d", agentID, status, listenPort)

	agent := &models.Agent{AgentID: agentID}
	err := q.Reload(agent)

	// agent can be already deleted, but we still can receive status message from pmm-agent.
	if errors.Is(err, reform.ErrNoRows) {
		if status == inventoryv1.AgentStatus_AGENT_STATUS_STOPPING || status == inventoryv1.AgentStatus_AGENT_STATUS_DONE {
			return nil
		}

		l.Warnf("Failed to select Agent by ID for (%s, %s).", agentID, status)
	}
	if err != nil {
		return errors.Wrap(err, "failed to select Agent by ID")
	}

	agent.Status = status.String()
	agent.ProcessExecPath = processExecPath
	agent.ListenPort = pointer.ToUint16(uint16(listenPort))
	if version != nil {
		agent.Version = version
	}
	if err = q.Update(agent); err != nil {
		return errors.Wrap(err, "failed to update Agent")
	}
	return nil
}
