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

package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/version"
)

const (
	pushMetricsTrue  = "((exporter_options ? 'push_metrics') AND (exporter_options->>'push_metrics')::boolean = true)"
	pushMetricsFalse = "(NOT (exporter_options ? 'push_metrics') OR (exporter_options->>'push_metrics')::boolean = false)"
)

// MySQLOptionsParams contains methods to create MySQLOptions object.
type MySQLOptionsParams interface {
	GetTlsCa() string
	GetTlsCert() string
	GetTlsKey() string
}

// MySQLOptionsFromRequest creates MySQLOptions object from request.
func MySQLOptionsFromRequest(params MySQLOptionsParams) *MySQLOptions {
	if params.GetTlsCa() != "" || params.GetTlsCert() != "" || params.GetTlsKey() != "" {
		return &MySQLOptions{
			TLSCa:   params.GetTlsCa(),
			TLSCert: params.GetTlsCert(),
			TLSKey:  params.GetTlsKey(),
		}
	}
	return nil
}

// PostgreSQLOptionsParams contains methods to create PostgreSQLOptions object.
type PostgreSQLOptionsParams interface {
	GetTlsCa() string
	GetTlsCert() string
	GetTlsKey() string
}

// PostgreSQLExtendedOptionsParams contains extended parameters for PostgreSQL exporter.
type PostgreSQLExtendedOptionsParams interface {
	GetAutoDiscoveryLimit() *int32
	GetMaxExporterConnections() int32
}

// PostgreSQLOptionsFromRequest creates PostgreSQLOptions object from request.
func PostgreSQLOptionsFromRequest(params PostgreSQLOptionsParams) *PostgreSQLOptions {
	res := &PostgreSQLOptions{}
	if params.GetTlsCa() != "" || params.GetTlsCert() != "" || params.GetTlsKey() != "" {
		res.SSLCa = params.GetTlsCa()
		res.SSLCert = params.GetTlsCert()
		res.SSLKey = params.GetTlsKey()
	}

	// PostgreSQL exporter has these parameters but they are not needed for QAN agent.
	if extendedOptions, ok := params.(PostgreSQLExtendedOptionsParams); ok && extendedOptions != nil {
		res.AutoDiscoveryLimit = extendedOptions.GetAutoDiscoveryLimit()
		res.MaxExporterConnections = extendedOptions.GetMaxExporterConnections()
	}

	return res
}

// MongoDBOptionsParams contains methods to create MongoDBOptions object.
type MongoDBOptionsParams interface {
	GetTlsCertificateKey() string
	GetTlsCertificateKeyFilePassword() string
	GetTlsCa() string
	GetAuthenticationMechanism() string
	GetAuthenticationDatabase() string
}

// MongoDBExtendedOptionsParams contains extended parameters for MongoDB exporter.
type MongoDBExtendedOptionsParams interface {
	GetStatsCollections() []string
	GetCollectionsLimit() int32
	GetEnableAllCollectors() bool
}

// MongoDBOptionsFromRequest creates MongoDBOptionsParams object from request.
func MongoDBOptionsFromRequest(params MongoDBOptionsParams) *MongoDBOptions {
	var mdbOptions *MongoDBOptions

	if params.GetTlsCertificateKey() != "" || params.GetTlsCertificateKeyFilePassword() != "" || params.GetTlsCa() != "" {
		mdbOptions = &MongoDBOptions{}
		mdbOptions.TLSCertificateKey = params.GetTlsCertificateKey()
		mdbOptions.TLSCertificateKeyFilePassword = params.GetTlsCertificateKeyFilePassword()
		mdbOptions.TLSCa = params.GetTlsCa()
	}

	if params.GetAuthenticationMechanism() != "" || params.GetAuthenticationDatabase() != "" {
		if mdbOptions == nil {
			mdbOptions = &MongoDBOptions{}
		}
		mdbOptions.AuthenticationMechanism = params.GetAuthenticationMechanism()
		mdbOptions.AuthenticationDatabase = params.GetAuthenticationDatabase()
	}

	// MongoDB exporter has these parameters but they are not needed for QAN agent.
	if extendedOptions, ok := params.(MongoDBExtendedOptionsParams); ok {
		if extendedOptions != nil {
			if mdbOptions == nil {
				mdbOptions = &MongoDBOptions{}
			}
			mdbOptions.StatsCollections = extendedOptions.GetStatsCollections()
			mdbOptions.CollectionsLimit = extendedOptions.GetCollectionsLimit()
			mdbOptions.EnableAllCollectors = extendedOptions.GetEnableAllCollectors()
		}
	}

	return mdbOptions
}

// AzureOptionsParams contains methods to create AzureOptions object.
type AzureOptionsParams interface {
	GetAzureSubscriptionId() string
	GetAzureClientId() string
	GetAzureClientSecret() string
	GetAzureTenantId() string
	GetAzureResourceGroup() string
}

