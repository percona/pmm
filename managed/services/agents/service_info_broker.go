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
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/reform.v1"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/logger"
)

// ServiceInfoBroker helps query various information from services.
type ServiceInfoBroker struct {
	r *Registry
}

// NewServiceInfoBroker creates a new ServiceInfoBroker.
func NewServiceInfoBroker(r *Registry) *ServiceInfoBroker {
	return &ServiceInfoBroker{
		r: r,
	}
}

// ServiceInfoRequest creates a ServiceInfoRequest for a given service.
func serviceInfoRequest(q *reform.Querier, service *models.Service, agent *models.Agent) (*agentv1.ServiceInfoRequest, error) {
	var request *agentv1.ServiceInfoRequest

	pmmAgentVersion := models.ExtractPmmAgentVersionFromAgent(q, agent)
	switch service.ServiceType {
	case models.MySQLServiceType:
		tdp := agent.TemplateDelimiters(service)
		request = &agentv1.ServiceInfoRequest{
			Type:    inventoryv1.ServiceType_SERVICE_TYPE_MYSQL_SERVICE,
			Dsn:     agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: service.DatabaseName}, nil, pmmAgentVersion),
			Timeout: durationpb.New(3 * time.Second),
			TextFiles: &agentv1.TextFiles{
				Files:              agent.Files(),
				TemplateLeftDelim:  tdp.Left,
				TemplateRightDelim: tdp.Right,
			},
			TlsSkipVerify: agent.TLSSkipVerify,
		}
	case models.PostgreSQLServiceType:
		tdp := agent.TemplateDelimiters(service)
		sqlSniSupported, err := models.IsPostgreSQLSSLSniSupported(q, pointer.GetString(agent.PMMAgentID))
		if err != nil {
			return nil, err
		}
		request = &agentv1.ServiceInfoRequest{
			Type: inventoryv1.ServiceType_SERVICE_TYPE_POSTGRESQL_SERVICE,
			Dsn: agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: service.DatabaseName, PostgreSQLSupportsSSLSNI: sqlSniSupported},
				nil, pmmAgentVersion),
			Timeout: durationpb.New(3 * time.Second),
			TextFiles: &agentv1.TextFiles{
				Files:              agent.Files(),
				TemplateLeftDelim:  tdp.Left,
				TemplateRightDelim: tdp.Right,
			},
		}
	case models.MongoDBServiceType:
		tdp := agent.TemplateDelimiters(service)
		request = &agentv1.ServiceInfoRequest{
			Type:    inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE,
			Dsn:     agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: service.DatabaseName}, nil, pmmAgentVersion),
			Timeout: durationpb.New(3 * time.Second),
			TextFiles: &agentv1.TextFiles{
				Files:              agent.Files(),
				TemplateLeftDelim:  tdp.Left,
				TemplateRightDelim: tdp.Right,
			},
		}
	case models.ProxySQLServiceType:
		request = &agentv1.ServiceInfoRequest{
			Type:    inventoryv1.ServiceType_SERVICE_TYPE_PROXYSQL_SERVICE,
			Dsn:     agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: service.DatabaseName}, nil, pmmAgentVersion),
			Timeout: durationpb.New(3 * time.Second),
		}
	case models.ExternalServiceType:
		exporterURL, err := agent.ExporterURL(q)
		if err != nil {
			return nil, err
		}

		request = &agentv1.ServiceInfoRequest{
			Type:    inventoryv1.ServiceType_SERVICE_TYPE_EXTERNAL_SERVICE,
			Dsn:     exporterURL,
			Timeout: durationpb.New(3 * time.Second),
		}
	case models.HAProxyServiceType:
		exporterURL, err := agent.ExporterURL(q)
		if err != nil {
			return nil, err
		}

		request = &agentv1.ServiceInfoRequest{
			Type:    inventoryv1.ServiceType_SERVICE_TYPE_HAPROXY_SERVICE,
			Dsn:     exporterURL,
			Timeout: durationpb.New(3 * time.Second),
		}
	default:
		return nil, errors.Errorf("unhandled Service type %s", service.ServiceType)
	}
	return request, nil
}

