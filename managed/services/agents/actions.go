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
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/managed/models"
)

var (
	defaultActionTimeout      = durationpb.New(10 * time.Second)
	defaultQueryActionTimeout = durationpb.New(15 * time.Second) // should be less than checks.resultTimeout
	defaultPtActionTimeout    = durationpb.New(30 * time.Second) // Percona-toolkit action timeout
)

// ActionsService handles sending actions to pmm agents.
type ActionsService struct {
	r *Registry
}

// NewActionsService creates new actions service.
func NewActionsService(r *Registry) *ActionsService {
	return &ActionsService{
		r: r,
	}
}

// StartMySQLExplainAction starts MySQL EXPLAIN Action on pmm-agent.
func (s *ActionsService) StartMySQLExplainAction(ctx context.Context, id, pmmAgentID, dsn, query string, format agentpb.MysqlExplainOutputFormat, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error {
	agent, err := s.r.get(pmmAgentID)
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
func (s *ActionsService) StartMySQLShowCreateTableAction(ctx context.Context, id, pmmAgentID, dsn, table string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error {
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMySQLShowTableStatusAction starts mysql-show-table-status action on pmm-agent.
func (s *ActionsService) StartMySQLShowTableStatusAction(ctx context.Context, id, pmmAgentID, dsn, table string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error {
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMySQLShowIndexAction starts mysql-show-index action on pmm-agent.
func (s *ActionsService) StartMySQLShowIndexAction(ctx context.Context, id, pmmAgentID, dsn, table string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error {
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartPostgreSQLShowCreateTableAction starts postgresql-show-create-table action on pmm-agent.
func (s *ActionsService) StartPostgreSQLShowCreateTableAction(ctx context.Context, id, pmmAgentID, dsn, table string) error {
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartPostgreSQLShowIndexAction starts postgresql-show-index action on pmm-agent.
func (s *ActionsService) StartPostgreSQLShowIndexAction(ctx context.Context, id, pmmAgentID, dsn, table string) error {
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMongoDBExplainAction starts MongoDB query explain action on pmm-agent.
func (s *ActionsService) StartMongoDBExplainAction(ctx context.Context, id, pmmAgentID, dsn, query string, files map[string]string, tdp *models.DelimiterPair) error {
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMySQLQueryShowAction starts MySQL SHOW query action on pmm-agent.
func (s *ActionsService) StartMySQLQueryShowAction(ctx context.Context, id, pmmAgentID, dsn, query string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error {
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMySQLQuerySelectAction starts MySQL SELECT query action on pmm-agent.
func (s *ActionsService) StartMySQLQuerySelectAction(ctx context.Context, id, pmmAgentID, dsn, query string, files map[string]string, tdp *models.DelimiterPair, tlsSkipVerify bool) error {
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartPostgreSQLQueryShowAction starts PostgreSQL SHOW query action on pmm-agent.
func (s *ActionsService) StartPostgreSQLQueryShowAction(ctx context.Context, id, pmmAgentID, dsn string) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_PostgresqlQueryShowParams{
			PostgresqlQueryShowParams: &agentpb.StartActionRequest_PostgreSQLQueryShowParams{
				Dsn: dsn,
			},
		},
		Timeout: defaultQueryActionTimeout,
	}

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartPostgreSQLQuerySelectAction starts PostgreSQL SELECT query action on pmm-agent.
func (s *ActionsService) StartPostgreSQLQuerySelectAction(ctx context.Context, id, pmmAgentID, dsn, query string) error {
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMongoDBQueryGetParameterAction starts MongoDB getParameter query action on pmm-agent.
func (s *ActionsService) StartMongoDBQueryGetParameterAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error {
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMongoDBQueryBuildInfoAction starts MongoDB buildInfo query action on pmm-agent.
func (s *ActionsService) StartMongoDBQueryBuildInfoAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error {
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMongoDBQueryGetCmdLineOptsAction starts MongoDB getCmdLineOpts query action on pmm-agent.
func (s *ActionsService) StartMongoDBQueryGetCmdLineOptsAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error {
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMongoDBQueryReplSetGetStatusAction starts MongoDB replSetGetStatus query action on pmm-agent.
func (s *ActionsService) StartMongoDBQueryReplSetGetStatusAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MongodbQueryReplsetgetstatusParams{
			MongodbQueryReplsetgetstatusParams: &agentpb.StartActionRequest_MongoDBQueryReplSetGetStatusParams{
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartMongoDBQueryGetDiagnosticDataAction starts MongoDB getDiagnosticData query action on pmm-agent.
func (s *ActionsService) StartMongoDBQueryGetDiagnosticDataAction(ctx context.Context, id, pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		Params: &agentpb.StartActionRequest_MongodbQueryGetdiagnosticdataParams{
			MongodbQueryGetdiagnosticdataParams: &agentpb.StartActionRequest_MongoDBQueryGetDiagnosticDataParams{
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartPTSummaryAction starts pt-summary action on pmm-agent.
func (s *ActionsService) StartPTSummaryAction(ctx context.Context, id, pmmAgentID string) error {
	aRequest := &agentpb.StartActionRequest{
		ActionId: id,
		// Requires params to be passed, even empty, othervise request's marshal fail.
		Params: &agentpb.StartActionRequest_PtSummaryParams{
			PtSummaryParams: &agentpb.StartActionRequest_PTSummaryParams{},
		},
		Timeout: defaultPtActionTimeout,
	}

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(aRequest)
	return err
}

// StartPTPgSummaryAction starts pt-pg-summary action on the pmm-agent.
func (s *ActionsService) StartPTPgSummaryAction(ctx context.Context, id, pmmAgentID, address string, port uint16, username, password string) error {
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

	pmmAgent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = pmmAgent.channel.SendAndWaitResponse(actionRequest)
	return err
}

// StartPTMongoDBSummaryAction starts pt-mongodb-summary action on the pmm-agent.
func (s *ActionsService) StartPTMongoDBSummaryAction(ctx context.Context, id, pmmAgentID, address string, port uint16, username, password string) error {
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
	pmmAgent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = pmmAgent.channel.SendAndWaitResponse(actionRequest)
	return err
}

// StartPTMySQLSummaryAction starts pt-mysql-summary action on the pmm-agent.
// The pt-mysql-summary's execution may require some of the following params: host, port, socket, username, password.
func (s *ActionsService) StartPTMySQLSummaryAction(ctx context.Context, id, pmmAgentID, address string, port uint16, socket, username, password string) error {
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

	pmmAgent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}
	_, err = pmmAgent.channel.SendAndWaitResponse(actionRequest)
	return err
}

// StopAction stops action with given given id.
func (s *ActionsService) StopAction(ctx context.Context, actionID string) error {
	// TODO Seems that we have a bug here, we passing actionID to the method that expects pmmAgentID
	agent, err := s.r.get(actionID)
	if err != nil {
		return err
	}
	_, err = agent.channel.SendAndWaitResponse(&agentpb.StopActionRequest{ActionId: actionID})
	return err
}
