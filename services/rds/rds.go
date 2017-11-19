// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Package rds contains business logic of working with AWS RDS.
package rds

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"sort"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/go-sql-driver/mysql"
	servicelib "github.com/percona/kardianos-service"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/prometheus"
	"github.com/percona/pmm-managed/services/supervisor"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/ports"
)

const awsCallTimeout = 5 * time.Second

type ServiceConfig struct {
	MySQLdExporterPath string

	DB            *reform.DB
	Prometheus    *prometheus.Service
	Supervisor    *supervisor.Supervisor
	PortsRegistry *ports.Registry
}

// Service is responsible for interactions with AWS RDS.
type Service struct {
	*ServiceConfig
	httpClient    *http.Client
	pmmServerNode *models.Node
}

// NewService creates a new service.
func NewService(config *ServiceConfig) (*Service, error) {
	var node models.Node
	err := config.DB.FindOneTo(&node, "type", models.PMMServerNodeType)
	if err != nil {
		return nil, err
	}

	if config == nil {
		config = &ServiceConfig{}
	}
	for _, path := range []*string{
		&config.MySQLdExporterPath,
	} {
		if *path == "" {
			continue
		}
		p, err := exec.LookPath(*path)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		*path = p
	}

	svc := &Service{
		ServiceConfig: config,
		httpClient:    new(http.Client),
		pmmServerNode: &node,
	}
	return svc, nil
}

// InstanceID uniquely identifies RDS instance.
// http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.html
// Each DB instance has a DB instance identifier. This customer-supplied name uniquely identifies the DB instance when interacting
// with the Amazon RDS API and AWS CLI commands. The DB instance identifier must be unique for that customer in an AWS Region.
type InstanceID struct {
	Region string
	Name   string // DBInstanceIdentifier
}

type Instance struct {
	Node    models.RDSNode
	Service models.RDSService
}

func (svc *Service) Discover(ctx context.Context, accessKey, secretKey string) ([]Instance, error) {
	l := logger.Get(ctx).WithField("component", "rds")

	// do not break our API if some AWS region is slow or down
	ctx, cancel := context.WithTimeout(ctx, awsCallTimeout)
	defer cancel()
	var g errgroup.Group
	instances := make(chan Instance)

	for _, r := range endpoints.AwsPartition().Services()[endpoints.RdsServiceID].Regions() {
		regionId := r.ID()
		g.Go(func() error {
			// use given credentials, or default credential chain
			var creds *credentials.Credentials
			if accessKey != "" || secretKey != "" {
				creds = credentials.NewCredentials(&credentials.StaticProvider{
					Value: credentials.Value{
						AccessKeyID:     accessKey,
						SecretAccessKey: secretKey,
					},
				})
			}
			config := &aws.Config{
				CredentialsChainVerboseErrors: aws.Bool(true),
				Credentials:                   creds,
				Region:                        aws.String(regionId),
				HTTPClient:                    svc.httpClient,
				Logger:                        aws.LoggerFunc(l.Debug),
			}
			if l.Level >= logrus.DebugLevel {
				config.LogLevel = aws.LogLevel(aws.LogDebug)
			}
			s, err := session.NewSession(config)
			if err != nil {
				return errors.WithStack(err)
			}

			out, err := rds.New(s).DescribeDBInstancesWithContext(ctx, new(rds.DescribeDBInstancesInput))
			if err != nil {
				l.Error(err)

				if err, ok := err.(awserr.Error); ok {
					if err.OrigErr() != nil && err.OrigErr() == ctx.Err() {
						// ignore timeout, let other goroutines return partial data
						return nil
					}
					switch err.Code() {
					case "InvalidClientTokenId", "EmptyStaticCreds":
						return status.Error(codes.InvalidArgument, err.Message())
					default:
						return err
					}
				}
				return errors.WithStack(err)
			}

			l.Debugf("Got %d instances from %s.", len(out.DBInstances), regionId)
			for _, db := range out.DBInstances {
				instances <- Instance{
					Node: models.RDSNode{
						Type: models.RDSNodeType,

						Name:   *db.DBInstanceIdentifier,
						Region: regionId,
					},
					Service: models.RDSService{
						Type: models.RDSServiceType,

						Address:       db.Endpoint.Address,
						Port:          pointer.ToUint16(uint16(*db.Endpoint.Port)),
						Engine:        db.Engine,
						EngineVersion: db.EngineVersion,
					},
				}
			}
			return nil
		})
	}

	go func() {
		g.Wait()
		close(instances)
	}()

	res := []Instance{}
	for i := range instances {
		res = append(res, i)
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].Node.Region != res[j].Node.Region {
			return res[i].Node.Region < res[j].Node.Region
		}
		return res[i].Node.Name < res[j].Node.Name
	})
	return res, g.Wait()
}

