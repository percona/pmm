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
	"errors"
	"slices"

	"github.com/AlekSi/pointer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/services/management/common"
)

// errDryRun rolls back the transaction of a dry run.
var errDryRun = errors.New("dry run")

// qanAgentTypes lists the QAN Agent types of each Service type.
var qanAgentTypes = map[models.ServiceType][]models.AgentType{
	models.MySQLServiceType:      {models.QANMySQLPerfSchemaAgentType, models.QANMySQLSlowlogAgentType},
	models.MongoDBServiceType:    {models.QANMongoDBProfilerAgentType, models.QANMongoDBMongologAgentType},
	models.PostgreSQLServiceType: {models.QANPostgreSQLPgStatementsAgentType, models.QANPostgreSQLPgStatMonitorAgentType},
}

// UpdateService changes the settings of a Service and its Agents in place, keeping their IDs.
func (s *ManagementService) UpdateService(ctx context.Context, req *managementv1.UpdateServiceRequest) (*managementv1.UpdateServiceResponse, error) {
	c, err := newServiceChange(req)
	if err != nil {
		return nil, err
	}

	current, err := s.findService(s.db.Querier, req.ServiceId)
	if err != nil {
		return nil, err
	}

	// Scheduled backups depend on the cluster; this mirrors the inventory ChangeService.
	clusterChanged := c.service.Cluster != nil && *c.service.Cluster != current.Cluster
	if clusterChanged && !req.DryRun {
		err = s.stm.RemoveScheduledTasks(ctx, s.db, &models.ChangeStandardLabelsParams{
			ServiceID: current.ServiceID,
			Cluster:   c.service.Cluster,
		})
		if err != nil {
			return nil, err
		}
	}

	res := &managementv1.UpdateServiceResponse{}
	var pmmAgentIDs []string
	var connectionChanged bool

	err = s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		u := &serviceUpdate{s: s, q: tx.Querier, c: c}

		err := u.run(ctx, current.ServiceID)
		if err != nil {
			return err
		}

		res.Before, res.After, res.Warning = u.before, u.after, u.warning
		pmmAgentIDs, connectionChanged = u.pmmAgentIDs, u.connectionChanged

		if req.DryRun {
			return errDryRun
		}

		return nil
	})
	if err != nil && !errors.Is(err, errDryRun) {
		return nil, err
	}

	if req.DryRun {
		return res, nil
	}

	for _, id := range pmmAgentIDs {
		s.state.RequestStateUpdate(ctx, id)
	}
	s.vmdb.RequestConfigurationUpdate()

	// The software versions are kept for MySQL and MongoDB only, as when the Service is added.
	versioned := current.ServiceType == models.MySQLServiceType || current.ServiceType == models.MongoDBServiceType
	if connectionChanged && versioned {
		s.vc.RequestSoftwareVersionsUpdate()
	}

	return res, nil
}

// findService finds a Service by ID or, if the value does not look like an ID, by name.
func (s *ManagementService) findService(q *reform.Querier, idOrName string) (*models.Service, error) {
	if LooksLikeID(idOrName) {
		return models.FindServiceByID(q, idOrName)
	}

	return models.FindServiceByName(q, idOrName)
}

// serviceUpdate applies a serviceChange to a Service and its Agents within one transaction.
type serviceUpdate struct {
	s *ManagementService
	q *reform.Querier
	c *serviceChange

	service  *models.Service
	agents   []*models.Agent
	exporter *models.Agent

	before            *managementv1.UniversalService
	after             *managementv1.UniversalService
	warning           string
	pmmAgentIDs       []string
	connectionChanged bool
}

func (u *serviceUpdate) run(ctx context.Context, serviceID string) error {
	var err error
	u.service, err = models.FindServiceByIDForUpdate(u.q, serviceID)
	if err != nil {
		return err
	}

	if u.service.ServiceType != u.c.serviceType {
		return status.Errorf(codes.InvalidArgument, "Service %s is a %s Service, not %s.",
			u.service.ServiceName, u.service.ServiceType, u.c.serviceType)
	}

	err = u.loadAgents()
	if err != nil {
		return err
	}

	u.before, err = u.universalService()
	if err != nil {
		return err
	}

	if u.c.service.Address != nil && u.service.ServiceType != models.ExternalServiceType {
		err = u.s.checkNodeIsEligible(ctx, pointer.GetString(u.exporter.PMMAgentID), *u.c.service.Address)
		if err != nil {
			return err
		}
	}

	u.service, err = models.ApplyServiceChange(u.q, u.service, &u.c.service)
	if err != nil {
		return err
	}

	err = u.changeAgents()
	if err != nil {
		return err
	}

	err = u.changeQANAgents(ctx)
	if err != nil {
		return err
	}

	err = u.loadAgents()
	if err != nil {
		return err
	}

	u.after, err = u.universalService()

	return err
}

