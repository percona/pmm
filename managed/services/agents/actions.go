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
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/ha"
)

var (
	defaultActionTimeout      = durationpb.New(10 * time.Second)
	defaultQueryActionTimeout = durationpb.New(15 * time.Second) // should be less than checks.resultTimeout
	defaultPtActionTimeout    = durationpb.New(30 * time.Second) // Percona-toolkit action timeout
)

// ActionsService handles sending actions to pmm agents.
type ActionsService struct {
	r         *Registry
	qanClient qanClient
	forwarder actionsForwarder
}

type actionsForwarder interface {
	ForwardServerMessage(ctx context.Context, pmmAgentID string, message *agentv1.ServerMessage) (*agentv1.AgentMessage, error)
	IsEnabled(ctx context.Context) bool
}

// NewActionsService creates new actions service.
// forwarder can be nil if HA forwarding is not enabled.
func NewActionsService(qanClient qanClient, r *Registry, fwd actionsForwarder) *ActionsService {
	return &ActionsService{
		r:         r,
		qanClient: qanClient,
		forwarder: fwd,
	}
}

// sendActionRequest sends an action request to an agent, with HA forwarding support.
// It tries local agent first, then forwards if HA is enabled, and retries locally if forwarding suggests reconnection.
func (s *ActionsService) sendActionRequest(ctx context.Context, pmmAgentID string, request *agentv1.StartActionRequest) error {
	// Try to get agent locally
	pmmAgent, err := s.r.get(pmmAgentID)
	if err != nil {
		// Agent not local, try forwarding if HA is enabled
		if s.forwarder != nil && s.forwarder.IsEnabled(ctx) {
			// Wrap in ServerMessage
			serverMsg := &agentv1.ServerMessage{
				Id: 0, // Will be set by the receiving server
				Payload: &agentv1.ServerMessage_StartAction{
					StartAction: request,
				},
			}

			_, forwardErr := s.forwarder.ForwardServerMessage(ctx, pmmAgentID, serverMsg)
			// If forwarding suggests agent may have reconnected, retry locally
			if errors.Is(forwardErr, ha.ErrAgentMayHaveReconnected) {
				if pmmAgent, err = s.r.get(pmmAgentID); err == nil {
					_, err = pmmAgent.channel.SendAndWaitResponse(request)
					return err
				}
			}
			return forwardErr
		}
		return err
	}

	// Agent is local, send directly
	_, err = pmmAgent.channel.SendAndWaitResponse(request)
	return err
}