// AzureOptionsFromRequest creates AzureOptions object from request.
func AzureOptionsFromRequest(params AzureOptionsParams) *AzureOptions {
	if params.GetAzureSubscriptionId() != "" || params.GetAzureClientId() != "" || params.GetAzureClientSecret() != "" ||
		params.GetAzureTenantId() != "" || params.GetAzureResourceGroup() != "" {
		return &AzureOptions{
			SubscriptionID: params.GetAzureSubscriptionId(),
			ClientID:       params.GetAzureClientId(),
			ClientSecret:   params.GetAzureClientSecret(),
			TenantID:       params.GetAzureTenantId(),
			ResourceGroup:  params.GetAzureResourceGroup(),
		}
	}
	return nil
}

func checkUniqueAgentID(q *reform.Querier, id string) error {
	if id == "" {
		panic("empty Agent ID")
	}

	agent := &Agent{AgentID: id}
	err := q.Reload(agent)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return errors.WithStack(err)
	}

	return status.Errorf(codes.AlreadyExists, "Agent with ID %s already exists.", id)
}

// AgentFilters represents filters for agents list.
type AgentFilters struct {
	// Return only Agents started by this pmm-agent.
	PMMAgentID string
	// Return only Agents that provide insights for that Node.
	NodeID string
	// Return only Agents that provide insights for that Service.
	ServiceID string
	// Return Agents with provided type.
	AgentType *AgentType
	// Return only Agents that provide insights for that AWSAccessKey.
	AWSAccessKey string
}

// FindAgents returns Agents by filters.
func FindAgents(q *reform.Querier, filters AgentFilters) ([]*Agent, error) {
	var conditions []string
	var args []interface{}
	idx := 1
	if filters.PMMAgentID != "" {
		if _, err := FindAgentByID(q, filters.PMMAgentID); err != nil {
			return nil, err
		}
		conditions = append(conditions, fmt.Sprintf("pmm_agent_id = %s", q.Placeholder(idx)))
		args = append(args, filters.PMMAgentID)
		idx++
	}
	if filters.NodeID != "" {
		if _, err := FindNodeByID(q, filters.NodeID); err != nil {
			return nil, err
		}
		conditions = append(conditions, fmt.Sprintf("node_id = %s", q.Placeholder(idx)))
		args = append(args, filters.NodeID)
		idx++
	}
	if filters.ServiceID != "" {
		if _, err := FindServiceByID(q, filters.ServiceID); err != nil {
			return nil, err
		}
		conditions = append(conditions, fmt.Sprintf("service_id = %s", q.Placeholder(idx)))
		args = append(args, filters.ServiceID)
		idx++
	}
	if filters.AgentType != nil {
		conditions = append(conditions, fmt.Sprintf("agent_type = %s", q.Placeholder(idx)))
		args = append(args, *filters.AgentType)
		idx++
	}
	if filters.AWSAccessKey != "" {
		conditions = append(conditions, fmt.Sprintf("(aws_options ? 'aws_access_key' AND aws_options->>'aws_access_key' = %s)", q.Placeholder(idx)))
		args = append(args, filters.AWSAccessKey)
	}

	var whereClause string
	if len(conditions) != 0 {
		whereClause = fmt.Sprintf("WHERE %s", strings.Join(conditions, " AND "))
	}
	structs, err := q.SelectAllFrom(AgentTable, fmt.Sprintf("%s ORDER BY agent_id", whereClause), args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	agents := make([]*Agent, len(structs))
	for i, s := range structs {
		decryptedAgent := DecryptAgent(*s.(*Agent)) //nolint:forcetypeassert
		agents[i] = &decryptedAgent
	}

	return agents, nil
}

// FindAgentByID finds Agent by ID.
func FindAgentByID(q *reform.Querier, id string) (*Agent, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Agent ID.")
	}

	agent := &Agent{AgentID: id}
	err := q.Reload(agent)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Agent with ID %s not found.", id)
		}
		return nil, errors.WithStack(err)
	}
	decryptedAgent := DecryptAgent(*agent)

	return &decryptedAgent, nil
}

// FindAgentsByIDs finds Agents by IDs.
func FindAgentsByIDs(q *reform.Querier, ids []string) ([]*Agent, error) {
	if len(ids) == 0 {
		return []*Agent{}, nil
	}

	p := strings.Join(q.Placeholders(1, len(ids)), ", ")
	tail := fmt.Sprintf("WHERE agent_id IN (%s) ORDER BY agent_id", p)
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	structs, err := q.SelectAllFrom(AgentTable, tail, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]*Agent, len(structs))
	for i, s := range structs {
		decryptedAgent := DecryptAgent(*s.(*Agent)) //nolint:forcetypeassert
		res[i] = &decryptedAgent
	}
	return res, nil
}

