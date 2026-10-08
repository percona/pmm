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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/api/common"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/duration"
)

// serviceChange is a requested Service update in the form shared by all Service types.
// A nil field leaves the stored value unchanged.
type serviceChange struct {
	serviceType  models.ServiceType
	exporterType models.AgentType

	service models.ChangeServiceParams

	// Applied to every Agent of the Service that stores them.
	username      *string
	password      *string
	tls           *bool
	tlsSkipVerify *bool
	logLevel      *inventoryv1.LogLevel
	mysql         models.ChangeMySQLOptions
	mongodb       models.ChangeMongoDBOptions
	postgresql    models.ChangePostgreSQLOptions
	valkey        models.ChangeValkeyOptions

	// Applied to the exporter only. Exporter-only fields of the options above are applied
	// to the exporter only as well: tablestats limit, MongoDB collector settings and
	// PostgreSQL auto-discovery and connection limits.
	agentPassword *string
	exporter      models.ChangeExporterOptions
	metricsMode   *managementv1.MetricsMode
	envVarNames   *[]string
	listenPort    *uint32

	// Applied to the QAN Agents only; MaxQueryLogSize to the slowlog Agent only.
	qan models.ChangeQANOptions
	// Requested presence of QAN Agents by type; types not in the map are left as they are.
	qanAgents map[models.AgentType]bool

	skipConnectionCheck bool
}

// newServiceChange converts the type-specific parameters of the request.
func newServiceChange(req *managementv1.UpdateServiceRequest) (*serviceChange, error) {
	var c *serviceChange
	switch req.Service.(type) {
	case *managementv1.UpdateServiceRequest_Mysql:
		c = mysqlServiceChange(req.GetMysql())
	case *managementv1.UpdateServiceRequest_Mongodb:
		c = mongodbServiceChange(req.GetMongodb())
	case *managementv1.UpdateServiceRequest_Postgresql:
		c = postgresqlServiceChange(req.GetPostgresql())
	case *managementv1.UpdateServiceRequest_Proxysql:
		c = proxysqlServiceChange(req.GetProxysql())
	case *managementv1.UpdateServiceRequest_Valkey:
		c = valkeyServiceChange(req.GetValkey())
	case *managementv1.UpdateServiceRequest_Haproxy:
		c = haproxyServiceChange(req.GetHaproxy())
	case *managementv1.UpdateServiceRequest_External:
		c = externalServiceChange(req.GetExternal())
	default:
		return nil, status.Error(codes.InvalidArgument, "Service parameters are expected.")
	}

	if c.logLevel != nil && *c.logLevel == inventoryv1.LogLevel_LOG_LEVEL_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "Log level must be one of the levels.")
	}

	return c, nil
}

func mysqlServiceChange(p *managementv1.UpdateMySQLServiceParams) *serviceChange {
	c := &serviceChange{
		serviceType:  models.MySQLServiceType,
		exporterType: models.MySQLdExporterType,
		service: models.ChangeServiceParams{
			Environment:    p.Environment,
			Cluster:        p.Cluster,
			ReplicationSet: p.ReplicationSet,
			CustomLabels:   stringMap(p.CustomLabels),
			Address:        p.Address,
			Port:           port(p.Port),
			Socket:         p.Socket,
		},
		username:      p.Username,
		password:      p.Password,
		tls:           p.Tls,
		tlsSkipVerify: p.TlsSkipVerify,
		logLevel:      p.LogLevel,
		mysql: models.ChangeMySQLOptions{
			TLSCa:          p.TlsCa,
			TLSCert:        p.TlsCert,
			TLSKey:         p.TlsKey,
			ExtraDSNParams: stringMap(p.ExtraDsnParams),
		},
		agentPassword: p.AgentPassword,
		exporter: models.ChangeExporterOptions{
			DisabledCollectors: stringArray(p.DisableCollectors),
			ExposeExporter:     p.ExposeExporter,
			ConnectionTimeout:  duration.OptionalFromProto(p.ConnectionTimeout),
		},
		metricsMode: p.MetricsMode,
		qan: models.ChangeQANOptions{
			MaxQueryLength:          p.MaxQueryLength,
			QueryExamplesDisabled:   p.DisableQueryExamples,
			CommentsParsingDisabled: p.DisableCommentsParsing,
		},
		qanAgents: qanAgents(map[models.AgentType]*bool{
			models.QANMySQLPerfSchemaAgentType: p.QanMysqlPerfschema,
			models.QANMySQLSlowlogAgentType:    p.QanMysqlSlowlog,
		}),
		skipConnectionCheck: p.SkipConnectionCheck,
	}

	if p.TablestatsGroupTableLimit != nil {
		c.mysql.TableCountTablestatsGroupLimit = new(storedTablestatsGroupTableLimit(*p.TablestatsGroupTableLimit))
	}
	if p.MaxSlowlogFileSize != nil {
		c.qan.MaxQueryLogSize = new(storedMaxSlowlogFileSize(*p.MaxSlowlogFileSize))
	}

	return c
}