// loadAgents loads the Agents of the Service and finds its exporter.
func (u *serviceUpdate) loadAgents() error {
	agents, err := models.FindAgents(u.q, models.AgentFilters{ServiceID: u.service.ServiceID})
	if err != nil {
		return err
	}

	u.agents = agents
	u.exporter = nil
	for _, agent := range agents {
		if agent.AgentType == u.c.exporterType {
			u.exporter = agent
		}

		pmmAgentID := pointer.GetString(agent.PMMAgentID)
		if pmmAgentID != "" && !slices.Contains(u.pmmAgentIDs, pmmAgentID) {
			u.pmmAgentIDs = append(u.pmmAgentIDs, pmmAgentID)
		}
	}

	if u.exporter == nil {
		return status.Errorf(codes.FailedPrecondition, "Service %s has no %s.", u.service.ServiceName, u.c.exporterType)
	}

	return nil
}

func (u *serviceUpdate) universalService() (*managementv1.UniversalService, error) {
	node, err := models.FindNodeByID(u.q, u.service.NodeID)
	if err != nil {
		return nil, err
	}

	return u.s.universalService(u.service, node.NodeName, u.agents)
}

// changeAgents applies the change to the existing Agents and checks the connection to the Service when it is affected.
func (u *serviceUpdate) changeAgents() error {
	exporterParams := u.c.agentParams(u.exporter)
	if u.c.metricsMode != nil {
		push, err := pushMetrics(u.q, u.exporter, *u.c.metricsMode)
		if err != nil {
			return err
		}
		exporterParams.ExporterOptions.PushMetrics = &push
	}

	if u.c.envVarNames != nil && len(*u.c.envVarNames) != 0 {
		err := common.ValidateMongoDBExporterEnvVarNames(*u.c.envVarNames, u.exporter.GrandfatheredEnvironmentVariableNames())
		if err != nil {
			return err
		}
	}

	exporter, err := models.ApplyAgentChange(u.q, u.exporter, exporterParams)
	if err != nil {
		return err
	}
	u.exporter = exporter

	for _, agent := range u.agents {
		if agent.AgentID == u.exporter.AgentID {
			continue
		}

		params := u.c.agentParams(agent)
		if params == nil {
			continue
		}

		_, err = models.ApplyAgentChange(u.q, agent, params)
		if err != nil {
			return err
		}
	}

	u.connectionChanged = u.c.service.IsEndpointChange() || exporterParams.AffectsConnection()

	return nil
}

// changeQANAgents adds and removes QAN Agents as requested, checking the connection to the Service first
// when the change affects it or adds an Agent, as adding the Service does.
func (u *serviceUpdate) changeQANAgents(ctx context.Context) error {
	var add, remove []models.AgentType
	for _, agentType := range qanAgentTypes[u.service.ServiceType] {
		requested, ok := u.c.qanAgents[agentType]
		if !ok {
			continue
		}

		exists := slices.ContainsFunc(u.agents, func(a *models.Agent) bool { return a.AgentType == agentType })
		switch {
		case requested && !exists:
			add = append(add, agentType)
		case !requested && exists:
			remove = append(remove, agentType)
		}
	}

	if !u.c.skipConnectionCheck && (u.connectionChanged || len(add) != 0) {
		err := u.s.cc.CheckConnectionToService(ctx, u.q, u.service, u.exporter)
		if err != nil {
			return err
		}

		err = u.s.sib.GetInfoFromService(ctx, u.q, u.service, u.exporter)
		if err != nil {
			return err
		}
	}

	// Without the pg_stat_monitor extension pg_stat_statements is used instead, as when the Service is added.
	if slices.Contains(add, models.QANPostgreSQLPgStatMonitorAgentType) {
		pgsmVersion := u.exporter.PostgreSQLOptions.PGSMVersion
		if pgsmVersion != nil && *pgsmVersion == "" {
			u.warning = "Could not to detect the pg_stat_monitor extension on your system. Falling back to the pg_stat_statements."
			add = slices.DeleteFunc(add, func(t models.AgentType) bool { return t == models.QANPostgreSQLPgStatMonitorAgentType })
			hasStatements := slices.ContainsFunc(u.agents, func(a *models.Agent) bool {
				return a.AgentType == models.QANPostgreSQLPgStatementsAgentType
			})
			if !hasStatements {
				add = append(add, models.QANPostgreSQLPgStatementsAgentType)
			}
			remove = slices.DeleteFunc(remove, func(t models.AgentType) bool { return t == models.QANPostgreSQLPgStatementsAgentType })
		}
	}

	template := u.qanTemplate()

	for _, agent := range u.agents {
		if !slices.Contains(remove, agent.AgentType) {
			continue
		}

		_, err := models.RemoveAgent(u.q, agent.AgentID, models.RemoveRestrict)
		if err != nil {
			return err
		}
	}

	for _, agentType := range add {
		_, err := models.CreateAgent(u.q, agentType, u.qanAgentParams(agentType, template))
		if err != nil {
			return err
		}
	}

	return nil
}