// GetInfoFromService sends a request to pmm-agent to query information from a service.
func (c *ServiceInfoBroker) GetInfoFromService(ctx context.Context, q *reform.Querier, service *models.Service, agent *models.Agent) error {
	l := logger.Get(ctx)
	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > 4*time.Second {
			l.Warnf("GetInfoFromService took %s.", dur)
		}
	}()

	// External exporters and haproxy do not support this functionality.
	if service.ServiceType == models.ExternalServiceType || service.ServiceType == models.HAProxyServiceType {
		return nil
	}

	pmmAgentID := pointer.GetString(agent.PMMAgentID)
	pmmAgent, err := c.r.get(pmmAgentID)
	if err != nil {
		return err
	}

	request, err := serviceInfoRequest(q, service, agent)
	if err != nil {
		return err
	}

	sanitizedDSN := request.Dsn
	for _, word := range redactWords(agent) {
		sanitizedDSN = strings.ReplaceAll(sanitizedDSN, word, "****")
	}
	l.Infof("ServiceInfoRequest: type: %s, DSN: %s timeout: %s.", request.Type, sanitizedDSN, request.Timeout)

	resp, err := pmmAgent.channel.SendAndWaitResponse(request)
	if err != nil {
		return err
	}
	l.Infof("ServiceInfo response: %+v.", resp)

	sInfo, ok := resp.(*agentv1.ServiceInfoResponse)
	if !ok {
		return status.Error(codes.Internal, "failed to cast response to *agentv1.ServiceInfoResponse")
	}

	msg := sInfo.Error
	if msg == context.Canceled.Error() || msg == context.DeadlineExceeded.Error() {
		msg = fmt.Sprintf("timeout (%s)", msg)
		return status.Error(codes.FailedPrecondition, fmt.Sprintf("failed to get connection service info: %s.", msg))
	}

	stype := service.ServiceType
	switch stype {
	case models.MySQLServiceType:
		agent.MySQLOptions.TableCount = &sInfo.TableCount
		l.Debugf("Updating table count: %d.", sInfo.TableCount)
		encryptedAgent := models.EncryptAgent(*agent)
		if err = q.Update(&encryptedAgent); err != nil {
			return errors.Wrap(err, "failed to update table count")
		}

		return updateServiceVersion(ctx, q, resp, service)
	case models.PostgreSQLServiceType:
		if agent.PostgreSQLOptions == nil {
			agent.PostgreSQLOptions = &models.PostgreSQLOptions{}
		}

		databaseList := sInfo.DatabaseList
		databaseCount := len(databaseList)
		excludedDatabaseList := postgresExcludedDatabases()
		excludedDatabaseCount := 0
		for _, e := range excludedDatabaseList {
			for _, v := range databaseList {
				if e == v {
					excludedDatabaseCount++
				}
			}
		}
		agent.PostgreSQLOptions.PGSMVersion = sInfo.PgsmVersion
		agent.PostgreSQLOptions.DatabaseCount = int32(databaseCount - excludedDatabaseCount)

		l.Debugf("Updating PostgreSQL options, database count: %d.", agent.PostgreSQLOptions.DatabaseCount)
		encryptedAgent := models.EncryptAgent(*agent)
		if err = q.Update(&encryptedAgent); err != nil {
			return errors.Wrap(err, "failed to update database count")
		}

		return updateServiceVersion(ctx, q, resp, service)
	case models.MongoDBServiceType,
		models.ProxySQLServiceType:
		return updateServiceVersion(ctx, q, resp, service)
	case models.ExternalServiceType, models.HAProxyServiceType:
		return nil
	default:
		return errors.Errorf("unhandled Service type %s", service.ServiceType)
	}
}

func updateServiceVersion(ctx context.Context, q *reform.Querier, resp agentv1.AgentResponsePayload, service *models.Service) error {
	l := logger.Get(ctx)

	version := resp.(*agentv1.ServiceInfoResponse).Version //nolint:forcetypeassert
	if version == "" {
		return nil
	}

	l.Debugf("Updating service version: %s.", version)
	service.Version = &version
	if err := q.Update(service); err != nil {
		return errors.Wrap(err, "failed to update service version")
	}

	return nil
}
