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

package management

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	azurev1beta1 "github.com/percona/pmm/api/managementpb/azure"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/logger"
)

const (
	// https://docs.microsoft.com/en-us/azure/governance/resource-graph/concepts/query-language
	// https://docs.microsoft.com/en-us/azure/azure-monitor/essentials/metrics-supported
	// TODO: add pagination and filtering https://jira.percona.com/browse/PMM-7813
	azureDatabaseResourceQuery = string(`
		Resources
			| where type in~ (
				'Microsoft.DBforMySQL/servers',
				'Microsoft.DBforMySQL/flexibleServers',
				'Microsoft.DBforMariaDB/servers',
				'Microsoft.DBforPostgreSQL/servers',
				'Microsoft.DBforPostgreSQL/serversv2',
				'Microsoft.DBforPostgreSQL/flexibleServers'
			)
			| order by name asc
			| limit 1000
	`)
)

// AzureDatabaseService represents instance discovery service.
type AzureDatabaseService struct {
	l        *logrus.Entry
	db       *reform.DB
	registry agentsRegistry
	state    agentsStateUpdater
	cc       connectionChecker
	sib      serviceInfoBroker

	azurev1beta1.UnimplementedAzureDatabaseServer
}

// NewAzureDatabaseService creates new instance discovery service.
func NewAzureDatabaseService(db *reform.DB, registry agentsRegistry, state agentsStateUpdater, cc connectionChecker, sib serviceInfoBroker) *AzureDatabaseService { //nolint:lll
	return &AzureDatabaseService{
		l:        logrus.WithField("component", "management/azure_database"),
		db:       db,
		registry: registry,
		state:    state,
		cc:       cc,
		sib:      sib,
	}
}

// Enabled returns if service is enabled and can be used.
func (s *AzureDatabaseService) Enabled() bool {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.WithError(err).Error("can't get settings")
		return false
	}
	return settings.Azurediscover.Enabled
}

// AzureDatabaseInstanceData reflects Azure Database Instance Data of Discovery Response.
type AzureDatabaseInstanceData struct {
	ID            string                 `json:"id"`
	Location      string                 `json:"location"`
	Name          string                 `json:"name"`
	Properties    map[string]interface{} `json:"properties"`
	Tags          map[string]string      `json:"tags"`
	Sku           map[string]interface{} `json:"sku"`
	ResourceGroup string                 `json:"resourceGroup"`
	Type          string                 `json:"type"`
	Zones         string                 `json:"zones"`
}