// FindDBConfigForService find DB config from agents running on service specified by serviceID.
func FindDBConfigForService(q *reform.Querier, serviceID string) (*DBConfig, error) {
	svc, err := FindServiceByID(q, serviceID)
	if err != nil {
		return nil, err
	}
	var agentTypes []AgentType
	switch svc.ServiceType {
	case MySQLServiceType:
		agentTypes = []AgentType{
			MySQLdExporterType,
			QANMySQLSlowlogAgentType,
			QANMySQLPerfSchemaAgentType,
		}
	case PostgreSQLServiceType:
		agentTypes = []AgentType{
			PostgresExporterType,
			QANPostgreSQLPgStatementsAgentType,
			QANPostgreSQLPgStatMonitorAgentType,
		}
	case MongoDBServiceType:
		agentTypes = []AgentType{
			MongoDBExporterType,
			QANMongoDBProfilerAgentType,
		}
	case ExternalServiceType, HAProxyServiceType, ProxySQLServiceType:
		fallthrough
	default:
		return nil, status.Error(codes.FailedPrecondition, "Unsupported service.")
	}
	p := strings.Join(q.Placeholders(2, len(agentTypes)), ", ")
	tail := fmt.Sprintf("WHERE service_id = $1 AND agent_type IN (%s) ORDER BY agent_id", p)

	args := make([]interface{}, len(agentTypes)+1)
	args[0] = serviceID
	for i, agentType := range agentTypes {
		args[i+1] = agentType
	}

	structs, err := q.SelectAllFrom(AgentTable, tail, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]*Agent, len(structs))
	for i, s := range structs {
		decryptedAgent := DecryptAgent(*s.(*Agent)) //nolint:forcetypeassert
		res[i] = &decryptedAgent
	}

	if len(res) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "No agents available.")
	}

	// Find config with specified user.
	for _, agent := range res {
		cfg := agent.DBConfig(svc)
		if cfg.Valid() {
			return cfg, nil
		}
	}

	return nil, status.Error(codes.FailedPrecondition, "No DB config found.")
}

// FindPMMAgentsRunningOnNode gets pmm-agents for node where it runs.
func FindPMMAgentsRunningOnNode(q *reform.Querier, nodeID string) ([]*Agent, error) {
	structs, err := q.SelectAllFrom(AgentTable, "WHERE runs_on_node_id = $1 AND agent_type = $2", nodeID, PMMAgentType)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Couldn't get agents by runs_on_node_id, %s", nodeID)
	}

	res := make([]*Agent, 0, len(structs))
	for _, str := range structs {
		decryptedAgent := DecryptAgent(*str.(*Agent)) //nolint:forcetypeassert
		res = append(res, &decryptedAgent)
	}

	return res, nil
}

// FindPMMAgentsForService gets pmm-agents for service.
func FindPMMAgentsForService(q *reform.Querier, serviceID string) ([]*Agent, error) {
	_, err := q.SelectOneFrom(ServiceTable, "WHERE service_id = $1", serviceID)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Couldn't get services by service_id, %s", serviceID)
	}

	// First, find agents with serviceID.
	allAgents, err := q.SelectAllFrom(AgentTable, "WHERE service_id = $1", serviceID)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Couldn't get all agents for service %s", serviceID)
	}
	pmmAgentIDs := make([]interface{}, len(allAgents))
	for _, str := range allAgents {
		row := str.(*Agent) //nolint:forcetypeassert
		if row.PMMAgentID != nil {
			for _, a := range pmmAgentIDs {
				if a == *row.PMMAgentID {
					break
				}
				pmmAgentIDs = append(pmmAgentIDs, *row.PMMAgentID) //nolint:makezero
			}
		}
	}

	if len(pmmAgentIDs) == 0 {
		return []*Agent{}, nil
	}

	// Last, find all pmm-agents.
	ph := strings.Join(q.Placeholders(1, len(pmmAgentIDs)), ", ")
	atail := fmt.Sprintf("WHERE agent_id IN (%s) AND agent_type = '%s' ORDER BY agent_id", ph, PMMAgentType)
	pmmAgentRecords, err := q.SelectAllFrom(AgentTable, atail, pmmAgentIDs...)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Couldn't get pmm-agents for service %s", serviceID)
	}
	res := make([]*Agent, 0, len(pmmAgentRecords))
	for _, str := range pmmAgentRecords {
		decryptedAgent := DecryptAgent(*str.(*Agent)) //nolint:forcetypeassert
		res = append(res, &decryptedAgent)
	}

	return res, nil
}

