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

package management

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/logger"
)

const (
	// maximum time for AWS discover APIs calls
	awsDiscoverTimeout = 7 * time.Second
)

// RDSService represents instance discovery service.
type RDSService struct {
	db    *reform.DB
	state agentsStateUpdater
	cc    connectionChecker

	managementpb.UnimplementedRDSServer
}

// NewRDSService creates new instance discovery service.
func NewRDSService(db *reform.DB, state agentsStateUpdater, cc connectionChecker) *RDSService {
	return &RDSService{
		db:    db,
		state: state,
		cc:    cc,
	}
}

var (
	// See https://pkg.go.dev/github.com/aws/aws-sdk-go/service/rds?tab=doc#CreateDBInstanceInput, Engine field

	rdsEngines = map[string]managementpb.DiscoverRDSEngine{
		"aurora":       managementpb.DiscoverRDSEngine_DISCOVER_RDS_MYSQL, // MySQL 5.6-compatible Aurora
		"aurora-mysql": managementpb.DiscoverRDSEngine_DISCOVER_RDS_MYSQL, // MySQL 5.7-compatible Aurora
		"mariadb":      managementpb.DiscoverRDSEngine_DISCOVER_RDS_MYSQL,
		"mysql":        managementpb.DiscoverRDSEngine_DISCOVER_RDS_MYSQL,

		"aurora-postgresql": managementpb.DiscoverRDSEngine_DISCOVER_RDS_POSTGRESQL,
		"postgres":          managementpb.DiscoverRDSEngine_DISCOVER_RDS_POSTGRESQL,
	}
	rdsEnginesKeys = []*string{
		pointer.ToString("aurora"),
		pointer.ToString("aurora-mysql"),
		pointer.ToString("mariadb"),
		pointer.ToString("mysql"),

		pointer.ToString("aurora-postgresql"),
		pointer.ToString("postgres"),
	}
)