// StartMySQLExplainAction starts MySQL EXPLAIN Action on pmm-agent.
func (s *ActionsService) StartMySQLExplainAction(
	ctx context.Context,
	id string,
	pmmAgentID string,
	serviceID string,
	dsn string,
	queryID string,
	placeholders []string,
	format agentv1.MysqlExplainOutputFormat,
	files map[string]string,
	tdp *models.DelimiterPair,
	tlsSkipVerify bool,
) error {
	if queryID == "" {
		return status.Error(codes.FailedPrecondition, "query or query_id is required")
	}

	var q, schema string
	res, err := s.qanClient.ExplainFingerprintByQueryID(ctx, serviceID, queryID)
	if err != nil {
		return err
	}

	if res.PlaceholdersCount != uint32(len(placeholders)) {
		return status.Error(codes.FailedPrecondition, "placeholders count is not correct")
	}
	q = res.ExplainFingerprint

	sc, err := s.qanClient.SchemaByQueryID(ctx, serviceID, queryID)
	if err != nil {
		return err
	}
	schema = sc.Schema

	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_MysqlExplainParams{
			MysqlExplainParams: &agentv1.StartActionRequest_MySQLExplainParams{
				Dsn:          dsn,
				Query:        q,
				Values:       placeholders,
				Schema:       schema,
				OutputFormat: format,
				TlsFiles: &agentv1.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
				TlsSkipVerify: tlsSkipVerify,
			},
		},
		Timeout: defaultActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartMySQLShowCreateTableAction starts mysql-show-create-table action on pmm-agent.
func (s *ActionsService) StartMySQLShowCreateTableAction(ctx context.Context, id, pmmAgentID, dsn, table string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error { //nolint:lll
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_MysqlShowCreateTableParams{
			MysqlShowCreateTableParams: &agentv1.StartActionRequest_MySQLShowCreateTableParams{
				Dsn:   dsn,
				Table: table,
				TlsFiles: &agentv1.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
				TlsSkipVerify: tlsSkipVerify,
			},
		},
		Timeout: defaultActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartMySQLShowTableStatusAction starts mysql-show-table-status action on pmm-agent.
func (s *ActionsService) StartMySQLShowTableStatusAction(ctx context.Context, id, pmmAgentID, dsn, table string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error { //nolint:lll
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_MysqlShowTableStatusParams{
			MysqlShowTableStatusParams: &agentv1.StartActionRequest_MySQLShowTableStatusParams{
				Dsn:   dsn,
				Table: table,
				TlsFiles: &agentv1.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
				TlsSkipVerify: tlsSkipVerify,
			},
		},
		Timeout: defaultActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartMySQLShowIndexAction starts mysql-show-index action on pmm-agent.
func (s *ActionsService) StartMySQLShowIndexAction(ctx context.Context, id, pmmAgentID, dsn, table string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error { //nolint:lll
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_MysqlShowIndexParams{
			MysqlShowIndexParams: &agentv1.StartActionRequest_MySQLShowIndexParams{
				Dsn:   dsn,
				Table: table,
				TlsFiles: &agentv1.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
				TlsSkipVerify: tlsSkipVerify,
			},
		},
		Timeout: defaultActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartPostgreSQLShowCreateTableAction starts postgresql-show-create-table action on pmm-agent.
func (s *ActionsService) StartPostgreSQLShowCreateTableAction(ctx context.Context, id, pmmAgentID, dsn, table string) error {
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_PostgresqlShowCreateTableParams{
			PostgresqlShowCreateTableParams: &agentv1.StartActionRequest_PostgreSQLShowCreateTableParams{
				Dsn:   dsn,
				Table: table,
			},
		},
		Timeout: defaultActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartPostgreSQLShowIndexAction starts postgresql-show-index action on pmm-agent.
func (s *ActionsService) StartPostgreSQLShowIndexAction(ctx context.Context, id, pmmAgentID, dsn, table string) error {
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_PostgresqlShowIndexParams{
			PostgresqlShowIndexParams: &agentv1.StartActionRequest_PostgreSQLShowIndexParams{
				Dsn:   dsn,
				Table: table,
			},
		},
		Timeout: defaultActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartMongoDBExplainAction starts MongoDB query explain action on pmm-agent.
func (s *ActionsService) StartMongoDBExplainAction(ctx context.Context, id, pmmAgentID, dsn, query string, files map[string]string, tdp *models.DelimiterPair) error {
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_MongodbExplainParams{
			MongodbExplainParams: &agentv1.StartActionRequest_MongoDBExplainParams{
				Dsn:   dsn,
				Query: query,
				TextFiles: &agentv1.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
			},
		},
		Timeout: defaultActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartMySQLQueryShowAction starts MySQL SHOW query action on pmm-agent.
func (s *ActionsService) StartMySQLQueryShowAction(ctx context.Context, id, pmmAgentID, dsn, query string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error { //nolint:lll
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_MysqlQueryShowParams{
			MysqlQueryShowParams: &agentv1.StartActionRequest_MySQLQueryShowParams{
				Dsn:   dsn,
				Query: query,
				TlsFiles: &agentv1.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
				TlsSkipVerify: tlsSkipVerify,
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartMySQLQuerySelectAction starts MySQL SELECT query action on pmm-agent.
func (s *ActionsService) StartMySQLQuerySelectAction(ctx context.Context, id, pmmAgentID, dsn, query string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error { //nolint:lll
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_MysqlQuerySelectParams{
			MysqlQuerySelectParams: &agentv1.StartActionRequest_MySQLQuerySelectParams{
				Dsn:   dsn,
				Query: query,
				TlsFiles: &agentv1.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
				TlsSkipVerify: tlsSkipVerify,
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartPostgreSQLQueryShowAction starts PostgreSQL SHOW query action on pmm-agent.
func (s *ActionsService) StartPostgreSQLQueryShowAction(ctx context.Context, id, pmmAgentID, dsn string) error {
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_PostgresqlQueryShowParams{
			PostgresqlQueryShowParams: &agentv1.StartActionRequest_PostgreSQLQueryShowParams{
				Dsn: dsn,
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartPostgreSQLQuerySelectAction starts PostgreSQL SELECT query action on pmm-agent.
func (s *ActionsService) StartPostgreSQLQuerySelectAction(ctx context.Context, id, pmmAgentID, dsn, query string) error {
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_PostgresqlQuerySelectParams{
			PostgresqlQuerySelectParams: &agentv1.StartActionRequest_PostgreSQLQuerySelectParams{
				Dsn:   dsn,
				Query: query,
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartMongoDBQueryGetParameterAction starts MongoDB getParameter query action on pmm-agent.
func (s *ActionsService) StartMongoDBQueryGetParameterAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error {
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_MongodbQueryGetparameterParams{
			MongodbQueryGetparameterParams: &agentv1.StartActionRequest_MongoDBQueryGetParameterParams{
				Dsn: dsn,
				TextFiles: &agentv1.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartMongoDBQueryBuildInfoAction starts MongoDB buildInfo query action on pmm-agent.
func (s *ActionsService) StartMongoDBQueryBuildInfoAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error {
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_MongodbQueryBuildinfoParams{
			MongodbQueryBuildinfoParams: &agentv1.StartActionRequest_MongoDBQueryBuildInfoParams{
				Dsn: dsn,
				TextFiles: &agentv1.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartMongoDBQueryGetCmdLineOptsAction starts MongoDB getCmdLineOpts query action on pmm-agent.
func (s *ActionsService) StartMongoDBQueryGetCmdLineOptsAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error { //nolint:lll
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_MongodbQueryGetcmdlineoptsParams{
			MongodbQueryGetcmdlineoptsParams: &agentv1.StartActionRequest_MongoDBQueryGetCmdLineOptsParams{
				Dsn: dsn,
				TextFiles: &agentv1.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartMongoDBQueryReplSetGetStatusAction starts MongoDB replSetGetStatus query action on pmm-agent.
func (s *ActionsService) StartMongoDBQueryReplSetGetStatusAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error { //nolint:lll
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_MongodbQueryReplsetgetstatusParams{
			MongodbQueryReplsetgetstatusParams: &agentv1.StartActionRequest_MongoDBQueryReplSetGetStatusParams{
				Dsn: dsn,
				TextFiles: &agentv1.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartMongoDBQueryGetDiagnosticDataAction starts MongoDB getDiagnosticData query action on pmm-agent.
func (s *ActionsService) StartMongoDBQueryGetDiagnosticDataAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error { //nolint:lll
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_MongodbQueryGetdiagnosticdataParams{
			MongodbQueryGetdiagnosticdataParams: &agentv1.StartActionRequest_MongoDBQueryGetDiagnosticDataParams{
				Dsn: dsn,
				TextFiles: &agentv1.TextFiles{
					Files:              files,
					TemplateLeftDelim:  tdp.Left,
					TemplateRightDelim: tdp.Right,
				},
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartPTSummaryAction starts pt-summary action on pmm-agent.
func (s *ActionsService) StartPTSummaryAction(ctx context.Context, id, pmmAgentID string) error {
	aRequest := &agentv1.StartActionRequest{
		ActionId: id,
		// Requires params to be passed, even empty, othervise request's marshal fail.
		Params: &agentv1.StartActionRequest_PtSummaryParams{
			PtSummaryParams: &agentv1.StartActionRequest_PTSummaryParams{},
		},
		Timeout: defaultPtActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, aRequest)
}

// StartPTPgSummaryAction starts pt-pg-summary action on the pmm-agent.
func (s *ActionsService) StartPTPgSummaryAction(ctx context.Context, id, pmmAgentID, address string, port uint16, username, password string) error {
	actionRequest := &agentv1.StartActionRequest{
		ActionId: id,
		Params: &agentv1.StartActionRequest_PtPgSummaryParams{
			PtPgSummaryParams: &agentv1.StartActionRequest_PTPgSummaryParams{
				Host:     address,
				Port:     uint32(port),
				Username: username,
				Password: password,
			},
		},
		Timeout: defaultPtActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, actionRequest)
}

// StartPTMongoDBSummaryAction starts pt-mongodb-summary action on the pmm-agent.
func (s *ActionsService) StartPTMongoDBSummaryAction(ctx context.Context, id, pmmAgentID, address string, port uint16, username, password string) error {
	// Action request data that'll be sent to agent
	actionRequest := &agentv1.StartActionRequest{
		ActionId: id,
		// Proper params that'll will be passed to the command on the agent's side, even empty, othervise request's marshal fail.
		Params: &agentv1.StartActionRequest_PtMongodbSummaryParams{
			PtMongodbSummaryParams: &agentv1.StartActionRequest_PTMongoDBSummaryParams{
				Host:     address,
				Port:     uint32(port),
				Username: username,
				Password: password,
			},
		},
		Timeout: defaultPtActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, actionRequest)
}

// StartPTMySQLSummaryAction starts pt-mysql-summary action on the pmm-agent.
// The pt-mysql-summary's execution may require some of the following params: host, port, socket, username, password.
func (s *ActionsService) StartPTMySQLSummaryAction(ctx context.Context, id, pmmAgentID, address string, port uint16, socket, username, password string) error {
	actionRequest := &agentv1.StartActionRequest{
		ActionId: id,
		// Proper params that'll will be passed to the command on the agent's side.
		Params: &agentv1.StartActionRequest_PtMysqlSummaryParams{
			PtMysqlSummaryParams: &agentv1.StartActionRequest_PTMySQLSummaryParams{
				Host:     address,
				Port:     uint32(port),
				Socket:   socket,
				Username: username,
				Password: password,
			},
		},
		Timeout: defaultPtActionTimeout,
	}

	return s.sendActionRequest(ctx, pmmAgentID, actionRequest)
}

// StopAction stops action with given id.
func (s *ActionsService) StopAction(_ context.Context, actionID string) error {
	// TODO Seems that we have a bug here, we passing actionID to the method that expects pmmAgentID
	agent, err := s.r.get(actionID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(&agentv1.StopActionRequest{ActionId: actionID})
	return err
}