// FindPMMAgentsForServicesOnNode gets pmm-agents for Services running on Node.
func FindPMMAgentsForServicesOnNode(q *reform.Querier, nodeID string) ([]*Agent, error) {
	structs, err := q.FindAllFrom(ServiceTable, "node_id", nodeID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Service IDs")
	}

	allAgents := make([]*Agent, 0, len(structs))
	for _, str := range structs {
		serviceID := str.(*Service).ServiceID //nolint:forcetypeassert
		agents, err := FindPMMAgentsForService(q, serviceID)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		allAgents = append(allAgents, agents...)
	}

	return allAgents, nil
}

// FindPMMAgentsForVersion selects pmm-agents with version >= minPMMAgentVersion.
func FindPMMAgentsForVersion(logger *logrus.Entry, agents []*Agent, minPMMAgentVersion *version.Parsed) []*Agent {
	if len(agents) == 0 {
		return nil
	}

	if minPMMAgentVersion == nil {
		return agents
	}
	result := make([]*Agent, 0, len(agents))

	for _, a := range agents {
		v, err := version.Parse(pointer.GetString(a.Version))
		if err != nil {
			logger.Warnf("Failed to parse pmm-agent version: %s.", err)
			continue
		}

		if v.Less(minPMMAgentVersion) {
			continue
		}

		result = append(result, a)
	}

	return result
}

// FindAgentsForScrapeConfig returns Agents for scrape config generation by pmm_agent_id and push_metrics value.
func FindAgentsForScrapeConfig(q *reform.Querier, pmmAgentID *string, pushMetrics bool) ([]*Agent, error) {
	var (
		args       []interface{}
		conditions []string
	)
	if pmmAgentID != nil {
		conditions = append(conditions, fmt.Sprintf("pmm_agent_id = %s", q.Placeholder(1)))
		args = append(args, pointer.GetString(pmmAgentID))
	}

	if pushMetrics {
		conditions = append(conditions, pushMetricsTrue)
	} else {
		conditions = append(conditions, pushMetricsFalse)
	}

	conditions = append(conditions, "NOT disabled", "listen_port IS NOT NULL")
	whereClause := fmt.Sprintf("WHERE %s ORDER BY agent_type, agent_id ", strings.Join(conditions, " AND "))
	allAgents, err := q.SelectAllFrom(AgentTable, whereClause, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]*Agent, len(allAgents))
	for i, s := range allAgents {
		decryptedAgent := DecryptAgent(*s.(*Agent)) //nolint:forcetypeassert
		res[i] = &decryptedAgent
	}
	return res, nil
}

// FindPMMAgentsIDsWithPushMetrics returns pmm-agents-ids with agent, that use push_metrics mode.
func FindPMMAgentsIDsWithPushMetrics(q *reform.Querier) ([]string, error) {
	structs, err := q.SelectAllFrom(AgentTable, fmt.Sprintf("WHERE NOT disabled AND pmm_agent_id IS NOT NULL AND %s ORDER BY agent_id", pushMetricsTrue))
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "Couldn't get agents")
	}

	uniqAgents := make(map[string]struct{})
	res := make([]string, 0, len(structs))
	for _, str := range structs {
		row := pointer.GetString(str.(*Agent).PMMAgentID) //nolint:forcetypeassert
		if _, ok := uniqAgents[row]; ok {
			continue
		}
		res = append(res, row)
		uniqAgents[row] = struct{}{}
	}

	return res, nil
}

// FindPmmAgentIDToRunActionOrJob finds pmm-agent-id to run action.
func FindPmmAgentIDToRunActionOrJob(pmmAgentID string, agents []*Agent) (string, error) {
	// no explicit ID is given, and there is only one
	if pmmAgentID == "" && len(agents) == 1 {
		return agents[0].AgentID, nil
	}

	// no explicit ID is given, and there are zero or several
	if pmmAgentID == "" {
		return "", status.Errorf(codes.InvalidArgument, "Couldn't find pmm-agent-id to run action")
	}

	// check that explicit agent id is correct
	for _, a := range agents {
		if a.AgentID == pmmAgentID {
			return a.AgentID, nil
		}
	}
	return "", status.Errorf(codes.FailedPrecondition, "Couldn't find pmm-agent-id to run action")
}

// ExtractPmmAgentVersionFromAgent extract PMM agent version from Agent by pmm-agent-id.
func ExtractPmmAgentVersionFromAgent(q *reform.Querier, agent *Agent) *version.Parsed {
	pmmAgentID, err := ExtractPmmAgentID(agent)
	if err != nil {
		return nil
	}
	pmmAgent, err := FindAgentByID(q, pmmAgentID)
	if err != nil {
		return nil
	}
	version, err := version.Parse(pointer.GetString(pmmAgent.Version))
	if err != nil {
		return nil
	}

	return version
}

