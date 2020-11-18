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

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/version"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/agents"
)

type actionsServer struct {
	r  *agents.Registry
	db *reform.DB
	l  *logrus.Entry
}

var pmmAgent2100 = version.MustParse("2.10.0-HEAD") // TODO: Remove HEAD later once 2.11.0 is released.

// NewActionsServer creates Management Actions Server.
func NewActionsServer(r *agents.Registry, db *reform.DB) managementpb.ActionsServer {
	l := logrus.WithField("component", "actions")
	return &actionsServer{r, db, l}
}

// GetAction gets an action result.
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

func (s *actionsServer) prepareServiceAction(serviceID, pmmAgentID, database string) (*models.ActionResult, string, error) {
	var res *models.ActionResult
	var dsn string
	e := s.db.InTransaction(func(tx *reform.TX) error {
		agents, err := models.FindPMMAgentsForService(tx.Querier, serviceID)
		if err != nil {
			return err
		}

		if pmmAgentID, err = models.FindPmmAgentIDToRunAction(pmmAgentID, agents); err != nil {
			return err
		}

		if dsn, err = models.FindDSNByServiceIDandPMMAgentID(tx.Querier, serviceID, pmmAgentID, database); err != nil {
			return err
		}

		res, err = models.CreateActionResult(tx.Querier, pmmAgentID)
		return err
	})
	if e != nil {
		return nil, "", e
	}
	return res, dsn, nil
}

