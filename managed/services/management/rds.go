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
	"net/http"
	"sort"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/smithy-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/utils/logger"
)

const (
	// Maximum time for AWS discover APIs calls.
	awsDiscoverTimeout = 7 * time.Second
)

var (
	// See https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/rds?tab=doc#CreateDBInstanceInput, Engine field.

	rdsEngines = map[string]managementv1.DiscoverRDSEngine{
		"aurora-mysql": managementv1.DiscoverRDSEngine_DISCOVER_RDS_ENGINE_MYSQL, // MySQL 5.7-compatible Aurora
		"mariadb":      managementv1.DiscoverRDSEngine_DISCOVER_RDS_ENGINE_MYSQL,
		"mysql":        managementv1.DiscoverRDSEngine_DISCOVER_RDS_ENGINE_MYSQL,

		"aurora-postgresql": managementv1.DiscoverRDSEngine_DISCOVER_RDS_ENGINE_POSTGRESQL,
		"postgres":          managementv1.DiscoverRDSEngine_DISCOVER_RDS_ENGINE_POSTGRESQL,
	}
	rdsEnginesKeys = []*string{
		pointer.ToString("aurora-mysql"),
		pointer.ToString("mariadb"),
		pointer.ToString("mysql"),

		pointer.ToString("aurora-postgresql"),
		pointer.ToString("postgres"),
	}
)

// discoverRDSRegion returns a list of RDS instances from a single region.
// Returned error is wrapped with a stack trace, but unchanged otherwise.
//
//nolint:interfacer
func discoverRDSRegion(ctx context.Context, cfg aws.Config, region string) ([]types.DBInstance, error) {
	var res []types.DBInstance

	// Create RDS client scoped to the region
	client := rds.NewFromConfig(cfg, func(o *rds.Options) {
		o.Region = region
	})

	input := &rds.DescribeDBInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("engine"),
				Values: []string{"postgres", "mysql"},
			},
		},
	}

	paginator := rds.NewDescribeDBInstancesPaginator(client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return res, err
		}
		res = append(res, page.DBInstances...)
	}

	return res, nil
}

// listRegions returns a list of AWS regions for given partitions.
func listRegions(partitions []string) []string {
	set := make(map[string]struct{})
	for _, p := range partitions {
		for _, partition := range endpoints.DefaultPartitions() {
			if p != partition.ID() {
				continue
			}

			for r := range partition.Services()["aws"].Regions() {
				set[r] = struct{}{}
			}
			break
		}
	}

	slice := make([]string, 0, len(set))
	for r := range set {
		slice = append(slice, r)
	}
	sort.Strings(slice)

	return slice
}