// ExtractPmmAgentID extract pmm-agent-id from Agent by type.
func ExtractPmmAgentID(agent *Agent) (string, error) {
	switch agent.AgentType {
	case PMMAgentType:
		return agent.AgentID, nil
	case NodeExporterType,
		MySQLdExporterType,
		MongoDBExporterType,
		PostgresExporterType,
		ProxySQLExporterType,
		RDSExporterType,
		AzureDatabaseExporterType,
		QANMySQLPerfSchemaAgentType,
		QANMySQLSlowlogAgentType,
		QANMongoDBProfilerAgentType,
		QANPostgreSQLPgStatementsAgentType,
		QANPostgreSQLPgStatMonitorAgentType,
		ExternalExporterType,
		VMAgentType:
		return pointer.GetString(agent.PMMAgentID), nil
	default:
		return "", status.Errorf(codes.Internal, "Unhandled inventory Agent type %s", agent.AgentType)
	}
}

// createPMMAgentWithID creates PMMAgent with given ID.
func createPMMAgentWithID(q *reform.Querier, id, runsOnNodeID string, customLabels map[string]string) (*Agent, error) {
	if err := checkUniqueAgentID(q, id); err != nil {
		return nil, err
	}

	if _, err := FindNodeByID(q, runsOnNodeID); err != nil {
		return nil, err
	}

	// TODO https://jira.percona.com/browse/PMM-4496
	// Check that Node is not remote.

	agent := &Agent{
		AgentID:      id,
		AgentType:    PMMAgentType,
		RunsOnNodeID: &runsOnNodeID,
	}
	if err := agent.SetCustomLabels(customLabels); err != nil {
		return nil, err
	}

	if err := q.Insert(agent); err != nil {
		return nil, errors.WithStack(err)
	}

	return agent, nil
}

// CreatePMMAgent creates PMMAgent.
func CreatePMMAgent(q *reform.Querier, runsOnNodeID string, customLabels map[string]string) (*Agent, error) {
	id := uuid.New().String()
	return createPMMAgentWithID(q, id, runsOnNodeID, customLabels)
}

// CreateNodeExporter creates NodeExporter.
func CreateNodeExporter(q *reform.Querier,
	pmmAgentID string,
	customLabels map[string]string,
	pushMetrics bool,
	exposeExporter bool,
	disableCollectors []string,
	agentPassword *string,
	logLevel string,
) (*Agent, error) {
	// TODO merge into CreateAgent

	id := uuid.New().String()
	if err := checkUniqueAgentID(q, id); err != nil {
		return nil, err
	}

	pmmAgent, err := FindAgentByID(q, pmmAgentID)
	if err != nil {
		return nil, err
	}
	if !IsPushMetricsSupported(pmmAgent.Version) {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot use push_metrics_enabled with pmm_agent version=%q,"+
			" it doesn't support it, minimum supported version=%q", pointer.GetString(pmmAgent.Version), PMMAgentWithPushMetricsSupport.String())
	}
	row := &Agent{
		AgentID:       id,
		AgentType:     NodeExporterType,
		PMMAgentID:    &pmmAgentID,
		NodeID:        pmmAgent.RunsOnNodeID,
		AgentPassword: agentPassword,
		ExporterOptions: &ExporterOptions{
			ExposeExporter:     exposeExporter,
			PushMetrics:        pushMetrics,
			DisabledCollectors: disableCollectors,
		},
		LogLevel: pointer.ToStringOrNil(logLevel),
	}
	if err := row.SetCustomLabels(customLabels); err != nil {
		return nil, err
	}

	encryptedAgent := EncryptAgent(*row)
	if err := q.Insert(&encryptedAgent); err != nil {
		return nil, errors.WithStack(err)
	}
	agent := DecryptAgent(encryptedAgent)

	return &agent, nil
}

// CreateExternalExporterParams params for add external exporter.
type CreateExternalExporterParams struct {
	RunsOnNodeID string
	ServiceID    string
	Username     string
	Password     string
	Scheme       string
	MetricsPath  string
	ListenPort   uint32
	CustomLabels map[string]string
	PushMetrics  bool
}