func mongodbServiceChange(p *managementv1.UpdateMongoDBServiceParams) *serviceChange {
	c := &serviceChange{
		serviceType:  models.MongoDBServiceType,
		exporterType: models.MongoDBExporterType,
		service: models.ChangeServiceParams{
			Environment:    p.Environment,
			Cluster:        p.Cluster,
			ReplicationSet: p.ReplicationSet,
			CustomLabels:   stringMap(p.CustomLabels),
			Address:        p.Address,
			Port:           port(p.Port),
			Socket:         p.Socket,
		},
		username:      p.Username,
		password:      p.Password,
		tls:           p.Tls,
		tlsSkipVerify: p.TlsSkipVerify,
		logLevel:      p.LogLevel,
		mongodb: models.ChangeMongoDBOptions{
			TLSCertificateKey:              p.TlsCertificateKey,
			TLSCertificateKeyFilePassword:  p.TlsCertificateKeyFilePassword,
			TLSCa:                          p.TlsCa,
			AuthenticationMechanism:        p.AuthenticationMechanism,
			AuthenticationDatabase:         p.AuthenticationDatabase,
			StatsCollections:               stringArray(p.StatsCollections),
			CollectionsLimit:               p.CollectionsLimit,
			EnableAllCollectors:            p.EnableAllCollectors,
			EnableDiagnosticDataHistograms: p.EnableDiagnosticDataHistograms,
		},
		agentPassword: p.AgentPassword,
		exporter: models.ChangeExporterOptions{
			DisabledCollectors: stringArray(p.DisableCollectors),
			ExposeExporter:     p.ExposeExporter,
			ConnectionTimeout:  duration.OptionalFromProto(p.ConnectionTimeout),
		},
		metricsMode: p.MetricsMode,
		qan: models.ChangeQANOptions{
			MaxQueryLength: p.MaxQueryLength,
		},
		qanAgents: qanAgents(map[models.AgentType]*bool{
			models.QANMongoDBProfilerAgentType: p.QanMongodbProfiler,
			models.QANMongoDBMongologAgentType: p.QanMongodbMongolog,
		}),
		skipConnectionCheck: p.SkipConnectionCheck,
	}

	if p.EnvironmentVariableNames != nil {
		c.envVarNames = new(p.EnvironmentVariableNames.GetValues())
	}

	return c
}

func postgresqlServiceChange(p *managementv1.UpdatePostgreSQLServiceParams) *serviceChange {
	return &serviceChange{
		serviceType:  models.PostgreSQLServiceType,
		exporterType: models.PostgresExporterType,
		service: models.ChangeServiceParams{
			Environment:    p.Environment,
			Cluster:        p.Cluster,
			ReplicationSet: p.ReplicationSet,
			CustomLabels:   stringMap(p.CustomLabels),
			Database:       p.Database,
			Address:        p.Address,
			Port:           port(p.Port),
			Socket:         p.Socket,
		},
		username:      p.Username,
		password:      p.Password,
		tls:           p.Tls,
		tlsSkipVerify: p.TlsSkipVerify,
		logLevel:      p.LogLevel,
		postgresql: models.ChangePostgreSQLOptions{
			SSLCa:                  p.TlsCa,
			SSLCert:                p.TlsCert,
			SSLKey:                 p.TlsKey,
			AutoDiscoveryLimit:     p.AutoDiscoveryLimit,
			MaxExporterConnections: p.MaxExporterConnections,
		},
		agentPassword: p.AgentPassword,
		exporter: models.ChangeExporterOptions{
			DisabledCollectors: stringArray(p.DisableCollectors),
			ExposeExporter:     p.ExposeExporter,
			ConnectionTimeout:  duration.OptionalFromProto(p.ConnectionTimeout),
		},
		metricsMode: p.MetricsMode,
		qan: models.ChangeQANOptions{
			MaxQueryLength:          p.MaxQueryLength,
			QueryExamplesDisabled:   p.DisableQueryExamples,
			CommentsParsingDisabled: p.DisableCommentsParsing,
		},
		qanAgents: qanAgents(map[models.AgentType]*bool{
			models.QANPostgreSQLPgStatementsAgentType:  p.QanPostgresqlPgstatementsAgent,
			models.QANPostgreSQLPgStatMonitorAgentType: p.QanPostgresqlPgstatmonitorAgent,
		}),
		skipConnectionCheck: p.SkipConnectionCheck,
	}
}