func (s *AzureDatabaseService) getAzureClient(req *azurev1beta1.DiscoverAzureDatabaseRequest) (*armresourcegraph.Client, error) {
	credential, err := azidentity.NewClientSecretCredential(req.AzureTenantId, req.AzureClientId, req.AzureClientSecret, nil)
	if err != nil {
		return nil, err
	}

	// Create and authorize a ResourceGraph client
	client, err := armresourcegraph.NewClient(credential, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (s *AzureDatabaseService) fetchAzureDatabaseInstancesData(
	ctx context.Context,
	req *azurev1beta1.DiscoverAzureDatabaseRequest,
	client *armresourcegraph.Client,
) ([]AzureDatabaseInstanceData, error) {
	query := azureDatabaseResourceQuery
	resultFormat := armresourcegraph.ResultFormatObjectArray
	request := armresourcegraph.QueryRequest{
		Subscriptions: []*string{&req.AzureSubscriptionId},
		Query:         &query,
		Options: &armresourcegraph.QueryRequestOptions{
			ResultFormat: &resultFormat,
		},
	}

	// Run the query and get the results
	results, err := client.Resources(ctx, request, nil)
	if err != nil {
		return nil, err
	}

	d, err := json.Marshal(results)
	if err != nil {
		return nil, err
	}

	dataInst := struct {
		Data []AzureDatabaseInstanceData `json:"data"`
	}{}

	err = json.Unmarshal(d, &dataInst)
	if err != nil {
		return nil, err
	}

	return dataInst.Data, nil
}

// DiscoverAzureDatabase discovers database instances on Azure.
func (s *AzureDatabaseService) DiscoverAzureDatabase(
	ctx context.Context,
	req *azurev1beta1.DiscoverAzureDatabaseRequest,
) (*azurev1beta1.DiscoverAzureDatabaseResponse, error) {
	client, err := s.getAzureClient(req)
	if err != nil {
		return nil, err
	}

	dataInstData, err := s.fetchAzureDatabaseInstancesData(ctx, req, client)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}

	resp := azurev1beta1.DiscoverAzureDatabaseResponse{}

	for _, instance := range dataInstData {
		inst := azurev1beta1.DiscoverAzureDatabaseInstance{
			InstanceId:         instance.ID,
			Region:             instance.Location,
			ServiceName:        instance.Name,
			AzureResourceGroup: instance.ResourceGroup,
			Environment:        instance.Tags["environment"],
			Az:                 instance.Zones,
		}
		switch instance.Type {
		case "microsoft.dbformysql/servers",
			"microsoft.dbformysql/flexibleservers",
			"microsoft.dbformariadb/servers":
			inst.Type = azurev1beta1.DiscoverAzureDatabaseType_DISCOVER_AZURE_DATABASE_TYPE_MYSQL
		case "microsoft.dbforpostgresql/servers",
			"microsoft.dbforpostgresql/flexibleservers",
			"microsoft.dbforpostgresql/serversv2":
			inst.Type = azurev1beta1.DiscoverAzureDatabaseType_DISCOVER_AZURE_DATABASE_TYPE_POSTGRESQL
		default:
			inst.Type = azurev1beta1.DiscoverAzureDatabaseType_DISCOVER_AZURE_DATABASE_TYPE_INVALID
		}

		if val, ok := instance.Properties["administratorLogin"].(string); ok {
			inst.Username = fmt.Sprintf("%s@%s", val, instance.Name)
		}
		if val, ok := instance.Properties["fullyQualifiedDomainName"].(string); ok {
			inst.Address = val
		}
		if val, ok := instance.Sku["name"].(string); ok {
			inst.NodeModel = val
		}

		resp.AzureDatabaseInstance = append(resp.AzureDatabaseInstance, &inst)
	}

	return &resp, nil
}

// AddAzureDatabase add azure database to monitoring.
func (s *AzureDatabaseService) AddAzureDatabase(ctx context.Context, req *azurev1beta1.AddAzureDatabaseRequest) (*azurev1beta1.AddAzureDatabaseResponse, error) {
	l := logger.Get(ctx).WithField("component", "discover/azureDatabase")
	// tweak according to API docs
	if req.NodeName == "" {
		req.NodeName = req.InstanceId
	}
	if req.ServiceName == "" {
		req.ServiceName = req.InstanceId
	}

	// tweak according to API docs
	tablestatsGroupTableLimit := req.TablestatsGroupTableLimit
	if tablestatsGroupTableLimit == 0 {
		tablestatsGroupTableLimit = defaultTablestatsGroupTableLimit
	}
	if tablestatsGroupTableLimit < 0 {
		tablestatsGroupTableLimit = -1
	}

	var serviceType models.ServiceType
	var exporterType models.AgentType
	var qanAgentType models.AgentType

	switch req.Type {
	case azurev1beta1.DiscoverAzureDatabaseType_DISCOVER_AZURE_DATABASE_TYPE_MYSQL:
		serviceType = models.MySQLServiceType
		exporterType = models.MySQLdExporterType
		qanAgentType = models.QANMySQLPerfSchemaAgentType
	case azurev1beta1.DiscoverAzureDatabaseType_DISCOVER_AZURE_DATABASE_TYPE_POSTGRESQL:
		serviceType = models.PostgreSQLServiceType
		exporterType = models.PostgresExporterType
		qanAgentType = models.QANPostgreSQLPgStatementsAgentType
		tablestatsGroupTableLimit = 0
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Unsupported Azure Database type %q.", req.Type)
	}

	if e := s.db.InTransaction(func(tx *reform.TX) error {
		// add Remote Azure Database Node
		node, err := models.CreateNode(tx.Querier, models.RemoteAzureDatabaseNodeType, &models.CreateNodeParams{
			NodeName:     req.NodeName,
			NodeModel:    req.NodeModel,
			AZ:           req.Az,
			Address:      req.Address,
			Region:       &req.Region,
			CustomLabels: req.CustomLabels,
		})
		if err != nil {
			return err
		}
		l.Infof("Created Azure Database Node with NodeID: %s", node.NodeID)

		service, err := models.AddNewService(tx.Querier, serviceType, &models.AddDBMSServiceParams{
			ServiceName:  req.ServiceName,
			NodeID:       node.NodeID,
			Environment:  req.Environment,
			CustomLabels: req.CustomLabels,
			Address:      &req.Address,
			Port:         pointer.ToUint16(uint16(req.Port)),
		})
		if err != nil {
			return err
		}
		l.Infof("Added Azure Database Service %s with ServiceID: %s", service.ServiceType, service.ServiceID)

		if req.AzureDatabaseExporter {
			azureDatabaseExporter, err := models.CreateAgent(tx.Querier, models.AzureDatabaseExporterType, &models.CreateAgentParams{
				PMMAgentID:   models.PMMServerAgentID,
				ServiceID:    service.ServiceID,
				AzureOptions: models.AzureOptionsFromRequest(req),
			})
			if err != nil {
				return err
			}
			l.Infof("Created Azure Database Exporter with AgentID: %s", azureDatabaseExporter.AgentID)
		}

		metricsExporter, err := models.CreateAgent(tx.Querier, exporterType, &models.CreateAgentParams{
			PMMAgentID:                     models.PMMServerAgentID,
			ServiceID:                      service.ServiceID,
			Username:                       req.Username,
			Password:                       req.Password,
			TLS:                            req.Tls,
			TLSSkipVerify:                  req.TlsSkipVerify,
			TableCountTablestatsGroupLimit: tablestatsGroupTableLimit,
		})
		if err != nil {
			return err
		}
		l.Infof("Added %s with AgentID: %s", metricsExporter.AgentType, metricsExporter.AgentID)

		if !req.SkipConnectionCheck {
			if err = s.cc.CheckConnectionToService(ctx, tx.Querier, service, metricsExporter); err != nil {
				return err
			}
			if err = s.sib.GetInfoFromService(ctx, tx.Querier, service, metricsExporter); err != nil {
				return err
			}
		}

		if req.Qan {
			qanAgent, err := models.CreateAgent(tx.Querier, qanAgentType, &models.CreateAgentParams{
				PMMAgentID:            models.PMMServerAgentID,
				ServiceID:             service.ServiceID,
				Username:              req.Username,
				Password:              req.Password,
				TLS:                   req.Tls,
				TLSSkipVerify:         req.TlsSkipVerify,
				QueryExamplesDisabled: req.DisableQueryExamples,
			})
			if err != nil {
				return err
			}
			l.Infof("Added QAN %s with AgentID: %s", qanAgent.AgentType, qanAgent.AgentID)
		}

		return nil
	}); e != nil {
		return nil, e
	}

	s.state.RequestStateUpdate(ctx, models.PMMServerAgentID)
	return &azurev1beta1.AddAzureDatabaseResponse{}, nil
}