// CreateExternalExporter creates ExternalExporter.
func CreateExternalExporter(q *reform.Querier, params *CreateExternalExporterParams) (*Agent, error) {
	if !(params.ListenPort > 0 && params.ListenPort < 65536) {
		return nil, status.Errorf(codes.InvalidArgument, "Listen port should be between 1 and 65535.")
	}
	var pmmAgentID *string
	runsOnNodeID := pointer.ToString(params.RunsOnNodeID)
	id := uuid.New().String()
	if err := checkUniqueAgentID(q, id); err != nil {
		return nil, err
	}
	// with push metrics we have to detect pmm_agent_id for external exporter.
	if params.PushMetrics {
		agentIDs, err := FindPMMAgentsRunningOnNode(q, params.RunsOnNodeID)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot find pmm_agent for external exporter with push_metrics")
		}
		switch len(agentIDs) {
		case 0:
			return nil, status.Errorf(codes.NotFound, "cannot find any pmm-agent by NodeID")
		case 1:
		default:
			return nil, errors.Errorf("exactly one pmm_agent expected for external exporter, but "+
				"(%d) found at node: %s", len(agentIDs), params.RunsOnNodeID)
		}
		if !IsPushMetricsSupported(agentIDs[0].Version) {
			return nil, status.Errorf(codes.FailedPrecondition, "cannot use push_metrics_enabled with pmm_agent version=%q,"+
				" it doesn't support it, minimum supported version=%q", pointer.GetString(agentIDs[0].Version), PMMAgentWithPushMetricsSupport.String())
		}
		pmmAgentID = pointer.ToString(agentIDs[0].AgentID)
		runsOnNodeID = nil
	}

	if _, err := FindNodeByID(q, params.RunsOnNodeID); err != nil {
		return nil, err
	}
	if _, err := FindServiceByID(q, params.ServiceID); err != nil {
		return nil, err
	}

	scheme := params.Scheme
	if scheme == "" {
		scheme = "http"
	}
	metricsPath := params.MetricsPath
	if metricsPath == "" {
		metricsPath = "/metrics"
	}
	row := &Agent{
		PMMAgentID:   pmmAgentID,
		AgentID:      id,
		AgentType:    ExternalExporterType,
		RunsOnNodeID: runsOnNodeID,
		ServiceID:    pointer.ToStringOrNil(params.ServiceID),
		Username:     pointer.ToStringOrNil(params.Username),
		Password:     pointer.ToStringOrNil(params.Password),
		ListenPort:   pointer.ToUint16(uint16(params.ListenPort)),
		ExporterOptions: &ExporterOptions{
			PushMetrics:   params.PushMetrics,
			MetricsPath:   pointer.ToStringOrNil(metricsPath),
			MetricsScheme: pointer.ToStringOrNil(scheme),
		},
	}
	if err := row.SetCustomLabels(params.CustomLabels); err != nil {
		return nil, err
	}

	encryptedAgent := EncryptAgent(*row)
	if err := q.Insert(&encryptedAgent); err != nil {
		return nil, errors.WithStack(err)
	}
	agent := DecryptAgent(encryptedAgent)

	return &agent, nil
}

// CreateAgentParams params for add common exporter.
type CreateAgentParams struct {
	PMMAgentID        string
	NodeID            string
	ServiceID         string
	Username          string
	Password          string
	AgentPassword     string
	CustomLabels      map[string]string
	TLS               bool
	TLSSkipVerify     bool
	LogLevel          string
	ExporterOptions   *ExporterOptions
	QANOptions        *QANOptions
	AWSOptions        *AWSOptions
	AzureOptions      *AzureOptions
	MongoDBOptions    *MongoDBOptions
	MySQLOptions      *MySQLOptions
	PostgreSQLOptions *PostgreSQLOptions
}

func compatibleNodeAndAgent(nodeType NodeType, agentType AgentType) bool {
	const allowAll = "allow_all"
	allow := map[NodeType]AgentType{
		GenericNodeType:             allowAll,
		ContainerNodeType:           allowAll,
		RemoteNodeType:              ExternalExporterType,
		RemoteRDSNodeType:           RDSExporterType,
		RemoteAzureDatabaseNodeType: AzureDatabaseExporterType,
	}

	allowed, ok := allow[nodeType]
	if !ok {
		return false
	}

	if allowed == allowAll {
		return true
	}

	return allowed == agentType
}

func compatibleServiceAndAgent(serviceType ServiceType, agentType AgentType) bool {
	allow := map[AgentType][]ServiceType{
		MySQLdExporterType: {
			MySQLServiceType,
		},
		QANMySQLSlowlogAgentType: {
			MySQLServiceType,
		},
		QANMySQLPerfSchemaAgentType: {
			MySQLServiceType,
		},
		MongoDBExporterType: {
			MongoDBServiceType,
		},
		QANMongoDBProfilerAgentType: {
			MongoDBServiceType,
		},
		PostgresExporterType: {
			PostgreSQLServiceType,
		},
		ProxySQLExporterType: {
			ProxySQLServiceType,
		},
		AzureDatabaseExporterType: {
			PostgreSQLServiceType,
			MySQLServiceType,
		},
		RDSExporterType: {
			PostgreSQLServiceType,
			MySQLServiceType,
		},
		QANPostgreSQLPgStatMonitorAgentType: {
			PostgreSQLServiceType,
		},
		QANPostgreSQLPgStatementsAgentType: {
			PostgreSQLServiceType,
		},
		ExternalExporterType: {
			ExternalServiceType,
		},
	}

	allowed, ok := allow[agentType]
	if !ok {
		return false
	}

	for _, svcType := range allowed {
		if svcType == serviceType {
			return true
		}
	}

	return false
}