// DiscoverRDS discovers RDS instances.
func (s *ManagementService) DiscoverRDS(ctx context.Context, req *managementv1.DiscoverRDSRequest) (*managementv1.DiscoverRDSResponse, error) {
	l := logger.Get(ctx).WithField("component", "discover/rds")

	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		return nil, err
	}

	// use given credentials, or default credential chain
	var creds aws.CredentialsProvider
	if req.AwsAccessKey != "" && req.AwsSecretKey != "" {
		creds = credentials.NewStaticCredentialsProvider(req.AwsAccessKey, req.AwsSecretKey, "")
	}
	// cfg := &aws.Config{
	// 	CredentialsChainVerboseErrors: aws.Bool(true),
	// 	Credentials:                   creds,
	// 	HTTPClient:                    &http.Client{},
	// }
	// if l.Logger.GetLevel() >= logrus.DebugLevel {
	// 	cfg.LogLevel = aws.LogLevel(aws.LogDebug)
	// }
	// sess, err := session.NewSession(cfg)
	// if err != nil {
	// 	return nil, errors.WithStack(err)
	// }

	opts := []func(*config.LoadOptions) error{
		config.WithCredentialsProvider(creds),
		config.WithHTTPClient(&http.Client{}),
	}

	// Enable verbose credentials chain errors equivalent in v2 (no direct flag, but can debug via logs)
	// Enable logging if log level is debug or higher
	if l.Logger != nil && l.Logger.Level >= logrus.DebugLevel {
		opts = append(opts, config.WithClientLogMode(aws.LogRetries|aws.LogRequestWithBody|aws.LogResponseWithBody))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// do not break our API if some AWS region is slow or down
	ctx, cancel := context.WithTimeout(ctx, awsDiscoverTimeout)
	defer cancel()
	var wg errgroup.Group
	instances := make(chan *managementv1.DiscoverRDSInstance)

	for _, region := range listRegions(settings.AWSPartitions) {
		region := region
		wg.Go(func() error {
			regInstances, err := discoverRDSRegion(ctx, cfg, region)
			if err != nil {
				l.Debugf("%s: %+v", region, err)
			}

			for _, db := range regInstances {
				l.Debugf("Discovered instance: %+v", db)

				// This happens when the database is in "Creating" state.
				// At this point there is no endpoint available.
				if db.Endpoint == nil {
					l.Debugf("Instance %q not ready yet. Please wait until the database is fully created in AWS.", *db.DBInstanceIdentifier)
					continue
				}

				instances <- &managementv1.DiscoverRDSInstance{
					Region:        region,
					Az:            *db.AvailabilityZone,
					InstanceId:    *db.DBInstanceIdentifier,
					NodeModel:     *db.DBInstanceClass,
					Address:       *db.Endpoint.Address,
					Port:          uint32(*db.Endpoint.Port), //nolint:gosec // port is not expected to overflow uint32
					Engine:        rdsEngines[*db.Engine],
					EngineVersion: *db.EngineVersion,
				}
			}

			return err
		})
	}

	go func() {
		_ = wg.Wait() // checked below
		close(instances)
	}()

	res := &managementv1.DiscoverRDSResponse{}
	for i := range instances {
		res.RdsInstances = append(res.RdsInstances, i)
	}

	// sort by region and id
	sort.Slice(res.RdsInstances, func(i, j int) bool {
		if res.RdsInstances[i].Region != res.RdsInstances[j].Region {
			return res.RdsInstances[i].Region < res.RdsInstances[j].Region
		}
		return res.RdsInstances[i].InstanceId < res.RdsInstances[j].InstanceId
	})

	// ignore error if there are some results
	if len(res.RdsInstances) != 0 {
		return res, nil
	}

	// return better gRPC errors in typical cases
	err = wg.Wait()
	// if e, ok := errors.Cause(err).(awserr.Error); ok { //nolint:errorlint
	// 	switch {
	// 	case e.Code() == "InvalidClientTokenId":
	// 		return res, status.Error(codes.InvalidArgument, e.Message())
	// 	case errors.Is(e.OrigErr(), context.Canceled) || errors.Is(e.OrigErr(), context.DeadlineExceeded):
	// 		return res, status.Error(codes.DeadlineExceeded, "Request timeout.")
	// 	default:
	// 		return res, status.Error(codes.Unknown, e.Error())
	// 	}
	// }
	if err != nil {
		var apiErr *smithy.GenericAPIError
		if errors.As(err, &apiErr) {
			switch {
			case apiErr.Code == "InvalidClientTokenId":
				return res, status.Error(codes.InvalidArgument, apiErr.Message)

			case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
				return res, status.Error(codes.DeadlineExceeded, "Request timeout.")

			default:
				return res, status.Error(codes.Unknown, apiErr.Error())
			}
		}

		// If not an AWS API error, but maybe a context error:
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return res, status.Error(codes.DeadlineExceeded, "Request timeout.")
		}

		// Fallback unknown error:
		return res, status.Error(codes.Unknown, err.Error())
	}

	return nil, err
}