func (svc *Service) List(ctx context.Context) ([]Instance, error) {
	res := []Instance{}
	err := svc.DB.InTransaction(func(tx *reform.TX) error {
		structs, e := tx.SelectAllFrom(models.RDSNodeTable, "WHERE type = ? ORDER BY id", models.RDSNodeType)
		if e != nil {
			return e
		}
		nodes := make([]models.RDSNode, len(structs))
		for i, str := range structs {
			nodes[i] = *str.(*models.RDSNode)
		}

		structs, e = tx.SelectAllFrom(models.RDSServiceTable, "WHERE type = ? ORDER BY id", models.RDSServiceType)
		if e != nil {
			return e
		}
		services := make([]models.RDSService, len(structs))
		for i, str := range structs {
			services[i] = *str.(*models.RDSService)
		}

		for _, node := range nodes {
			for _, service := range services {
				if node.ID == service.NodeID {
					res = append(res, Instance{
						Node:    node,
						Service: service,
					})
				}
			}
		}
		return nil
	})
	return res, err
}

func (svc *Service) Add(ctx context.Context, accessKey, secretKey string, id *InstanceID, username, password string) error {
	if id.Name == "" {
		return status.Error(codes.InvalidArgument, "RDS instance name is not given.")
	}
	if id.Region == "" {
		return status.Error(codes.InvalidArgument, "RDS instance region is not given.")
	}
	if username == "" {
		return status.Error(codes.InvalidArgument, "Username is not given.")
	}

	instances, err := svc.Discover(ctx, accessKey, secretKey)
	if err != nil {
		return err
	}

	var add *Instance
	for _, instance := range instances {
		if instance.Node.Name == id.Name && instance.Node.Region == id.Region {
			add = &instance
			break
		}
	}
	if add == nil {
		return status.Errorf(codes.NotFound, "RDS instance %q not found in region %q.", id.Name, id.Region)
	}

	return svc.DB.InTransaction(func(tx *reform.TX) error {
		// insert node
		node := &models.RDSNode{
			Type: models.RDSNodeType,
			Name: add.Node.Name,

			Region: add.Node.Region,
		}
		if e := tx.Insert(node); e != nil {
			if e, ok := e.(*mysql.MySQLError); ok && e.Number == 0x426 {
				return status.Errorf(codes.AlreadyExists, "RDS instance %q already exists in region %q.",
					node.Name, node.Region)
			}
			return errors.WithStack(e)
		}

		// insert service
		service := &models.RDSService{
			Type:   models.RDSServiceType,
			NodeID: node.ID,

			Address:       add.Service.Address,
			Port:          add.Service.Port,
			Engine:        add.Service.Engine,
			EngineVersion: add.Service.EngineVersion,
		}
		if accessKey != "" || secretKey != "" {
			service.AWSAccessKey = &accessKey
			service.AWSSecretKey = &secretKey
		}
		if e := tx.Insert(service); e != nil {
			return errors.WithStack(e)
		}

		// insert mysqld_exporter agent
		port, e := svc.PortsRegistry.Reserve()
		if e != nil {
			return e
		}
		agent := &models.MySQLdExporter{
			Type:         models.MySQLdExporterAgentType,
			RunsOnNodeID: svc.pmmServerNode.ID,

			ServiceUsername: &username,
			ServicePassword: &password,
			ListenPort:      &port,
		}
		if e := tx.Insert(agent); e != nil {
			return errors.WithStack(e)
		}
		if e := tx.Insert(&models.AgentService{AgentID: agent.ID, ServiceID: service.ID}); e != nil {
			return errors.WithStack(e)
		}

		// start mysqld_exporter agent
		if svc.MySQLdExporterPath != "" {
			name := agent.NameForSupervisor()
			cfg := &servicelib.Config{
				Name:        name,
				DisplayName: name,
				Description: name,
				Executable:  svc.MySQLdExporterPath,
				Arguments: []string{
					// TODO use proper flags
					"-collect.auto_increment.columns",
					"-collect.binlog_size",
					"-collect.global_status",
					"-collect.global_variables",
					"-collect.info_schema.innodb_metrics",
					"-collect.info_schema.processlist",
					"-collect.info_schema.query_response_time",
					"-collect.info_schema.tables",
					"-collect.info_schema.tablestats",
					"-collect.info_schema.userstats",
					"-collect.perf_schema.eventswaits",
					"-collect.perf_schema.file_events",
					"-collect.perf_schema.indexiowaits",
					"-collect.perf_schema.tableiowaits",
					"-collect.perf_schema.tablelocks",
					"-collect.slave_status",

					fmt.Sprintf("-web.listen-address=127.0.0.1:%d", port),
				},
				Environment: []string{fmt.Sprintf("DATA_SOURCE_NAME=%s", agent.DSN(service))},
			}
			if e := svc.Supervisor.Start(ctx, cfg); e != nil {
				return e
			}
		}

		// TODO use proper flags

		// TODO insert and start other agents

		// if e := tx.Insert(&models.AgentNode{AgentID: agent.ID, NodeID: node.ID}); e != nil {
		// 	return errors.WithStack(e)
		// }

		return nil
	})
}