func prepareOptionsParams(params *CreateAgentParams) *CreateAgentParams {
	if params.ExporterOptions == nil {
		params.ExporterOptions = &ExporterOptions{}
	}
	if params.QANOptions == nil {
		params.QANOptions = &QANOptions{}
	}
	if params.AWSOptions == nil {
		params.AWSOptions = &AWSOptions{}
	}
	if params.AzureOptions == nil {
		params.AzureOptions = &AzureOptions{}
	}
	if params.MongoDBOptions == nil {
		params.MongoDBOptions = &MongoDBOptions{}
	}
	if params.MySQLOptions == nil {
		params.MySQLOptions = &MySQLOptions{}
	}
	if params.PostgreSQLOptions == nil {
		params.PostgreSQLOptions = &PostgreSQLOptions{}
	}

	return params
}

// CreateAgent creates Agent with given type.
func CreateAgent(q *reform.Querier, agentType AgentType, params *CreateAgentParams) (*Agent, error) { //nolint:unparam
	id := uuid.New().String()
	if err := checkUniqueAgentID(q, id); err != nil {
		return nil, err
	}

	params = prepareOptionsParams(params)

	pmmAgent, err := FindAgentByID(q, params.PMMAgentID)
	if err != nil {
		return nil, err
	}
	// check version for agent, if it exists.
	if params.ExporterOptions.PushMetrics {
		// special case for vmAgent, it always supports push metrics.
		if agentType != VMAgentType && !IsPushMetricsSupported(pmmAgent.Version) {
			return nil, status.Errorf(codes.FailedPrecondition, "cannot use push_metrics_enabled with pmm_agent version=%q,"+
				" it doesn't support it, minimum supported version=%q", pointer.GetString(pmmAgent.Version), PMMAgentWithPushMetricsSupport.String())
		}
	}

	if params.NodeID != "" {
		node, err := FindNodeByID(q, params.NodeID)
		if err != nil {
			return nil, err
		}

		if !compatibleNodeAndAgent(node.NodeType, agentType) {
			return nil, status.Errorf(codes.FailedPrecondition, "invalid combination of node type %s and agent type %s", node.NodeType, agentType)
		}
	}

	if params.ServiceID != "" {
		svc, err := FindServiceByID(q, params.ServiceID)
		if err != nil {
			return nil, err
		}

		if !compatibleServiceAndAgent(svc.ServiceType, agentType) {
			return nil, status.Errorf(codes.FailedPrecondition, "invalid combination of service type %s and agent type %s", svc.ServiceType, agentType)
		}
	}

	row := &Agent{
		AgentID:           id,
		AgentType:         agentType,
		PMMAgentID:        &params.PMMAgentID,
		ServiceID:         pointer.ToStringOrNil(params.ServiceID),
		NodeID:            pointer.ToStringOrNil(params.NodeID),
		Username:          pointer.ToStringOrNil(params.Username),
		Password:          pointer.ToStringOrNil(params.Password),
		AgentPassword:     pointer.ToStringOrNil(params.AgentPassword),
		TLS:               params.TLS,
		TLSSkipVerify:     params.TLSSkipVerify,
		ExporterOptions:   params.ExporterOptions,
		QANOptions:        params.QANOptions,
		AWSOptions:        params.AWSOptions,
		AzureOptions:      params.AzureOptions,
		MongoDBOptions:    params.MongoDBOptions,
		MySQLOptions:      params.MySQLOptions,
		PostgreSQLOptions: params.PostgreSQLOptions,
		LogLevel:          pointer.ToStringOrNil(params.LogLevel),
	}
	if err := row.SetCustomLabels(params.CustomLabels); err != nil {
		return nil, err
	}

	encryptedAgent := EncryptAgent(*row)
	if err := q.Insert(&encryptedAgent); err != nil {
		return nil, errors.WithStack(err)
	}
	agent := DecryptAgent(encryptedAgent)

	return &agent, nil
}

// ChangeCommonAgentParams contains parameters that can be changed for all Agents.
type ChangeCommonAgentParams struct {
	Enabled            *bool              // true - enable, false - disable, nil - no change
	CustomLabels       *map[string]string // empty map - remove all custom labels, non-empty - change, nil - no change
	EnablePushMetrics  *bool
	MetricsResolutions ChangeMetricsResolutionsParams
}

// ChangeMetricsResolutionsParams contains metrics resolutions for change.
type ChangeMetricsResolutionsParams struct {
	HR *time.Duration
	MR *time.Duration
	LR *time.Duration
}

