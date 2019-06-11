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

package grpc

import (
	"context"

	"github.com/percona/pmm/api/managementpb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/agents"
)

type actionsServer struct {
	r  *agents.Registry
	db *reform.DB
}

// NewActionsServer creates Management Actions Server.
func NewActionsServer(r *agents.Registry, db *reform.DB) managementpb.ActionsServer {
	return &actionsServer{r, db}
}

// GetAction gets an action result.
//nolint:lll
func (s *actionsServer) GetAction(ctx context.Context, req *managementpb.GetActionRequest) (*managementpb.GetActionResponse, error) {
	res, err := models.FindActionResultByID(s.db.Querier, req.ActionId)
	if err != nil {
		return nil, err
	}

	return &managementpb.GetActionResponse{
		ActionId:   res.ID,
		PmmAgentId: res.PMMAgentID,
		Done:       res.Done,
		Error:      res.Error,
		Output:     res.Output,
	}, nil
}

// StartPTSummaryAction starts pt-summary action.
//nolint:lll,dupl
func (s *actionsServer) StartPTSummaryAction(ctx context.Context, req *managementpb.StartPTSummaryActionRequest) (*managementpb.StartPTSummaryActionResponse, error) {
	ag, err := models.FindPMMAgentsForNode(s.db.Querier, req.NodeId)
	if err != nil {
		return nil, err
	}

	req.PmmAgentId, err = models.FindPmmAgentIDToRunAction(req.PmmAgentId, ag)
	if err != nil {
		return nil, err
	}

	res, err := models.CreateActionResult(s.db.Querier, req.PmmAgentId)
	if err != nil {
		return nil, err
	}

	err = s.r.StartPTSummaryAction(ctx, res.ID, req.PmmAgentId, []string{})
	if err != nil {
		return nil, err
	}

	return &managementpb.StartPTSummaryActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartPTMySQLSummaryAction starts pt-mysql-summary action.
//nolint:lll,dupl
func (s *actionsServer) StartPTMySQLSummaryAction(ctx context.Context, req *managementpb.StartPTMySQLSummaryActionRequest) (*managementpb.StartPTMySQLSummaryActionResponse, error) {
	ag, err := models.FindPMMAgentsForService(s.db.Querier, req.ServiceId)
	if err != nil {
		return nil, err
	}

	req.PmmAgentId, err = models.FindPmmAgentIDToRunAction(req.PmmAgentId, ag)
	if err != nil {
		return nil, err
	}

	res, err := models.CreateActionResult(s.db.Querier, req.PmmAgentId)
	if err != nil {
		return nil, err
	}

	err = s.r.StartPTMySQLSummaryAction(ctx, res.ID, req.PmmAgentId, []string{})
	if err != nil {
		return nil, err
	}

	return &managementpb.StartPTMySQLSummaryActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLExplainAction starts mysql-explain action.
//nolint:lll,dupl
func (s *actionsServer) StartMySQLExplainAction(ctx context.Context, req *managementpb.StartMySQLExplainActionRequest) (*managementpb.StartMySQLExplainActionResponse, error) {
	ag, err := models.FindPMMAgentsForService(s.db.Querier, req.ServiceId)
	if err != nil {
		return nil, err
	}

	req.PmmAgentId, err = models.FindPmmAgentIDToRunAction(req.PmmAgentId, ag)
	if err != nil {
		return nil, err
	}

	dsn, err := models.FindDSNByServiceIDandPMMAgentID(s.db.Querier, req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	res, err := models.CreateActionResult(s.db.Querier, req.PmmAgentId)
	if err != nil {
		return nil, err
	}

	err = s.r.StartMySQLExplainAction(ctx, res.ID, req.PmmAgentId, dsn, req.Query)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLExplainActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLExplainJSONAction starts mysql-explain json action.
//nolint:lll,dupl
func (s *actionsServer) StartMySQLExplainJSONAction(ctx context.Context, req *managementpb.StartMySQLExplainJSONActionRequest) (*managementpb.StartMySQLExplainJSONActionResponse, error) {
	ag, err := models.FindPMMAgentsForService(s.db.Querier, req.ServiceId)
	if err != nil {
		return nil, err
	}

	req.PmmAgentId, err = models.FindPmmAgentIDToRunAction(req.PmmAgentId, ag)
	if err != nil {
		return nil, err
	}

	dsn, err := models.FindDSNByServiceIDandPMMAgentID(s.db.Querier, req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	res, err := models.CreateActionResult(s.db.Querier, req.PmmAgentId)
	if err != nil {
		return nil, err
	}

	err = s.r.StartMySQLExplainJSONAction(ctx, res.ID, req.PmmAgentId, dsn, req.Query)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLExplainJSONActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLShowCreateTableAction starts mysql-show-create-table action.
//nolint:lll,dupl
func (s *actionsServer) StartMySQLShowCreateTableAction(ctx context.Context, req *managementpb.StartMySQLShowCreateTableActionRequest) (*managementpb.StartMySQLShowCreateTableActionResponse, error) {
	ag, err := models.FindPMMAgentsForService(s.db.Querier, req.ServiceId)
	if err != nil {
		return nil, err
	}

	req.PmmAgentId, err = models.FindPmmAgentIDToRunAction(req.PmmAgentId, ag)
	if err != nil {
		return nil, err
	}

	dsn, err := models.FindDSNByServiceIDandPMMAgentID(s.db.Querier, req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	res, err := models.CreateActionResult(s.db.Querier, req.PmmAgentId)
	if err != nil {
		return nil, err
	}

	err = s.r.StartMySQLShowCreateTableAction(ctx, res.ID, req.PmmAgentId, dsn, req.TableName)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLShowCreateTableActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLShowTableStatusAction starts mysql-show-table-status action.
//nolint:lll,dupl
func (s *actionsServer) StartMySQLShowTableStatusAction(ctx context.Context, req *managementpb.StartMySQLShowTableStatusActionRequest) (*managementpb.StartMySQLShowTableStatusActionResponse, error) {
	ag, err := models.FindPMMAgentsForService(s.db.Querier, req.ServiceId)
	if err != nil {
		return nil, err
	}

	req.PmmAgentId, err = models.FindPmmAgentIDToRunAction(req.PmmAgentId, ag)
	if err != nil {
		return nil, err
	}

	dsn, err := models.FindDSNByServiceIDandPMMAgentID(s.db.Querier, req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	res, err := models.CreateActionResult(s.db.Querier, req.PmmAgentId)
	if err != nil {
		return nil, err
	}

	err = s.r.StartMySQLShowTableStatusAction(ctx, res.ID, req.PmmAgentId, dsn, req.TableName)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLShowTableStatusActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLShowIndexAction starts mysql-show-index action.
func (s *actionsServer) StartMySQLShowIndexAction(ctx context.Context, req *managementpb.StartMySQLShowIndexActionRequest) (*managementpb.StartMySQLShowIndexActionResponse, error) {
	panic("TODO")
}

// CancelAction stops an Action.
//nolint:lll
func (s *actionsServer) CancelAction(ctx context.Context, req *managementpb.CancelActionRequest) (*managementpb.CancelActionResponse, error) {
	ar, err := models.FindActionResultByID(s.db.Querier, req.ActionId)
	if err != nil {
		return nil, err
	}

	err = s.r.StopAction(ctx, ar.ID)
	if err != nil {
		return nil, err
	}

	return &managementpb.CancelActionResponse{}, nil
}