// qanTemplate returns an existing QAN Agent whose QAN settings a new QAN Agent takes over, if any.
func (u *serviceUpdate) qanTemplate() *models.Agent {
	for _, agent := range u.agents {
		if slices.Contains(qanAgentTypes[u.service.ServiceType], agent.AgentType) {
			return agent
		}
	}

	return nil
}

// qanAgentParams returns the parameters of a new QAN Agent, which connects to the Service the way its exporter does.
func (u *serviceUpdate) qanAgentParams(agentType models.AgentType, template *models.Agent) *models.CreateAgentParams {
	e := u.exporter
	params := &models.CreateAgentParams{
		PMMAgentID:    pointer.GetString(e.PMMAgentID),
		ServiceID:     u.service.ServiceID,
		Username:      pointer.GetString(e.Username),
		Password:      pointer.GetString(e.Password),
		TLS:           e.TLS,
		TLSSkipVerify: e.TLSSkipVerify,
		LogLevel:      pointer.GetString(e.LogLevel),
		MySQLOptions: models.MySQLOptions{
			TLSCa:          e.MySQLOptions.TLSCa,
			TLSCert:        e.MySQLOptions.TLSCert,
			TLSKey:         e.MySQLOptions.TLSKey,
			ExtraDSNParams: e.MySQLOptions.ExtraDSNParams,
		},
		MongoDBOptions: models.MongoDBOptions{
			TLSCertificateKey:             e.MongoDBOptions.TLSCertificateKey,
			TLSCertificateKeyFilePassword: e.MongoDBOptions.TLSCertificateKeyFilePassword,
			TLSCa:                         e.MongoDBOptions.TLSCa,
			AuthenticationMechanism:       e.MongoDBOptions.AuthenticationMechanism,
			AuthenticationDatabase:        e.MongoDBOptions.AuthenticationDatabase,
		},
		PostgreSQLOptions: models.PostgreSQLOptions{
			SSLCa:   e.PostgreSQLOptions.SSLCa,
			SSLCert: e.PostgreSQLOptions.SSLCert,
			SSLKey:  e.PostgreSQLOptions.SSLKey,
		},
	}

	if template != nil {
		params.LogLevel = pointer.GetString(template.LogLevel)
		params.QANOptions = models.QANOptions{
			MaxQueryLength:          template.QANOptions.MaxQueryLength,
			QueryExamplesDisabled:   template.QANOptions.QueryExamplesDisabled,
			CommentsParsingDisabled: template.QANOptions.CommentsParsingDisabled,
		}
	}

	if agentType == models.QANMySQLSlowlogAgentType {
		params.QANOptions.MaxQueryLogSize = defaultMaxSlowlogFileSize
	}

	if u.c.logLevel != nil {
		params.LogLevel = services.SpecifyLogLevel(*u.c.logLevel, minLogLevel(agentType))
	}

	q := u.c.qan
	if q.MaxQueryLength != nil {
		params.QANOptions.MaxQueryLength = *q.MaxQueryLength
	}
	if q.QueryExamplesDisabled != nil {
		params.QANOptions.QueryExamplesDisabled = *q.QueryExamplesDisabled
	}
	if q.CommentsParsingDisabled != nil {
		params.QANOptions.CommentsParsingDisabled = *q.CommentsParsingDisabled
	}
	if q.MaxQueryLogSize != nil && agentType == models.QANMySQLSlowlogAgentType {
		params.QANOptions.MaxQueryLogSize = *q.MaxQueryLogSize
	}

	return params
}