func (svc *Service) Remove(ctx context.Context, id *InstanceID) error {
	if id.Name == "" {
		return status.Error(codes.InvalidArgument, "RDS instance name is not given.")
	}
	if id.Region == "" {
		return status.Error(codes.InvalidArgument, "RDS instance region is not given.")
	}

	return svc.DB.InTransaction(func(tx *reform.TX) error {
		var node models.RDSNode
		if e := tx.SelectOneTo(&node, "WHERE type = ? AND name = ? AND region = ?", models.RDSNodeType, id.Name, id.Region); e != nil {
			if e == reform.ErrNoRows {
				return status.Errorf(codes.NotFound, "RDS instance %q not found in region %q.", id.Name, id.Region)
			}
			return errors.WithStack(e)
		}

		var service models.RDSService
		if e := tx.SelectOneTo(&service, "WHERE node_id = ?", node.ID); e != nil {
			return errors.WithStack(e)
		}

		var agents []models.Agent

		// remove associations of the service and agents
		structs, e := tx.FindAllFrom(models.AgentServiceView, "service_id", service.ID)
		if e != nil {
			return errors.WithStack(e)
		}
		for _, str := range structs {
			agentService := str.(*models.AgentService)
			agent := models.Agent{ID: agentService.AgentID}
			agents = append(agents, agent)
			deleted, e := tx.DeleteFrom(models.AgentServiceView, "WHERE service_id = ? AND agent_id = ?", service.ID, agent.ID)
			if e == nil && deleted != 1 {
				e = errors.Errorf("expected to delete 1 record, deleted %d", deleted)
			}
			if e != nil {
				return errors.WithStack(e)
			}
		}

		// remove associations of the node and agents
		structs, e = tx.FindAllFrom(models.AgentNodeView, "node_id", node.ID)
		if e != nil {
			return errors.WithStack(e)
		}
		for _, str := range structs {
			agentNode := str.(*models.AgentNode)
			agent := models.Agent{ID: agentNode.AgentID}
			agents = append(agents, agent)
			deleted, e := tx.DeleteFrom(models.AgentNodeView, "WHERE node_id = ? AND agent_id = ?", node.ID, agent.ID)
			if e == nil && deleted != 1 {
				e = errors.Errorf("expected to delete 1 record, deleted %d", deleted)
			}
			if e != nil {
				return errors.WithStack(e)
			}
		}

		// stop agents
		for _, agent := range agents {
			// fill type
			if e := tx.Reload(&agent); e != nil {
				return errors.WithStack(e)
			}

			var name string
			switch agent.Type {
			case models.MySQLdExporterAgentType:
				a := models.MySQLdExporter{ID: agent.ID}
				if e := tx.Reload(&a); e != nil {
					return errors.WithStack(e)
				}
				if svc.MySQLdExporterPath != "" {
					name = a.NameForSupervisor()
				}
			}

			if name != "" {
				if e := svc.Supervisor.Stop(ctx, name); e != nil {
					return e
				}
			}
		}

		// remove agents
		for _, agent := range agents {
			if e := tx.Delete(&agent); e != nil {
				return errors.WithStack(e)
			}
		}

		if e := tx.Delete(&service); e != nil {
			return errors.WithStack(e)
		}
		if e := tx.Delete(&node); e != nil {
			return errors.WithStack(e)
		}

		return nil
	})
}