func proxysqlServiceChange(p *managementv1.UpdateProxySQLServiceParams) *serviceChange {
	return &serviceChange{
		serviceType:  models.ProxySQLServiceType,
		exporterType: models.ProxySQLExporterType,
		service: models.ChangeServiceParams{
			Environment:    p.Environment,
			Cluster:        p.Cluster,
			ReplicationSet: p.ReplicationSet,
			CustomLabels:   stringMap(p.CustomLabels),
			Address:        p.Address,
			Port:           port(p.Port),
			Socket:         p.Socket,
		},
		username:      p.Username,
		password:      p.Password,
		tls:           p.Tls,
		tlsSkipVerify: p.TlsSkipVerify,
		logLevel:      p.LogLevel,
		agentPassword: p.AgentPassword,
		exporter: models.ChangeExporterOptions{
			DisabledCollectors: stringArray(p.DisableCollectors),
			ExposeExporter:     p.ExposeExporter,
			ConnectionTimeout:  duration.OptionalFromProto(p.ConnectionTimeout),
		},
		metricsMode:         p.MetricsMode,
		skipConnectionCheck: p.SkipConnectionCheck,
	}
}

func valkeyServiceChange(p *managementv1.UpdateValkeyServiceParams) *serviceChange {
	return &serviceChange{
		serviceType:  models.ValkeyServiceType,
		exporterType: models.ValkeyExporterType,
		service: models.ChangeServiceParams{
			Environment:    p.Environment,
			Cluster:        p.Cluster,
			ReplicationSet: p.ReplicationSet,
			CustomLabels:   stringMap(p.CustomLabels),
			Address:        p.Address,
			Port:           port(p.Port),
			Socket:         p.Socket,
		},
		username:      p.Username,
		password:      p.Password,
		tls:           p.Tls,
		tlsSkipVerify: p.TlsSkipVerify,
		logLevel:      p.LogLevel,
		valkey: models.ChangeValkeyOptions{
			SSLCa:   p.TlsCa,
			SSLCert: p.TlsCert,
			SSLKey:  p.TlsKey,
		},
		agentPassword: p.AgentPassword,
		exporter: models.ChangeExporterOptions{
			ExposeExporter:    p.ExposeExporter,
			ConnectionTimeout: duration.OptionalFromProto(p.ConnectionTimeout),
		},
		metricsMode:         p.MetricsMode,
		skipConnectionCheck: p.SkipConnectionCheck,
	}
}

func haproxyServiceChange(p *managementv1.UpdateHAProxyServiceParams) *serviceChange {
	return &serviceChange{
		serviceType:  models.HAProxyServiceType,
		exporterType: models.ExternalExporterType,
		service: models.ChangeServiceParams{
			Environment:    p.Environment,
			Cluster:        p.Cluster,
			ReplicationSet: p.ReplicationSet,
			CustomLabels:   stringMap(p.CustomLabels),
		},
		username:      p.Username,
		password:      p.Password,
		tlsSkipVerify: p.TlsSkipVerify,
		exporter: models.ChangeExporterOptions{
			MetricsScheme: p.Scheme,
			MetricsPath:   p.MetricsPath,
		},
		metricsMode:         p.MetricsMode,
		listenPort:          p.ListenPort,
		skipConnectionCheck: p.SkipConnectionCheck,
	}
}

func externalServiceChange(p *managementv1.UpdateExternalServiceParams) *serviceChange {
	c := &serviceChange{
		serviceType:  models.ExternalServiceType,
		exporterType: models.ExternalExporterType,
		service: models.ChangeServiceParams{
			Environment:    p.Environment,
			Cluster:        p.Cluster,
			ReplicationSet: p.ReplicationSet,
			CustomLabels:   stringMap(p.CustomLabels),
			ExternalGroup:  p.Group,
		},
		username:      p.Username,
		password:      p.Password,
		tlsSkipVerify: p.TlsSkipVerify,
		exporter: models.ChangeExporterOptions{
			MetricsScheme: p.Scheme,
			MetricsPath:   p.MetricsPath,
		},
		metricsMode:         p.MetricsMode,
		listenPort:          p.ListenPort,
		skipConnectionCheck: p.SkipConnectionCheck,
	}

	// The Service port mirrors the exporter's listen port, as when the Service is added.
	c.service.Port = port(p.ListenPort)

	return c
}

// stringMap returns nil for a missing map and a non-nil, possibly empty map otherwise, so that an empty map clears.
func stringMap(m *common.StringMap) *map[string]string {
	if m == nil {
		return nil
	}

	values := m.GetValues()
	if values == nil {
		values = map[string]string{}
	}

	return &values
}

// stringArray returns nil for a missing list and a non-nil, possibly empty slice otherwise, so that an empty list clears.
func stringArray(a *common.StringArray) []string {
	if a == nil {
		return nil
	}

	return append([]string{}, a.GetValues()...)
}

func port(p *uint32) *uint16 {
	if p == nil {
		return nil
	}

	return new(uint16(*p)) //nolint:gosec // validated to be less than 65536
}

// qanAgents keeps the QAN Agent types whose presence is requested.
func qanAgents(requested map[models.AgentType]*bool) map[models.AgentType]bool {
	res := make(map[models.AgentType]bool, len(requested))
	for agentType, present := range requested {
		if present != nil {
			res[agentType] = *present
		}
	}

	return res
}