// agentParams returns the part of the change that applies to the given Agent, or nil if none does.
func (c *serviceChange) agentParams(agent *models.Agent) *models.ChangeAgentParams {
	isExporter := agent.AgentType == c.exporterType
	isQAN := slices.Contains(qanAgentTypes[c.serviceType], agent.AgentType)
	if !isExporter && !isQAN && agent.AgentType != models.RTAMongoDBAgentType {
		return nil
	}

	params := &models.ChangeAgentParams{
		Username:      c.username,
		Password:      c.password,
		TLS:           c.tls,
		TLSSkipVerify: c.tlsSkipVerify,
	}
	if c.logLevel != nil {
		params.LogLevel = new(services.SpecifyLogLevel(*c.logLevel, minLogLevel(agent.AgentType)))
	}

	switch c.serviceType {
	case models.MySQLServiceType:
		params.MySQLOptions = &models.ChangeMySQLOptions{
			TLSCa:          c.mysql.TLSCa,
			TLSCert:        c.mysql.TLSCert,
			TLSKey:         c.mysql.TLSKey,
			ExtraDSNParams: c.mysql.ExtraDSNParams,
		}
	case models.MongoDBServiceType:
		params.MongoDBOptions = &models.ChangeMongoDBOptions{
			TLSCertificateKey:             c.mongodb.TLSCertificateKey,
			TLSCertificateKeyFilePassword: c.mongodb.TLSCertificateKeyFilePassword,
			TLSCa:                         c.mongodb.TLSCa,
			AuthenticationMechanism:       c.mongodb.AuthenticationMechanism,
			AuthenticationDatabase:        c.mongodb.AuthenticationDatabase,
		}
	case models.PostgreSQLServiceType:
		params.PostgreSQLOptions = &models.ChangePostgreSQLOptions{
			SSLCa:   c.postgresql.SSLCa,
			SSLCert: c.postgresql.SSLCert,
			SSLKey:  c.postgresql.SSLKey,
		}
	case models.ValkeyServiceType:
		params.ValkeyOptions = &models.ChangeValkeyOptions{
			SSLCa:   c.valkey.SSLCa,
			SSLCert: c.valkey.SSLCert,
			SSLKey:  c.valkey.SSLKey,
		}
	default:
		// ProxySQL, HAProxy and External Agents have no options of their own type.
	}

	if isQAN {
		qan := c.qan
		if agent.AgentType != models.QANMySQLSlowlogAgentType {
			qan.MaxQueryLogSize = nil
		}
		params.QANOptions = &qan
	}

	if !isExporter {
		return params
	}

	exporter := c.exporter
	params.ExporterOptions = &exporter
	params.AgentPassword = c.agentPassword
	params.EnvironmentVariableNames = c.envVarNames
	params.ListenPort = c.listenPort

	switch c.serviceType {
	case models.MySQLServiceType:
		params.MySQLOptions.TableCountTablestatsGroupLimit = c.mysql.TableCountTablestatsGroupLimit
	case models.MongoDBServiceType:
		params.MongoDBOptions.StatsCollections = c.mongodb.StatsCollections
		params.MongoDBOptions.CollectionsLimit = c.mongodb.CollectionsLimit
		params.MongoDBOptions.EnableAllCollectors = c.mongodb.EnableAllCollectors
		params.MongoDBOptions.EnableDiagnosticDataHistograms = c.mongodb.EnableDiagnosticDataHistograms
	case models.PostgreSQLServiceType:
		params.PostgreSQLOptions.AutoDiscoveryLimit = c.postgresql.AutoDiscoveryLimit
		params.PostgreSQLOptions.MaxExporterConnections = c.postgresql.MaxExporterConnections
	case models.HAProxyServiceType, models.ExternalServiceType:
		// The exporter carries the Service labels, as when the Service is added.
		params.CustomLabels = c.service.CustomLabels
	default:
		// ProxySQL and Valkey exporters have no exporter-only options.
	}

	return params
}

// minLogLevel returns the lowest log level the Agent supports; a lower requested level is raised to it.
func minLogLevel(agentType models.AgentType) inventoryv1.LogLevel {
	switch agentType {
	case models.MySQLdExporterType, models.PostgresExporterType, models.ValkeyExporterType:
		return inventoryv1.LogLevel_LOG_LEVEL_ERROR
	default:
		return inventoryv1.LogLevel_LOG_LEVEL_FATAL
	}
}

// pushMetrics resolves the requested metrics mode of the exporter the way adding the Service does.
func pushMetrics(q *reform.Querier, exporter *models.Agent, mode managementv1.MetricsMode) (bool, error) {
	pmmAgentID := pointer.GetString(exporter.PMMAgentID)

	// An external exporter scraped by the Server has no pmm-agent; the one on its Node pushes for it.
	if exporter.AgentType == models.ExternalExporterType && pmmAgentID == "" {
		agents, err := models.FindPMMAgentsRunningOnNode(q, pointer.GetString(exporter.RunsOnNodeID))
		if err != nil {
			return false, err
		}

		if len(agents) != 1 {
			if mode == managementv1.MetricsMode_METRICS_MODE_PUSH {
				return false, status.Error(codes.FailedPrecondition, "Push metrics mode requires exactly one pmm-agent on the Node.")
			}

			return false, nil
		}

		pmmAgentID = agents[0].AgentID
	}

	mode, err := supportedMetricsMode(mode, pmmAgentID)
	if err != nil {
		return false, err
	}

	return isPushMode(mode), nil
}
