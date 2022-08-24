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

	"github.com/pkg/errors"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/logger"
)

// CredentialsSourceAgentInvoker requests from agent to parse credentials-source passed in request.
type CredentialsSourceAgentInvoker struct {
	r *Registry
}

// NewCredentialsSourceAgentInvoker creates new CredentialsSourceAgentInvoker request.
func NewCredentialsSourceAgentInvoker(r *Registry) *CredentialsSourceAgentInvoker {
	return &CredentialsSourceAgentInvoker{
		r: r,
	}
}

// InvokeAgent sends request (with file path) to pmm-agent to parse credentials-source file.
func (p *CredentialsSourceAgentInvoker) InvokeAgent(ctx context.Context, pmmAgentID, filePath string, serviceType models.ServiceType) (*models.CredentialsSourceParsingResult, error) {
	l := logger.Get(ctx)

	pmmAgent, err := p.r.get(pmmAgentID)
	if err != nil {
		return nil, err
	}

	defer func(t time.Time) {
		if dur := time.Since(t); dur > 5*time.Second {
			l.Warnf("Invoking agent took %s.", dur)
		}
	}(time.Now())

	request, err := createRequest(filePath, serviceType)
	if err != nil {
		l.Debugf("can't create ParseCredentialsSourceRequest %s", err)
		return nil, err
	}

	resp, err := pmmAgent.channel.SendAndWaitResponse(request)
	if err != nil {
		return nil, err
	}

	l.Infof("ParseCredentialsSource response from agent: %+v.", resp)
	parserResponse, ok := resp.(*agentpb.ParseCredentialsSourceResponse)
	if !ok {
		return nil, errors.New("wrong response from agent (not ParseCredentialsSourceResponse model)")
	}
	if parserResponse.Error != "" {
		return nil, errors.New(parserResponse.Error)
	}

	return &models.CredentialsSourceParsingResult{
		Username:      parserResponse.Username,
		Password:      parserResponse.Password,
		AgentPassword: parserResponse.AgentPassword,
		Host:          parserResponse.Host,
		Port:          parserResponse.Port,
		Socket:        parserResponse.Socket,
	}, nil
}

func createRequest(configPath string, serviceType models.ServiceType) (*agentpb.ParseCredentialsSourceRequest, error) {
	switch serviceType {
	case models.MySQLServiceType:
		return &agentpb.ParseCredentialsSourceRequest{
			ServiceType: inventorypb.ServiceType_MYSQL_SERVICE,
			FilePath:    configPath,
		}, nil
	case models.PostgreSQLServiceType:
		return &agentpb.ParseCredentialsSourceRequest{
			ServiceType: inventorypb.ServiceType_POSTGRESQL_SERVICE,
			FilePath:    configPath,
		}, nil
	case models.HAProxyServiceType:
		return &agentpb.ParseCredentialsSourceRequest{
			ServiceType: inventorypb.ServiceType_HAPROXY_SERVICE,
			FilePath:    configPath,
		}, nil
	case models.ExternalServiceType:
		return &agentpb.ParseCredentialsSourceRequest{
			ServiceType: inventorypb.ServiceType_EXTERNAL_SERVICE,
			FilePath:    configPath,
		}, nil
	case models.MongoDBServiceType:
		return &agentpb.ParseCredentialsSourceRequest{
			ServiceType: inventorypb.ServiceType_MONGODB_SERVICE,
			FilePath:    configPath,
		}, nil
	case models.ProxySQLServiceType:
		return &agentpb.ParseCredentialsSourceRequest{
			ServiceType: inventorypb.ServiceType_PROXYSQL_SERVICE,
			FilePath:    configPath,
		}, nil
	}

	return nil, errors.Errorf("unhandled service type %s", serviceType)
}
