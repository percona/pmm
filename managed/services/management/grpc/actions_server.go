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

package grpc

import (
	"context"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
	"github.com/percona/pmm/version"
)

type actionsServer struct {
	a  *agents.ActionsService
	db *reform.DB
	l  *logrus.Entry

	managementpb.UnimplementedActionsServer
}

var (
	pmmAgent2100 = version.MustParse("2.10.0")
	pmmAgent2150 = version.MustParse("2.15.0")
)

// NewActionsServer creates Management Actions Server.
func NewActionsServer(a *agents.ActionsService, db *reform.DB) managementpb.ActionsServer { //nolint:ireturn
	l := logrus.WithField("component", "actions.go")
	return &actionsServer{a: a, db: db, l: l}
}

// GetAction gets an action result.
func (s *actionsServer) GetAction(ctx context.Context, req *managementpb.GetActionRequest) (*managementpb.GetActionResponse, error) { //nolint:revive
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

		if pmmAgentID, err = models.FindPmmAgentIDToRunActionOrJob(pmmAgentID, agents); err != nil {
			return err
		}

		if dsn, _, err = models.FindDSNByServiceIDandPMMAgentID(tx.Querier, serviceID, pmmAgentID, database); err != nil {
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

func (s *actionsServer) prepareServiceActionWithFiles(serviceID, pmmAgentID, database string) (*models.ActionResult, string, map[string]string, *models.DelimiterPair, error) { //nolint:lll
	var res *models.ActionResult
	var dsn string
	var files map[string]string
	var tdp *models.DelimiterPair
	e := s.db.InTransaction(func(tx *reform.TX) error {
		svc, err := models.FindServiceByID(tx.Querier, serviceID)
		if err != nil {
			return err
		}

		pmmAgents, err := models.FindPMMAgentsForService(tx.Querier, serviceID)
		if err != nil {
			return err
		}

		if pmmAgentID, err = models.FindPmmAgentIDToRunActionOrJob(pmmAgentID, pmmAgents); err != nil {
			return err
		}

		var agent *models.Agent
		if dsn, agent, err = models.FindDSNByServiceIDandPMMAgentID(tx.Querier, serviceID, pmmAgentID, database); err != nil {
			return err
		}

		tdp = agent.TemplateDelimiters(svc)
		files = agent.Files()

		res, err = models.CreateActionResult(tx.Querier, pmmAgentID)
		return err
	})
	if e != nil {
		return nil, "", nil, nil, e
	}
	return res, dsn, files, tdp, nil
}

// StartMySQLExplainAction starts MySQL EXPLAIN Action with traditional output.
//
//nolint:lll
func (s *actionsServer) StartMySQLExplainAction(ctx context.Context, req *managementpb.StartMySQLExplainActionRequest) (*managementpb.StartMySQLExplainActionResponse, error) {
	res, dsn, files, tdp, err := s.prepareServiceActionWithFiles(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	agents, err := models.FindAgents(s.db.Querier, models.AgentFilters{ServiceID: req.ServiceId, PMMAgentID: req.PmmAgentId, AgentType: pointerToAgentType(models.MySQLdExporterType)})
	if err != nil {
		return nil, err
	}
	if len(agents) != 1 {
		return nil, status.Errorf(codes.FailedPrecondition, "Cannot find right agent")
	}

	err = s.a.StartMySQLExplainAction(ctx, res.ID, res.PMMAgentID, req.ServiceId, dsn, req.Query, //nolint:staticcheck
		req.QueryId, req.Placeholders, agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT, files, tdp, agents[0].TLSSkipVerify)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLExplainActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLExplainJSONAction starts MySQL EXPLAIN Action with JSON output.
//
//nolint:lll
func (s *actionsServer) StartMySQLExplainJSONAction(ctx context.Context, req *managementpb.StartMySQLExplainJSONActionRequest) (*managementpb.StartMySQLExplainJSONActionResponse, error) {
	res, dsn, files, tdp, err := s.prepareServiceActionWithFiles(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	agents, err := models.FindAgents(s.db.Querier, models.AgentFilters{ServiceID: req.ServiceId, PMMAgentID: req.PmmAgentId, AgentType: pointerToAgentType(models.MySQLdExporterType)})
	if err != nil {
		return nil, err
	}
	if len(agents) != 1 {
		return nil, status.Errorf(codes.FailedPrecondition, "Cannot find right agent")
	}

	err = s.a.StartMySQLExplainAction(ctx, res.ID, res.PMMAgentID, req.ServiceId, dsn, req.Query, //nolint:staticcheck
		req.QueryId, req.Placeholders, agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_JSON, files, tdp, agents[0].TLSSkipVerify)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLExplainJSONActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLExplainTraditionalJSONAction starts MySQL EXPLAIN Action with traditional JSON output.
//
//nolint:lll
func (s *actionsServer) StartMySQLExplainTraditionalJSONAction(ctx context.Context, req *managementpb.StartMySQLExplainTraditionalJSONActionRequest) (*managementpb.StartMySQLExplainTraditionalJSONActionResponse, error) {
	res, dsn, files, tdp, err := s.prepareServiceActionWithFiles(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	agents, err := models.FindAgents(s.db.Querier, models.AgentFilters{ServiceID: req.ServiceId, PMMAgentID: req.PmmAgentId, AgentType: pointerToAgentType(models.MySQLdExporterType)})
	if err != nil {
		return nil, err
	}
	if len(agents) != 1 {
		return nil, status.Errorf(codes.FailedPrecondition, "Cannot find right agent")
	}

	err = s.a.StartMySQLExplainAction(ctx, res.ID, res.PMMAgentID, req.ServiceId, dsn, req.Query, //nolint:staticcheck
		req.QueryId, req.Placeholders, agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_TRADITIONAL_JSON, files, tdp, agents[0].TLSSkipVerify)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLExplainTraditionalJSONActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLShowCreateTableAction starts MySQL SHOW CREATE TABLE Action.
//
//nolint:lll
func (s *actionsServer) StartMySQLShowCreateTableAction(ctx context.Context, req *managementpb.StartMySQLShowCreateTableActionRequest) (*managementpb.StartMySQLShowCreateTableActionResponse, error) {
	res, dsn, files, tdp, err := s.prepareServiceActionWithFiles(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	agents, err := models.FindAgents(s.db.Querier, models.AgentFilters{ServiceID: req.ServiceId, PMMAgentID: req.PmmAgentId, AgentType: pointerToAgentType(models.MySQLdExporterType)})
	if err != nil {
		return nil, err
	}
	if len(agents) != 1 {
		return nil, status.Errorf(codes.FailedPrecondition, "Cannot find right agent")
	}

	err = s.a.StartMySQLShowCreateTableAction(ctx, res.ID, res.PMMAgentID, dsn, req.TableName, files, tdp, agents[0].TLSSkipVerify)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLShowCreateTableActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLShowTableStatusAction starts MySQL SHOW TABLE STATUS Action.
//
//nolint:lll
func (s *actionsServer) StartMySQLShowTableStatusAction(ctx context.Context, req *managementpb.StartMySQLShowTableStatusActionRequest) (*managementpb.StartMySQLShowTableStatusActionResponse, error) {
	res, dsn, files, tdp, err := s.prepareServiceActionWithFiles(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	agents, err := models.FindAgents(s.db.Querier, models.AgentFilters{ServiceID: req.ServiceId, PMMAgentID: req.PmmAgentId, AgentType: pointerToAgentType(models.MySQLdExporterType)})
	if err != nil {
		return nil, err
	}
	if len(agents) != 1 {
		return nil, status.Errorf(codes.FailedPrecondition, "Cannot find right agent")
	}

	err = s.a.StartMySQLShowTableStatusAction(ctx, res.ID, res.PMMAgentID, dsn, req.TableName, files, tdp, agents[0].TLSSkipVerify)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLShowTableStatusActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMySQLShowIndexAction starts MySQL SHOW INDEX Action.
//
//nolint:lll
func (s *actionsServer) StartMySQLShowIndexAction(ctx context.Context, req *managementpb.StartMySQLShowIndexActionRequest) (*managementpb.StartMySQLShowIndexActionResponse, error) {
	res, dsn, files, tdp, err := s.prepareServiceActionWithFiles(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	agents, err := models.FindAgents(s.db.Querier, models.AgentFilters{ServiceID: req.ServiceId, PMMAgentID: req.PmmAgentId, AgentType: pointerToAgentType(models.MySQLdExporterType)})
	if err != nil {
		return nil, err
	}
	if len(agents) != 1 {
		return nil, status.Errorf(codes.FailedPrecondition, "Cannot find right agent")
	}

	err = s.a.StartMySQLShowIndexAction(ctx, res.ID, res.PMMAgentID, dsn, req.TableName, files, tdp, agents[0].TLSSkipVerify)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMySQLShowIndexActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartPostgreSQLShowCreateTableAction starts PostgreSQL SHOW CREATE TABLE Action.
//
//nolint:lll
func (s *actionsServer) StartPostgreSQLShowCreateTableAction(ctx context.Context, req *managementpb.StartPostgreSQLShowCreateTableActionRequest) (*managementpb.StartPostgreSQLShowCreateTableActionResponse, error) {
	res, dsn, err := s.prepareServiceAction(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	err = s.a.StartPostgreSQLShowCreateTableAction(ctx, res.ID, res.PMMAgentID, dsn, req.TableName)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartPostgreSQLShowCreateTableActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartPostgreSQLShowIndexAction starts PostgreSQL SHOW INDEX Action.
//
//nolint:lll
func (s *actionsServer) StartPostgreSQLShowIndexAction(ctx context.Context, req *managementpb.StartPostgreSQLShowIndexActionRequest) (*managementpb.StartPostgreSQLShowIndexActionResponse, error) {
	res, dsn, err := s.prepareServiceAction(req.ServiceId, req.PmmAgentId, req.Database)
	if err != nil {
		return nil, err
	}

	err = s.a.StartPostgreSQLShowIndexAction(ctx, res.ID, res.PMMAgentID, dsn, req.TableName)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartPostgreSQLShowIndexActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartMongoDBExplainAction starts MongoDB Explain action.
func (s *actionsServer) StartMongoDBExplainAction(ctx context.Context, req *managementpb.StartMongoDBExplainActionRequest) (
	*managementpb.StartMongoDBExplainActionResponse, error,
) {
	// Explain action must be executed against the admin database
	res, dsn, files, tdp, err := s.prepareServiceActionWithFiles(req.ServiceId, req.PmmAgentId, "admin")
	if err != nil {
		return nil, err
	}

	err = s.a.StartMongoDBExplainAction(ctx, res.ID, res.PMMAgentID, dsn, req.Query, files, tdp)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartMongoDBExplainActionResponse{
		PmmAgentId: req.PmmAgentId,
		ActionId:   res.ID,
	}, nil
}

// StartPTSummaryAction starts pt-summary action.
func (s *actionsServer) StartPTSummaryAction(ctx context.Context, req *managementpb.StartPTSummaryActionRequest) (*managementpb.StartPTSummaryActionResponse, error) {
	agents, err := models.FindPMMAgentsRunningOnNode(s.db.Querier, req.NodeId)
	if err != nil {
		s.l.Warnf("StartPTSummaryAction: %s", err)
		return nil, err
	}
	if len(agents) == 0 {
		return nil, status.Error(codes.NotFound, "no pmm-agent running on this node")
	}

	agents = models.FindPMMAgentsForVersion(s.l, agents, pmmAgent2100)
	if len(agents) == 0 {
		return nil, status.Error(codes.NotFound, "all available agents are outdated")
	}

	agentID, err := models.FindPmmAgentIDToRunActionOrJob(req.PmmAgentId, agents)
	if err != nil {
		return nil, err
	}

	res, err := models.CreateActionResult(s.db.Querier, agentID)
	if err != nil {
		return nil, err
	}

	err = s.a.StartPTSummaryAction(ctx, res.ID, agentID)
	if err != nil {
		return nil, err
	}

	return &managementpb.StartPTSummaryActionResponse{
		PmmAgentId: agentID,
		ActionId:   res.ID,
	}, nil
}

func pointerToAgentType(agentType models.AgentType) *models.AgentType {
	return &agentType
}

// StartPTPgSummaryAction starts pt-pg-summary (PostgreSQL) action and returns the pointer to the response message
//
//nolint:lll
func (s *actionsServer) StartPTPgSummaryAction(ctx context.Context, req *managementpb.StartPTPgSummaryActionRequest) (*managementpb.StartPTPgSummaryActionResponse, error) {
	service, err := models.FindServiceByID(s.db.Querier, req.ServiceId)
	if err != nil {
		return nil, err
	}

	node, err := models.FindNodeByID(s.db.Querier, service.NodeID)
	if err != nil {
		return nil, err
	}

	var pmmAgentID string
	switch node.NodeType {
	case models.RemoteNodeType:
		pmmAgentID = models.PMMServerAgentID
	default:
		pmmAgents, err := models.FindPMMAgentsRunningOnNode(s.db.Querier, service.NodeID)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "No pmm-agent running node %s", service.NodeID)
		}
		pmmAgents = models.FindPMMAgentsForVersion(s.l, pmmAgents, pmmAgent2150)
		if len(pmmAgents) == 0 {
			return nil, status.Error(codes.FailedPrecondition, "all available agents are outdated")
		}
		pmmAgentID, err = models.FindPmmAgentIDToRunActionOrJob(req.PmmAgentId, pmmAgents)
		if err != nil {
			return nil, err
		}
	}

	res, err := models.CreateActionResult(s.db.Querier, pmmAgentID)
	if err != nil {
		return nil, err
	}

	agentFilter := models.AgentFilters{ServiceID: req.ServiceId, AgentType: pointerToAgentType(models.PostgresExporterType)}
	postgresExporters, err := models.FindAgents(s.db.Querier, agentFilter)
	if err != nil {
		return nil, err
	}

	exportersCount := len(postgresExporters)
	if exportersCount < 1 {
		return nil, status.Errorf(codes.FailedPrecondition, "No postgres exporter")
	}
	if exportersCount > 1 {
		return nil, status.Errorf(codes.FailedPrecondition, "Found more than one postgres exporter")
	}

	if pointer.GetString(service.Socket) != "" {
		service.Address = service.Socket
	}

	err = s.a.StartPTPgSummaryAction(ctx, res.ID, pmmAgentID, pointer.GetString(service.Address), pointer.GetUint16(service.Port),
		pointer.GetString(postgresExporters[0].Username), pointer.GetString(postgresExporters[0].Password))
	if err != nil {
		return nil, err
	}

	return &managementpb.StartPTPgSummaryActionResponse{PmmAgentId: pmmAgentID, ActionId: res.ID}, nil
}

// StartPTMongoDBSummaryAction starts pt-mongodb-summary action and returns the pointer to the response message
//
//nolint:lll
func (s *actionsServer) StartPTMongoDBSummaryAction(ctx context.Context, req *managementpb.StartPTMongoDBSummaryActionRequest) (*managementpb.StartPTMongoDBSummaryActionResponse, error) {
	// Need to get the service id's pointer to retrieve the list of agent pointers therefrom
	// to get the particular agentID from the request.
	service, err := models.FindServiceByID(s.db.Querier, req.ServiceId)
	if err != nil {
		return nil, err
	}

	pmmAgents, err := models.FindPMMAgentsRunningOnNode(s.db.Querier, service.NodeID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "No pmm-agent running on this node")
	}

	pmmAgentID, err := models.FindPmmAgentIDToRunActionOrJob(req.PmmAgentId, pmmAgents)
	if err != nil {
		return nil, err
	}

	res, err := models.CreateActionResult(s.db.Querier, pmmAgentID)
	if err != nil {
		return nil, err
	}

	// Exporters to be filtered by service ID and agent type
	agentFilter := models.AgentFilters{
		PMMAgentID: "", NodeID: "",
		ServiceID: req.ServiceId, AgentType: pointerToAgentType(models.MongoDBExporterType),
	}

	// Need to get the mongoDB exporters to get the username and password therefrom
	mongoDBExporters, err := models.FindAgents(s.db.Querier, agentFilter)
	if err != nil {
		return nil, err
	}

	exportersCount := len(mongoDBExporters)

	// Must be only one result
	if exportersCount < 1 {
		return nil, status.Errorf(codes.FailedPrecondition, "No mongoDB exporter")
	}

	if exportersCount > 1 {
		return nil, status.Errorf(codes.FailedPrecondition, "Found more than one mongoDB exporter")
	}

	// Starts the pt-pg-summary with the host address, port, username and password
	err = s.a.StartPTMongoDBSummaryAction(ctx, res.ID, pmmAgentID, pointer.GetString(service.Address), pointer.GetUint16(service.Port),
		pointer.GetString(mongoDBExporters[0].Username), pointer.GetString(mongoDBExporters[0].Password))
	if err != nil {
		return nil, err
	}

	return &managementpb.StartPTMongoDBSummaryActionResponse{PmmAgentId: pmmAgentID, ActionId: res.ID}, nil
}

// StartPTMySQLSummaryAction starts pt-mysql-summary action and returns the pointer to the response message
//
//nolint:lll
func (s *actionsServer) StartPTMySQLSummaryAction(ctx context.Context, req *managementpb.StartPTMySQLSummaryActionRequest) (*managementpb.StartPTMySQLSummaryActionResponse, error) {
	service, err := models.FindServiceByID(s.db.Querier, req.ServiceId)
	if err != nil {
		return nil, err
	}

	node, err := models.FindNodeByID(s.db.Querier, service.NodeID)
	if err != nil {
		return nil, err
	}

	var pmmAgentID string
	switch node.NodeType {
	case models.RemoteNodeType:
		// Remove this error after: https://jira.percona.com/browse/PMM-7562
		return nil, status.Errorf(codes.FailedPrecondition, "PTMySQL Summary doesn't work with remote instances yet")

		// pmmAgentID = models.PMMServerAgentID
	default:
		pmmAgents, err := models.FindPMMAgentsRunningOnNode(s.db.Querier, service.NodeID)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "No pmm-agent running node %s", service.NodeID)
		}
		pmmAgents = models.FindPMMAgentsForVersion(s.l, pmmAgents, pmmAgent2150)
		if len(pmmAgents) == 0 {
			return nil, status.Error(codes.FailedPrecondition, "all available agents are outdated")
		}
		pmmAgentID, err = models.FindPmmAgentIDToRunActionOrJob(req.PmmAgentId, pmmAgents)
		if err != nil {
			return nil, err
		}
	}

	res, err := models.CreateActionResult(s.db.Querier, pmmAgentID)
	if err != nil {
		return nil, err
	}

	agentFilter := models.AgentFilters{
		PMMAgentID: "", NodeID: "",
		ServiceID: req.ServiceId, AgentType: pointerToAgentType(models.MySQLdExporterType),
	}
	mysqldExporters, err := models.FindAgents(s.db.Querier, agentFilter)
	if err != nil {
		return nil, err
	}

	exportersCount := len(mysqldExporters)
	if exportersCount < 1 {
		return nil, status.Errorf(codes.FailedPrecondition, "No mysql exporter")
	}
	if exportersCount > 1 {
		return nil, status.Errorf(codes.FailedPrecondition, "Found more than one mysql exporter")
	}

	err = s.a.StartPTMySQLSummaryAction(ctx, res.ID, pmmAgentID, pointer.GetString(service.Address), pointer.GetUint16(service.Port),
		pointer.GetString(service.Socket), pointer.GetString(mysqldExporters[0].Username),
		pointer.GetString(mysqldExporters[0].Password))
	if err != nil {
		return nil, err
	}

	return &managementpb.StartPTMySQLSummaryActionResponse{PmmAgentId: pmmAgentID, ActionId: res.ID}, nil
}

// CancelAction stops an Action.
func (s *actionsServer) CancelAction(ctx context.Context, req *managementpb.CancelActionRequest) (*managementpb.CancelActionResponse, error) {
	ar, err := models.FindActionResultByID(s.db.Querier, req.ActionId)
	if err != nil {
		return nil, err
	}

	err = s.a.StopAction(ctx, ar.ID)
	if err != nil {
		return nil, err
	}

	return &managementpb.CancelActionResponse{}, nil
}