// ChangeAgent changes common parameters for given Agent.
func ChangeAgent(q *reform.Querier, agentID string, params *ChangeCommonAgentParams) (*Agent, error) {
	row, err := FindAgentByID(q, agentID)
	if err != nil {
		return nil, err
	}

	if params.Enabled != nil {
		row.Disabled = !(*params.Enabled)
	}

	if params.EnablePushMetrics != nil {
		row.ExporterOptions = &ExporterOptions{
			PushMetrics: *params.EnablePushMetrics,
		}
		if row.AgentType == ExternalExporterType {
			if err := updateExternalExporterParams(q, row); err != nil {
				return nil, errors.Wrap(err, "failed to update External exporterParams for PushMetrics")
			}
		}
	}

	if params.CustomLabels != nil {
		if len(*params.CustomLabels) == 0 {
			if err = row.SetCustomLabels(nil); err != nil {
				return nil, err
			}
		} else {
			if err = row.SetCustomLabels(*params.CustomLabels); err != nil {
				return nil, err
			}
		}
	}

	if row.ExporterOptions == nil {
		row.ExporterOptions = &ExporterOptions{}
	}
	if row.ExporterOptions.MetricsResolutions == nil {
		row.ExporterOptions.MetricsResolutions = &MetricsResolutions{}
	}
	if params.MetricsResolutions.LR != nil {
		row.ExporterOptions.MetricsResolutions.LR = *params.MetricsResolutions.LR
	}
	if params.MetricsResolutions.MR != nil {
		row.ExporterOptions.MetricsResolutions.MR = *params.MetricsResolutions.MR
	}
	if params.MetricsResolutions.HR != nil {
		row.ExporterOptions.MetricsResolutions.HR = *params.MetricsResolutions.HR
	}

	// If all resolutions are empty, then drop whole MetricsResolution field.
	if row.ExporterOptions.MetricsResolutions.HR == 0 && row.ExporterOptions.MetricsResolutions.MR == 0 && row.ExporterOptions.MetricsResolutions.LR == 0 {
		row.ExporterOptions.MetricsResolutions = nil
	}

	if err = q.Update(row); err != nil {
		return nil, errors.WithStack(err)
	}

	return row, nil
}

// RemoveAgent removes Agent by ID.
func RemoveAgent(q *reform.Querier, id string, mode RemoveMode) (*Agent, error) { //nolint:unparam
	a, err := FindAgentByID(q, id)
	if err != nil {
		return nil, err
	}

	if id == PMMServerAgentID {
		return nil, status.Error(codes.PermissionDenied, "pmm-agent on PMM Server can't be removed.")
	}

	structs, err := q.SelectAllFrom(AgentTable, "WHERE pmm_agent_id = $1", id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Agents")
	}
	if len(structs) != 0 {
		switch mode {
		case RemoveRestrict:
			return nil, status.Errorf(codes.FailedPrecondition, "pmm-agent with ID %s has agents.", id)
		case RemoveCascade:
			for _, str := range structs {
				agentID := str.(*Agent).AgentID //nolint:forcetypeassert
				if _, err = RemoveAgent(q, agentID, RemoveRestrict); err != nil {
					return nil, err
				}
			}
		default:
			panic(fmt.Errorf("unhandled RemoveMode %v", mode))
		}
	}

	if err = q.Delete(a); err != nil {
		return nil, errors.Wrap(err, "failed to delete Agent")
	}

	return a, nil
}

// updateExternalExporterParams updates RunsOnNodeID and PMMAgentID params
// for external exporter, is needed for push_metrics mode.
func updateExternalExporterParams(q *reform.Querier, row *Agent) error {
	// with push metrics, external exporter must have PMMAgent id without RunsOnNodeID
	if row.ExporterOptions != nil && row.ExporterOptions.PushMetrics && row.PMMAgentID == nil {
		pmmAgent, err := FindPMMAgentsRunningOnNode(q, pointer.GetString(row.RunsOnNodeID))
		if err != nil {
			return err
		}
		switch len(pmmAgent) {
		case 0:
			return status.Errorf(codes.NotFound, "cannot find any pmm-agent by NodeID")
		case 1:
		default:
			return errors.Errorf("exactly one pmm agent expected, but (%d) found", len(pmmAgent))
		}

		row.RunsOnNodeID = nil
		row.PMMAgentID = pointer.ToString(pmmAgent[0].AgentID)
	}
	// without push metrics, external exporter must have RunsOnNodeID without PMMAgentID
	if row.ExporterOptions != nil && !row.ExporterOptions.PushMetrics && row.RunsOnNodeID == nil {
		pmmAgent, err := FindAgentByID(q, pointer.GetString(row.PMMAgentID))
		if err != nil {
			return err
		}
		row.RunsOnNodeID = pmmAgent.RunsOnNodeID
		row.PMMAgentID = nil
	}
	return nil
}

// IsPushMetricsSupported return if PUSH mode is supported for pmm agent version.
func IsPushMetricsSupported(pmmAgentVersion *string) bool {
	if agentVersion, err := version.Parse(pointer.GetString(pmmAgentVersion)); err == nil {
		if agentVersion.Less(PMMAgentWithPushMetricsSupport) {
			return false
		}
	}
	return true
}