// discoverRDSRegion returns a list of RDS instances from a single region.
// Returned error is wrapped with a stack trace, but unchanged otherwise.
//nolint:interfacer
func discoverRDSRegion(ctx context.Context, sess *session.Session, region string) ([]*rds.DBInstance, error) {
	var res []*rds.DBInstance
	input := &rds.DescribeDBInstancesInput{
		Filters: []*rds.Filter{{
			Name:   pointer.ToString("engine"),
			Values: rdsEnginesKeys,
		}},
	}
	fn := func(out *rds.DescribeDBInstancesOutput, lastPage bool) bool {
		res = append(res, out.DBInstances...)
		return true // continue pagination
	}
	err := rds.New(sess, &aws.Config{Region: &region}).DescribeDBInstancesPagesWithContext(ctx, input, fn)
	if err != nil {
		return res, errors.WithStack(err)
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

			for r := range partition.Services()[endpoints.RdsServiceID].Regions() {
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
func (s *RDSService) DiscoverRDS(ctx context.Context, req *managementpb.DiscoverRDSRequest) (*managementpb.DiscoverRDSResponse, error) {
	l := logger.Get(ctx).WithField("component", "discover/rds")

	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		return nil, err
	}

	// use given credentials, or default credential chain
	var creds *credentials.Credentials
	if req.AwsAccessKey != "" || req.AwsSecretKey != "" {
		creds = credentials.NewStaticCredentials(req.AwsAccessKey, req.AwsSecretKey, "")
	}
	cfg := &aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(true),
		Credentials:                   creds,
		HTTPClient:                    &http.Client{},
	}
	if l.Logger.GetLevel() >= logrus.DebugLevel {
		cfg.LogLevel = aws.LogLevel(aws.LogDebug)
	}
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// do not break our API if some AWS region is slow or down
	ctx, cancel := context.WithTimeout(ctx, awsDiscoverTimeout)
	defer cancel()
	var wg errgroup.Group
	instances := make(chan *managementpb.DiscoverRDSInstance)

	for _, region := range listRegions(settings.AWSPartitions) {
		region := region
		wg.Go(func() error {
			regInstances, err := discoverRDSRegion(ctx, sess, region)
			if err != nil {
				l.Debugf("%s: %+v", region, err)
			}

			for _, db := range regInstances {
				l.Debugf("Discovered instance: %+v", db)

				instances <- &managementpb.DiscoverRDSInstance{
					Region:        region,
					Az:            *db.AvailabilityZone,
					InstanceId:    *db.DBInstanceIdentifier,
					NodeModel:     *db.DBInstanceClass,
					Address:       *db.Endpoint.Address,
					Port:          uint32(*db.Endpoint.Port),
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

	res := &managementpb.DiscoverRDSResponse{}
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
	if e, ok := errors.Cause(err).(awserr.Error); ok {
		switch {
		case e.Code() == "InvalidClientTokenId":
			return res, status.Error(codes.InvalidArgument, e.Message())
		case e.OrigErr() == context.Canceled || e.OrigErr() == context.DeadlineExceeded:
			return res, status.Error(codes.DeadlineExceeded, "Request timeout.")
		default:
			return res, status.Error(codes.Unknown, e.Error())
		}
	}
	return nil, err
}

// AddRDS adds RDS instance.
func (s *RDSService) AddRDS(ctx context.Context, req *managementpb.AddRDSRequest) (*managementpb.AddRDSResponse, error) {
	res := &managementpb.AddRDSResponse{}

	if e := s.db.InTransaction(func(tx *reform.TX) error {
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
		res.Node = invNode.(*inventorypb.RemoteRDSNode)

		// add RDSExporter Agent
		if req.RdsExporter {
			rdsExporter, err := models.CreateAgent(tx.Querier, models.RDSExporterType, &models.CreateAgentParams{
				PMMAgentID:                 models.PMMServerAgentID,
				NodeID:                     node.NodeID,
				AWSAccessKey:               req.AwsAccessKey,
				AWSSecretKey:               req.AwsSecretKey,
				RDSBasicMetricsDisabled:    req.DisableBasicMetrics,
				RDSEnhancedMetricsDisabled: req.DisableEnhancedMetrics,
			})
			if err != nil {
				return err
			}
			invRDSExporter, err := services.ToAPIAgent(tx.Querier, rdsExporter)
			if err != nil {
				return err
			}
			res.RdsExporter = invRDSExporter.(*inventorypb.RDSExporter)
		}

		switch req.Engine {
		case managementpb.DiscoverRDSEngine_DISCOVER_RDS_MYSQL:
			// add MySQL Service
			service, err := models.AddNewService(tx.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
				ServiceName:    req.ServiceName,
				NodeID:         node.NodeID,
				Environment:    req.Environment,
				Cluster:        req.Cluster,
				ReplicationSet: req.ReplicationSet,
				CustomLabels:   req.CustomLabels,
				Address:        &req.Address,
				Port:           pointer.ToUint16(uint16(req.Port)),
			})
			if err != nil {
				return err
			}
			invService, err := services.ToAPIService(service)
			if err != nil {
				return err
			}
			res.Mysql = invService.(*inventorypb.MySQLService)

			_, err = supportedMetricsMode(tx.Querier, req.MetricsMode, models.PMMServerAgentID)
			if err != nil {
				return err
			}

			// add MySQL Exporter
			mysqldExporter, err := models.CreateAgent(tx.Querier, models.MySQLdExporterType, &models.CreateAgentParams{
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
			invMySQLdExporter, err := services.ToAPIAgent(tx.Querier, mysqldExporter)
			if err != nil {
				return err
			}
			res.MysqldExporter = invMySQLdExporter.(*inventorypb.MySQLdExporter)

			if !req.SkipConnectionCheck {
				if err = s.cc.CheckConnectionToService(ctx, tx.Querier, service, mysqldExporter); err != nil {
					return err
				}
				// CheckConnectionToService updates the table count in row so, let's also update the response
				res.TableCount = *mysqldExporter.TableCount
			}

			// add MySQL PerfSchema QAN Agent
			if req.QanMysqlPerfschema {
				qanAgent, err := models.CreateAgent(tx.Querier, models.QANMySQLPerfSchemaAgentType, &models.CreateAgentParams{
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
				invQANAgent, err := services.ToAPIAgent(tx.Querier, qanAgent)
				if err != nil {
					return err
				}
				res.QanMysqlPerfschema = invQANAgent.(*inventorypb.QANMySQLPerfSchemaAgent)
			}

			return nil
		// PostgreSQL RDS
		case managementpb.DiscoverRDSEngine_DISCOVER_RDS_POSTGRESQL:
			// add PostgreSQL Service
			service, err := models.AddNewService(tx.Querier, models.PostgreSQLServiceType, &models.AddDBMSServiceParams{
				ServiceName:    req.ServiceName,
				NodeID:         node.NodeID,
				Environment:    req.Environment,
				Cluster:        req.Cluster,
				ReplicationSet: req.ReplicationSet,
				CustomLabels:   req.CustomLabels,
				Address:        &req.Address,
				Port:           pointer.ToUint16(uint16(req.Port)),
			})
			if err != nil {
				return err
			}
			invService, err := services.ToAPIService(service)
			if err != nil {
				return err
			}
			res.Postgresql = invService.(*inventorypb.PostgreSQLService)

			_, err = supportedMetricsMode(tx.Querier, req.MetricsMode, models.PMMServerAgentID)
			if err != nil {
				return err
			}

			// add PostgreSQL Exporter
			postgresExporter, err := models.CreateAgent(tx.Querier, models.PostgresExporterType, &models.CreateAgentParams{
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
			invPostgresExporter, err := services.ToAPIAgent(tx.Querier, postgresExporter)
			if err != nil {
				return err
			}
			res.PostgresqlExporter = invPostgresExporter.(*inventorypb.PostgresExporter)

			if !req.SkipConnectionCheck {
				if err = s.cc.CheckConnectionToService(ctx, tx.Querier, service, postgresExporter); err != nil {
					return err
				}
			}

			// add MySQL PerfSchema QAN Agent
			if req.QanPostgresqlPgstatements {
				qanAgent, err := models.CreateAgent(tx.Querier, models.QANPostgreSQLPgStatementsAgentType, &models.CreateAgentParams{
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
				invQANAgent, err := services.ToAPIAgent(tx.Querier, qanAgent)
				if err != nil {
					return err
				}
				res.QanPostgresqlPgstatements = invQANAgent.(*inventorypb.QANPostgreSQLPgStatementsAgent)
			}

			return nil

		default:
			return status.Errorf(codes.InvalidArgument, "Unsupported Engine type %q.", req.Engine)
		}
	}); e != nil {
		return nil, e
	}

	s.state.RequestStateUpdate(ctx, models.PMMServerAgentID)
	return res, nil
}
