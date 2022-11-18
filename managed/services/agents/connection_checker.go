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
	"fmt"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/logger"
	"github.com/percona/pmm/version"
)

var checkExternalExporterConnectionPMMVersion = version.MustParse("2.14.99")

// ConnectionChecker checks if connection can be established to service.
type ConnectionChecker struct {
	r *Registry
}

// NewConnectionChecker creates new connection checker.
func NewConnectionChecker(r *Registry) *ConnectionChecker {
	return &ConnectionChecker{
		r: r,
	}
}

// CheckConnectionToService sends request to pmm-agent to check connection to service.
func (c *ConnectionChecker) CheckConnectionToService(ctx context.Context, q *reform.Querier, service *models.Service, agent *models.Agent) error {
	l := logger.Get(ctx)
	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > 4*time.Second {
			l.Warnf("CheckConnectionToService took %s.", dur)
		}
	}()

	pmmAgentID := pointer.GetString(agent.PMMAgentID)
	if !agent.PushMetrics && (service.ServiceType == models.ExternalServiceType || service.ServiceType == models.HAProxyServiceType) {
		pmmAgentID = models.PMMServerAgentID
	}

	// Skip check connection to external exporter with old pmm-agent.
	if service.ServiceType == models.ExternalServiceType || service.ServiceType == models.HAProxyServiceType {
		isCheckConnSupported, err := isExternalExporterConnectionCheckSupported(q, pmmAgentID)
		if err != nil {
			return err
		}

		if !isCheckConnSupported {
			return nil
		}
	}

	pmmAgent, err := c.r.get(pmmAgentID)
	if err != nil {
		return err
	}

	request, err := connectionRequest(q, service, agent)
	if err != nil {
		return err
	}

	var sanitizedDSN string
	for _, word := range redactWords(agent) {
		sanitizedDSN = strings.ReplaceAll(request.Dsn, word, "****")
	}
	l.Infof("CheckConnectionRequest: type: %s, DSN: %s timeout: %s.", request.Type, sanitizedDSN, request.Timeout)
	resp, err := pmmAgent.channel.SendAndWaitResponse(request)
	if err != nil {
		return err
	}
	l.Infof("CheckConnection response: %+v.", resp)

	switch service.ServiceType {
	case models.MySQLServiceType:
		tableCount := resp.(*agentpb.CheckConnectionResponse).GetStats().GetTableCount()
		agent.TableCount = &tableCount
		l.Debugf("Updating table count: %d.", tableCount)
		if err = q.Update(agent); err != nil {
			return errors.Wrap(err, "failed to update table count")
		}
	case models.ExternalServiceType, models.HAProxyServiceType:
	case models.PostgreSQLServiceType:
	case models.MongoDBServiceType:
	case models.ProxySQLServiceType:
		// nothing yet

	default:
		return errors.Errorf("unhandled Service type %s", service.ServiceType)
	}

	msg := resp.(*agentpb.CheckConnectionResponse).Error
	switch msg {
	case "":
		return nil
	case context.Canceled.Error(), context.DeadlineExceeded.Error():
		msg = fmt.Sprintf("timeout (%s)", msg)
	}
	return status.Error(codes.FailedPrecondition, fmt.Sprintf("Connection check failed: %s.", msg))
}

func connectionRequest(q *reform.Querier, service *models.Service, agent *models.Agent) (*agentpb.CheckConnectionRequest, error) {
	var request *agentpb.CheckConnectionRequest
	switch service.ServiceType {
	case models.MySQLServiceType:
		tdp := agent.TemplateDelimiters(service)
		request = &agentpb.CheckConnectionRequest{
			Type:    inventorypb.ServiceType_MYSQL_SERVICE,
			Dsn:     agent.DSN(service, 2*time.Second, service.DatabaseName, nil),
			Timeout: durationpb.New(3 * time.Second),
			TextFiles: &agentpb.TextFiles{
				Files:              agent.Files(),
				TemplateLeftDelim:  tdp.Left,
				TemplateRightDelim: tdp.Right,
			},
			TlsSkipVerify: agent.TLSSkipVerify,
		}
	case models.PostgreSQLServiceType:
		tdp := agent.TemplateDelimiters(service)
		request = &agentpb.CheckConnectionRequest{
			Type:    inventorypb.ServiceType_POSTGRESQL_SERVICE,
			Dsn:     agent.DSN(service, 2*time.Second, service.DatabaseName, nil),
			Timeout: durationpb.New(3 * time.Second),
			TextFiles: &agentpb.TextFiles{
				Files:              agent.Files(),
				TemplateLeftDelim:  tdp.Left,
				TemplateRightDelim: tdp.Right,
			},
		}
	case models.MongoDBServiceType:
		tdp := agent.TemplateDelimiters(service)
		request = &agentpb.CheckConnectionRequest{
			Type:    inventorypb.ServiceType_MONGODB_SERVICE,
			Dsn:     agent.DSN(service, 2*time.Second, service.DatabaseName, nil),
			Timeout: durationpb.New(3 * time.Second),
			TextFiles: &agentpb.TextFiles{
				Files:              agent.Files(),
				TemplateLeftDelim:  tdp.Left,
				TemplateRightDelim: tdp.Right,
			},
		}
	case models.ProxySQLServiceType:
		request = &agentpb.CheckConnectionRequest{
			Type:    inventorypb.ServiceType_PROXYSQL_SERVICE,
			Dsn:     agent.DSN(service, 2*time.Second, service.DatabaseName, nil),
			Timeout: durationpb.New(3 * time.Second),
		}
	case models.ExternalServiceType:
		exporterURL, err := agent.ExporterURL(q)
		if err != nil {
			return nil, err
		}

		request = &agentpb.CheckConnectionRequest{
			Type:    inventorypb.ServiceType_EXTERNAL_SERVICE,
			Dsn:     exporterURL,
			Timeout: durationpb.New(3 * time.Second),
		}
	case models.HAProxyServiceType:
		exporterURL, err := agent.ExporterURL(q)
		if err != nil {
			return nil, err
		}

		request = &agentpb.CheckConnectionRequest{
			Type:    inventorypb.ServiceType_HAPROXY_SERVICE,
			Dsn:     exporterURL,
			Timeout: durationpb.New(3 * time.Second),
		}
	default:
		return nil, errors.Errorf("unhandled Service type %s", service.ServiceType)
	}
	return request, nil
}

func isExternalExporterConnectionCheckSupported(q *reform.Querier, pmmAgentID string) (bool, error) {
	pmmAgent, err := models.FindAgentByID(q, pmmAgentID)
	if err != nil {
		return false, fmt.Errorf("failed to get PMM Agent: %s", err)
	}
	pmmAgentVersion, err := version.Parse(*pmmAgent.Version)
	if err != nil {
		return false, fmt.Errorf("failed to parse PMM agent version %q: %s", *pmmAgent.Version, err)
	}

	if pmmAgentVersion.Less(checkExternalExporterConnectionPMMVersion) {
		return false, nil
	}
	return true, nil
}