// AddRDS adds RDS instance.
func (s *ManagementService) addRDS(ctx context.Context, req *managementv1.AddRDSServiceParams) (*managementv1.AddServiceResponse, error) { //nolint:cyclop,maintidx
	rds := &managementv1.RDSServiceResult{}

	pmmAgentID := models.PMMServerAgentID
	if req.GetPmmAgentId() != "" {
		pmmAgentID = req.GetPmmAgentId()
	}

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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

		// add RemoteRDS Node
		node, err := models.CreateNode(tx.Querier, models.RemoteRDSNodeType, &models.CreateNodeParams{
			NodeName:     req.NodeName,
			NodeModel:    req.NodeModel,
			AZ:           req.Az,
			Address:      req.InstanceId,
			Region:       &req.Region,
			CustomLabels: req.CustomLabels,
		})
		if err != nil {
			return err
		}
		invNode, err := services.ToAPINode(node)
		if err != nil {
			return err
		}
		rds.Node = invNode.(*inventoryv1.RemoteRDSNode) //nolint:forcetypeassert

		metricsMode, err := supportedMetricsMode(req.MetricsMode, pmmAgentID)
		if err != nil {
			return err
		}

		// add RDSExporter Agent
		if req.RdsExporter {
			rdsExporter, err := models.CreateAgent(tx.Querier, models.RDSExporterType, &models.CreateAgentParams{
				PMMAgentID: pmmAgentID,
				NodeID:     node.NodeID,
				AWSOptions: models.AWSOptions{
					AWSAccessKey:               req.AwsAccessKey,
					AWSSecretKey:               req.AwsSecretKey,
					RDSBasicMetricsDisabled:    req.DisableBasicMetrics,
					RDSEnhancedMetricsDisabled: req.DisableEnhancedMetrics,
				},
				ExporterOptions: models.ExporterOptions{
					PushMetrics: isPushMode(metricsMode),
				},
			})
			if err != nil {
				return err
			}
			invRDSExporter, err := services.ToAPIAgent(tx.Querier, rdsExporter)
			if err != nil {
				return err
			}
			rds.RdsExporter = invRDSExporter.(*inventoryv1.RDSExporter) //nolint:forcetypeassert
		}

		switch req.Engine {
		case managementv1.DiscoverRDSEngine_DISCOVER_RDS_ENGINE_MYSQL:
			// add MySQL Service
			service, err := models.AddNewService(tx.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
				ServiceName:    req.ServiceName,
				NodeID:         node.NodeID,
				Environment:    req.Environment,
				Cluster:        req.Cluster,
				ReplicationSet: req.ReplicationSet,
				CustomLabels:   req.CustomLabels,
				Address:        &req.Address,
				Port:           pointer.ToUint16(uint16(req.Port)), //nolint:gosec // port is not expected to overflow uint16
			})
			if err != nil {
				return err
			}
			invService, err := services.ToAPIService(service)
			if err != nil {
				return err
			}
			rds.Mysql = invService.(*inventoryv1.MySQLService) //nolint:forcetypeassert

			// add MySQL Exporter
			mysqldExporter, err := models.CreateAgent(tx.Querier, models.MySQLdExporterType, &models.CreateAgentParams{
				PMMAgentID:    pmmAgentID,
				ServiceID:     service.ServiceID,
				Username:      req.Username,
				Password:      req.Password,
				TLS:           req.Tls,
				TLSSkipVerify: req.TlsSkipVerify,
				ExporterOptions: models.ExporterOptions{
					PushMetrics: isPushMode(metricsMode),
				},
				MySQLOptions: models.MySQLOptions{
					TableCountTablestatsGroupLimit: tablestatsGroupTableLimit,
				},
			})
			if err != nil {
				return err
			}
			invMySQLdExporter, err := services.ToAPIAgent(tx.Querier, mysqldExporter)
			if err != nil {
				return err
			}
			rds.MysqldExporter = invMySQLdExporter.(*inventoryv1.MySQLdExporter) //nolint:forcetypeassert

			if !req.SkipConnectionCheck {
				if err = s.cc.CheckConnectionToService(ctx, tx.Querier, service, mysqldExporter); err != nil {
					return err
				}
				if err = s.sib.GetInfoFromService(ctx, tx.Querier, service, mysqldExporter); err != nil {
					return err
				}
			}

			// add MySQL PerfSchema QAN Agent
			if req.QanMysqlPerfschema {
				qanAgent, err := models.CreateAgent(tx.Querier, models.QANMySQLPerfSchemaAgentType, &models.CreateAgentParams{
					PMMAgentID:    pmmAgentID,
					ServiceID:     service.ServiceID,
					Username:      req.Username,
					Password:      req.Password,
					TLS:           req.Tls,
					TLSSkipVerify: req.TlsSkipVerify,
					QANOptions: models.QANOptions{
						QueryExamplesDisabled:   req.DisableQueryExamples,
						CommentsParsingDisabled: req.DisableCommentsParsing,
					},
				})
				if err != nil {
					return err
				}
				invQANAgent, err := services.ToAPIAgent(tx.Querier, qanAgent)
				if err != nil {
					return err
				}
				rds.QanMysqlPerfschema = invQANAgent.(*inventoryv1.QANMySQLPerfSchemaAgent) //nolint:forcetypeassert
			}

			return nil
		// PostgreSQL RDS
		case managementv1.DiscoverRDSEngine_DISCOVER_RDS_ENGINE_POSTGRESQL:
			// add PostgreSQL Service
			service, err := models.AddNewService(tx.Querier, models.PostgreSQLServiceType, &models.AddDBMSServiceParams{
				ServiceName:    req.ServiceName,
				NodeID:         node.NodeID,
				Environment:    req.Environment,
				Cluster:        req.Cluster,
				ReplicationSet: req.ReplicationSet,
				CustomLabels:   req.CustomLabels,
				Address:        &req.Address,
				Port:           pointer.ToUint16(uint16(req.Port)), //nolint:gosec // port is not expected to overflow uint16
				Database:       req.Database,
			})
			if err != nil {
				return err
			}
			invService, err := services.ToAPIService(service)
			if err != nil {
				return err
			}
			rds.Postgresql = invService.(*inventoryv1.PostgreSQLService) //nolint:forcetypeassert

			// add PostgreSQL Exporter
			postgresExporter, err := models.CreateAgent(tx.Querier, models.PostgresExporterType, &models.CreateAgentParams{
				PMMAgentID:    pmmAgentID,
				ServiceID:     service.ServiceID,
				Username:      req.Username,
				Password:      req.Password,
				TLS:           req.Tls,
				TLSSkipVerify: req.TlsSkipVerify,
				ExporterOptions: models.ExporterOptions{
					PushMetrics: isPushMode(metricsMode),
				},
				MySQLOptions: models.MySQLOptions{
					TableCountTablestatsGroupLimit: tablestatsGroupTableLimit,
				},

				PostgreSQLOptions: models.PostgreSQLOptions{
					AutoDiscoveryLimit:     pointer.ToInt32(req.AutoDiscoveryLimit),
					MaxExporterConnections: req.MaxPostgresqlExporterConnections,
				},
			})
			if err != nil {
				return err
			}
			invPostgresExporter, err := services.ToAPIAgent(tx.Querier, postgresExporter)
			if err != nil {
				return err
			}
			rds.PostgresqlExporter = invPostgresExporter.(*inventoryv1.PostgresExporter) //nolint:forcetypeassert

			if !req.SkipConnectionCheck {
				if err = s.cc.CheckConnectionToService(ctx, tx.Querier, service, postgresExporter); err != nil {
					return err
				}
				if err = s.sib.GetInfoFromService(ctx, tx.Querier, service, postgresExporter); err != nil {
					return err
				}
			}

			// add PostgreSQL Pgstatements QAN Agent
			if req.QanPostgresqlPgstatements {
				qanAgent, err := models.CreateAgent(tx.Querier, models.QANPostgreSQLPgStatementsAgentType, &models.CreateAgentParams{
					PMMAgentID:    pmmAgentID,
					ServiceID:     service.ServiceID,
					Username:      req.Username,
					Password:      req.Password,
					TLS:           req.Tls,
					TLSSkipVerify: req.TlsSkipVerify,
					QANOptions: models.QANOptions{
						QueryExamplesDisabled:   req.DisableQueryExamples,
						CommentsParsingDisabled: req.DisableCommentsParsing,
					},
				})
				if err != nil {
					return err
				}
				invQANAgent, err := services.ToAPIAgent(tx.Querier, qanAgent)
				if err != nil {
					return err
				}
				rds.QanPostgresqlPgstatements = invQANAgent.(*inventoryv1.QANPostgreSQLPgStatementsAgent) //nolint:forcetypeassert
			}

			return nil

		default:
			return status.Errorf(codes.InvalidArgument, "Unsupported Engine type %q.", req.Engine)
		}
	})

	if errTx != nil {
		return nil, errTx
	}

	s.state.RequestStateUpdate(ctx, pmmAgentID)

	res := &managementv1.AddServiceResponse{
		Service: &managementv1.AddServiceResponse_Rds{
			Rds: rds,
		},
	}
	return res, nil
}