// StartMySQLExplainAction starts MySQL EXPLAIN Action with traditional output.
//nolint:lll
func (s *actionsServer) StartMySQLExplainAction(ctx context.Context, req *managementpb.StartMySQLExplainActionRequest) (*managementpb.StartMySQLExplainActionResponse, error) {
	res, dsn, err := s.prepareServiceAction(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	err = s.r.StartMySQLExplainAction(ctx, res.ID, res.PMMAgentID, dsn, req.Query, agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLExplainActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLExplainJSONAction starts MySQL EXPLAIN Action with JSON output.
//nolint:lll
func (s *actionsServer) StartMySQLExplainJSONAction(ctx context.Context, req *managementpb.StartMySQLExplainJSONActionRequest) (*managementpb.StartMySQLExplainJSONActionResponse, error) {
	res, dsn, err := s.prepareServiceAction(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	err = s.r.StartMySQLExplainAction(ctx, res.ID, res.PMMAgentID, dsn, req.Query, agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_JSON)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLExplainJSONActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLExplainTraditionalJSONAction starts MySQL EXPLAIN Action with traditional JSON output.
//nolint:lll
func (s *actionsServer) StartMySQLExplainTraditionalJSONAction(ctx context.Context, req *managementpb.StartMySQLExplainTraditionalJSONActionRequest) (*managementpb.StartMySQLExplainTraditionalJSONActionResponse, error) {
	res, dsn, err := s.prepareServiceAction(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	err = s.r.StartMySQLExplainAction(ctx, res.ID, res.PMMAgentID, dsn, req.Query, agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_TRADITIONAL_JSON)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLExplainTraditionalJSONActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLShowCreateTableAction starts MySQL SHOW CREATE TABLE Action.
//nolint:lll
func (s *actionsServer) StartMySQLShowCreateTableAction(ctx context.Context, req *managementpb.StartMySQLShowCreateTableActionRequest) (*managementpb.StartMySQLShowCreateTableActionResponse, error) {
	res, dsn, err := s.prepareServiceAction(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	err = s.r.StartMySQLShowCreateTableAction(ctx, res.ID, res.PMMAgentID, dsn, req.TableName)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLShowCreateTableActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLShowTableStatusAction starts MySQL SHOW TABLE STATUS Action.
//nolint:lll
func (s *actionsServer) StartMySQLShowTableStatusAction(ctx context.Context, req *managementpb.StartMySQLShowTableStatusActionRequest) (*managementpb.StartMySQLShowTableStatusActionResponse, error) {
	res, dsn, err := s.prepareServiceAction(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	err = s.r.StartMySQLShowTableStatusAction(ctx, res.ID, res.PMMAgentID, dsn, req.TableName)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLShowTableStatusActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLShowIndexAction starts MySQL SHOW INDEX Action.
//nolint:lll
func (s *actionsServer) StartMySQLShowIndexAction(ctx context.Context, req *managementpb.StartMySQLShowIndexActionRequest) (*managementpb.StartMySQLShowIndexActionResponse, error) {
	res, dsn, err := s.prepareServiceAction(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	err = s.r.StartMySQLShowIndexAction(ctx, res.ID, res.PMMAgentID, dsn, req.TableName)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLShowIndexActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartPostgreSQLShowCreateTableAction starts PostgreSQL SHOW CREATE TABLE Action.
//nolint:lll
func (s *actionsServer) StartPostgreSQLShowCreateTableAction(ctx context.Context, req *managementpb.StartPostgreSQLShowCreateTableActionRequest) (*managementpb.StartPostgreSQLShowCreateTableActionResponse, error) {
	res, dsn, err := s.prepareServiceAction(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	err = s.r.StartPostgreSQLShowCreateTableAction(ctx, res.ID, res.PMMAgentID, dsn, req.TableName)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartPostgreSQLShowCreateTableActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartPostgreSQLShowIndexAction starts PostgreSQL SHOW INDEX Action.
//nolint:lll
func (s *actionsServer) StartPostgreSQLShowIndexAction(ctx context.Context, req *managementpb.StartPostgreSQLShowIndexActionRequest) (*managementpb.StartPostgreSQLShowIndexActionResponse, error) {
	res, dsn, err := s.prepareServiceAction(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	err = s.r.StartPostgreSQLShowIndexAction(ctx, res.ID, res.PMMAgentID, dsn, req.TableName)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartPostgreSQLShowIndexActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMongoDBExplainAction starts MongoDB Explain action
//nolint:lll
func (s *actionsServer) StartMongoDBExplainAction(ctx context.Context, req *managementpb.StartMongoDBExplainActionRequest) (
	*managementpb.StartMongoDBExplainActionResponse, error) {
	// Explain action must be executed against the admin database
	res, dsn, err := s.prepareServiceAction(req.ServiceId, req.PmmAgentId, "admin")
	if err != nil {
		return nil, err
	}

	err = s.r.StartMongoDBExplainAction(ctx, res.ID, res.PMMAgentID, dsn, req.Query)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMongoDBExplainActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartPTSummaryAction starts pt-summary action.
//nolint:lll
func (s *actionsServer) StartPTSummaryAction(ctx context.Context, req *managementpb.StartPTSummaryActionRequest) (*managementpb.StartPTSummaryActionResponse, error) {
	agents, err := models.FindPMMAgentsRunningOnNode(s.db.Querier, req.NodeId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "No pmm-agent running on this node")
	}

	agents = models.FindPMMAgentsForVersion(s.l, agents, pmmAgent2100)
	if len(agents) == 0 {
		return nil, status.Error(codes.NotFound, "all available agents are outdated")
	}

	agentID, err := models.FindPmmAgentIDToRunAction(req.PmmAgentId, agents)
	if err != nil {
		return nil, err
	}

	res, err := models.CreateActionResult(s.db.Querier, agentID)
	if err != nil {
		return nil, err
	}

	err = s.r.StartPTSummaryAction(ctx, res.ID, agentID)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartPTSummaryActionResponse{
		PmmAgentId: agentID,
		ActionId:   res.ID,
	}, nil
}

// StartPTMySQLSummaryAction starts pt-mysql-summary action.
//nolint:lll
func (s *actionsServer) StartPTMySQLSummaryAction(context.Context, *managementpb.StartPTMySQLSummaryActionRequest) (*managementpb.StartPTMySQLSummaryActionResponse, error) {
	panic("not implemented yet")
}

// CancelAction stops an Action.
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
