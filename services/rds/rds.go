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
	"database/sql"
	"fmt"
	"io/ioutil"
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
	"github.com/percona/pmm-managed/services/qan"
	"github.com/percona/pmm-managed/services/supervisor"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/ports"
)

const (
	awsCallTimeout         = 7 * time.Second
	qanAgentPort    uint16 = 9000
	rdsExporterPort uint16 = 9042
)

type ServiceConfig struct {
	MySQLdExporterPath    string
	RDSExporterPath       string
	RDSExporterConfigPath string

	DB            *reform.DB
	Prometheus    *prometheus.Service
	QAN           *qan.Service
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

	for _, path := range []*string{
		&config.MySQLdExporterPath,
		&config.RDSExporterPath,
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

func (svc *Service) ApplyPrometheusConfiguration(ctx context.Context, q *reform.Querier) error {
	rdsMySQLHR := &prometheus.ScrapeConfig{
		JobName:        "rds-mysql-hr",
		ScrapeInterval: "1s",
		ScrapeTimeout:  "1s",
		MetricsPath:    "/metrics-hr",
		RelabelConfigs: []prometheus.RelabelConfig{{
			TargetLabel: "job",
			Replacement: "mysql",
		}},
	}
	rdsMySQLMR := &prometheus.ScrapeConfig{
		JobName:        "rds-mysql-mr",
		ScrapeInterval: "5s",
		ScrapeTimeout:  "1s",
		MetricsPath:    "/metrics-mr",
		RelabelConfigs: []prometheus.RelabelConfig{{
			TargetLabel: "job",
			Replacement: "mysql",
		}},
	}
	rdsMySQLLR := &prometheus.ScrapeConfig{
		JobName:        "rds-mysql-lr",
		ScrapeInterval: "60s",
		ScrapeTimeout:  "5s",
		MetricsPath:    "/metrics-lr",
		RelabelConfigs: []prometheus.RelabelConfig{{
			TargetLabel: "job",
			Replacement: "mysql",
		}},
	}
	rdsBasic := &prometheus.ScrapeConfig{
		JobName:        "rds-basic",
		ScrapeInterval: "60s",
		ScrapeTimeout:  "55s",
		MetricsPath:    "/basic",
	}
	rdsEnhanced := &prometheus.ScrapeConfig{
		JobName:        "rds-enhanced",
		ScrapeInterval: "10s",
		ScrapeTimeout:  "9s",
		MetricsPath:    "/enhanced",
	}

	nodes, err := q.FindAllFrom(models.RDSNodeTable, "type", models.RDSNodeType)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, n := range nodes {
		node := n.(*models.RDSNode)

		var service models.RDSService
		if e := q.SelectOneTo(&service, "WHERE node_id = ?", node.ID); e != nil {
			return errors.WithStack(e)
		}

		agents, err := models.AgentsForServiceID(q, service.ID)
		if err != nil {
			return err
		}
		for _, agent := range agents {
			switch agent.Type {
			case models.MySQLdExporterAgentType:
				a := models.MySQLdExporter{ID: agent.ID}
				if e := q.Reload(&a); e != nil {
					return errors.WithStack(e)
				}
				logger.Get(ctx).Infof("%s %s %s %d", a.Type, node.Name, node.Region, *a.ListenPort)

				sc := prometheus.StaticConfig{
					Targets: []string{fmt.Sprintf("127.0.0.1:%d", *a.ListenPort)},
					Labels: []prometheus.LabelPair{
						{Name: "aws_region", Value: node.Region},
						{Name: "instance", Value: node.Name},
					},
				}
				rdsMySQLHR.StaticConfigs = append(rdsMySQLHR.StaticConfigs, sc)
				rdsMySQLMR.StaticConfigs = append(rdsMySQLMR.StaticConfigs, sc)
				rdsMySQLLR.StaticConfigs = append(rdsMySQLLR.StaticConfigs, sc)

			case models.RDSExporterAgentType:
				a := models.RDSExporter{ID: agent.ID}
				if e := q.Reload(&a); e != nil {
					return errors.WithStack(e)
				}
				logger.Get(ctx).Infof("%s %s %s %d", a.Type, node.Name, node.Region, *a.ListenPort)

				sc := prometheus.StaticConfig{
					Targets: []string{fmt.Sprintf("127.0.0.1:%d", *a.ListenPort)},
					Labels: []prometheus.LabelPair{
						{Name: "aws_region", Value: node.Region},
						{Name: "instance", Value: node.Name},
					},
				}
				rdsBasic.StaticConfigs = append(rdsBasic.StaticConfigs, sc)
				rdsEnhanced.StaticConfigs = append(rdsEnhanced.StaticConfigs, sc)
			}
		}
	}

	// sort by region and name
	sorterFor := func(sc []prometheus.StaticConfig) func(int, int) bool {
		return func(i, j int) bool {
			if sc[i].Labels[0].Value != sc[j].Labels[0].Value {
				return sc[i].Labels[0].Value < sc[j].Labels[0].Value
			}
			return sc[i].Labels[1].Value < sc[j].Labels[1].Value
		}
	}
	sort.Slice(rdsMySQLHR.StaticConfigs, sorterFor(rdsMySQLHR.StaticConfigs))
	sort.Slice(rdsMySQLMR.StaticConfigs, sorterFor(rdsMySQLMR.StaticConfigs))
	sort.Slice(rdsMySQLLR.StaticConfigs, sorterFor(rdsMySQLLR.StaticConfigs))
	sort.Slice(rdsBasic.StaticConfigs, sorterFor(rdsBasic.StaticConfigs))
	sort.Slice(rdsEnhanced.StaticConfigs, sorterFor(rdsEnhanced.StaticConfigs))

	return svc.Prometheus.SetScrapeConfigs(ctx, false, rdsMySQLHR, rdsMySQLMR, rdsMySQLLR, rdsBasic, rdsEnhanced)
}

func (svc *Service) Discover(ctx context.Context, accessKey, secretKey string) ([]Instance, error) {
	l := logger.Get(ctx).WithField("component", "rds")

	// do not break our API if some AWS region is slow or down
	ctx, cancel := context.WithTimeout(ctx, awsCallTimeout)
	defer cancel()
	var g errgroup.Group
	instances := make(chan Instance)

	for _, r := range endpoints.AwsPartition().Services()[endpoints.RdsServiceID].Regions() {
		region := r.ID()
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
				Region:                        aws.String(region),
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

			l.Debugf("Got %d instances from %s.", len(out.DBInstances), region)
			for _, db := range out.DBInstances {
				instances <- Instance{
					Node: models.RDSNode{
						Type: models.RDSNodeType,

						Name:   *db.DBInstanceIdentifier,
						Region: region,
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

	// sort by region and name
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

func (svc *Service) addMySQLdExporter(ctx context.Context, tx *reform.TX, service *models.RDSService, username, password string) error {
	// insert mysqld_exporter agent and association
	port, err := svc.PortsRegistry.Reserve()
	if err != nil {
		return err
	}
	agent := &models.MySQLdExporter{
		Type:         models.MySQLdExporterAgentType,
		RunsOnNodeID: svc.pmmServerNode.ID,

		ServiceUsername: &username,
		ServicePassword: &password,
		ListenPort:      &port,
	}
	if err = tx.Insert(agent); err != nil {
		return errors.WithStack(err)
	}
	if err = tx.Insert(&models.AgentService{AgentID: agent.ID, ServiceID: service.ID}); err != nil {
		return errors.WithStack(err)
	}

	// check connection
	dsn := agent.DSN(service)
	db, err := sql.Open("mysql", dsn)
	if err == nil {
		err = db.PingContext(ctx)
		db.Close()
	}
	if err != nil {
		if err, ok := err.(*mysql.MySQLError); ok {
			switch err.Number {
			case 0x414: // 1044
				return status.Error(codes.PermissionDenied, err.Message)
			case 0x415: // 1045
				return status.Error(codes.Unauthenticated, err.Message)
			}
		}
		return errors.WithStack(err)
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
			Environment: []string{fmt.Sprintf("DATA_SOURCE_NAME=%s", dsn)},
		}
		if err = svc.Supervisor.Start(ctx, cfg); err != nil {
			return err
		}
	}

	return nil
}

func (svc *Service) updateRDSExporterConfig(tx *reform.TX, service *models.RDSService) (*rdsExporterConfig, error) {
	// collect all RDS nodes
	var config rdsExporterConfig
	nodes, err := tx.FindAllFrom(models.RDSNodeTable, "type", models.RDSNodeType)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for _, n := range nodes {
		node := n.(*models.RDSNode)
		instance := rdsExporterInstance{
			Region:       node.Region,
			Instance:     node.Name,
			Type:         unknown,
			AWSAccessKey: service.AWSAccessKey,
			AWSSecretKey: service.AWSSecretKey,
		}
		switch *service.Engine {
		case "aurora":
			instance.Type = auroraMySQL
		case "mysql":
			instance.Type = mySQL
		}
		config.Instances = append(config.Instances, instance)
	}
	sort.Slice(config.Instances, func(i, j int) bool {
		if config.Instances[i].Region != config.Instances[j].Region {
			return config.Instances[i].Region < config.Instances[j].Region
		}
		return config.Instances[i].Instance < config.Instances[j].Instance
	})

	// update rds_exporter configuration
	b, err := config.Marshal()
	if err != nil {
		return nil, err
	}
	if err = ioutil.WriteFile(svc.RDSExporterConfigPath, b, 0666); err != nil {
		return nil, errors.WithStack(err)
	}

	return &config, nil
}

func (svc *Service) rdsExporterServiceConfig(serviceName string) *servicelib.Config {
	return &servicelib.Config{
		Name:        serviceName,
		DisplayName: serviceName,
		Description: serviceName,
		Executable:  svc.RDSExporterPath,
		Arguments: []string{
			fmt.Sprintf("-config.file=%s", svc.RDSExporterConfigPath),
			fmt.Sprintf("-web.listen-address=127.0.0.1:%d", rdsExporterPort),
		},
	}
}

func (svc *Service) addRDSExporter(ctx context.Context, tx *reform.TX, service *models.RDSService, node *models.RDSNode) error {
	// insert rds_exporter agent and associations
	agent := &models.RDSExporter{
		Type:         models.RDSExporterAgentType,
		RunsOnNodeID: svc.pmmServerNode.ID,

		ListenPort: pointer.ToUint16(rdsExporterPort),
	}
	var err error
	if err = tx.Insert(agent); err != nil {
		return errors.WithStack(err)
	}
	if err = tx.Insert(&models.AgentService{AgentID: agent.ID, ServiceID: service.ID}); err != nil {
		return errors.WithStack(err)
	}
	if err = tx.Insert(&models.AgentNode{AgentID: agent.ID, NodeID: node.ID}); err != nil {
		return errors.WithStack(err)
	}

	if svc.RDSExporterPath != "" {
		// update rds_exporter configuration
		if _, err = svc.updateRDSExporterConfig(tx, service); err != nil {
			return err
		}

		// restart rds_exporter
		name := agent.NameForSupervisor()
		if err = svc.Supervisor.Stop(ctx, name); err != nil {
			logger.Get(ctx).Warn(err)
		}
		if err = svc.Supervisor.Start(ctx, svc.rdsExporterServiceConfig(name)); err != nil {
			return err
		}
	}

	return nil
}

func (svc *Service) addQanAgent(ctx context.Context, tx *reform.TX, service *models.RDSService, node *models.RDSNode, username, password string) error {
	// Despite running a single qan-agent process on PMM Server, we use one database record per MySQL instance
	// to store username/password and UUID.

	// insert qan-agent agent and association
	agent := &models.QanAgent{
		Type:         models.QanAgentAgentType,
		RunsOnNodeID: svc.pmmServerNode.ID,

		ServiceUsername: &username,
		ServicePassword: &password,
		ListenPort:      pointer.ToUint16(qanAgentPort),
	}
	var err error
	if err = tx.Insert(agent); err != nil {
		return errors.WithStack(err)
	}
	if err = tx.Insert(&models.AgentService{AgentID: agent.ID, ServiceID: service.ID}); err != nil {
		return errors.WithStack(err)
	}

	// DSNs for mysqld_exporter and qan-agent are currently identical,
	// so we do not check connection again

	// start or reconfigure qan-agent
	if svc.QAN != nil {
		if err = svc.QAN.AddMySQL(ctx, node, service, agent); err != nil {
			return err
		}

		// re-save agent with set QANDBInstanceUUID
		if err = tx.Save(agent); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
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
		if err = tx.Insert(node); err != nil {
			if err, ok := err.(*mysql.MySQLError); ok && err.Number == 0x426 {
				return status.Errorf(codes.AlreadyExists, "RDS instance %q already exists in region %q.",
					node.Name, node.Region)
			}
			return errors.WithStack(err)
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
		if err = tx.Insert(service); err != nil {
			return errors.WithStack(err)
		}

		if err = svc.addMySQLdExporter(ctx, tx, service, username, password); err != nil {
			return err
		}
		if err = svc.addRDSExporter(ctx, tx, service, node); err != nil {
			return err
		}
		if err = svc.addQanAgent(ctx, tx, service, node, username, password); err != nil {
			return err
		}

		return svc.ApplyPrometheusConfiguration(ctx, tx.Querier)
	})
}

func (svc *Service) Remove(ctx context.Context, id *InstanceID) error {
	if id.Name == "" {
		return status.Error(codes.InvalidArgument, "RDS instance name is not given.")
	}
	if id.Region == "" {
		return status.Error(codes.InvalidArgument, "RDS instance region is not given.")
	}

	var err error
	return svc.DB.InTransaction(func(tx *reform.TX) error {
		var node models.RDSNode
		if err = tx.SelectOneTo(&node, "WHERE type = ? AND name = ? AND region = ?", models.RDSNodeType, id.Name, id.Region); err != nil {
			if err == reform.ErrNoRows {
				return status.Errorf(codes.NotFound, "RDS instance %q not found in region %q.", id.Name, id.Region)
			}
			return errors.WithStack(err)
		}

		var service models.RDSService
		if err = tx.SelectOneTo(&service, "WHERE node_id = ?", node.ID); err != nil {
			return errors.WithStack(err)
		}

		// remove associations of the service and agents
		agentsForService, err := models.AgentsForServiceID(tx.Querier, service.ID)
		if err != nil {
			return err
		}
		for _, agent := range agentsForService {
			var deleted uint
			deleted, err = tx.DeleteFrom(models.AgentServiceView, "WHERE service_id = ? AND agent_id = ?", service.ID, agent.ID)
			if err != nil {
				return errors.WithStack(err)
			}
			if deleted != 1 {
				return errors.Errorf("expected to delete 1 record, deleted %d", deleted)
			}
		}

		// remove associations of the node and agents
		agentsForNode, err := models.AgentsForNodeID(tx.Querier, node.ID)
		if err != nil {
			return err
		}
		for _, agent := range agentsForNode {
			var deleted uint
			deleted, err = tx.DeleteFrom(models.AgentNodeView, "WHERE node_id = ? AND agent_id = ?", node.ID, agent.ID)
			if err != nil {
				return errors.WithStack(err)
			}
			if deleted != 1 {
				return errors.Errorf("expected to delete 1 record, deleted %d", deleted)
			}
		}

		// stop agents
		agents := make(map[int32]models.Agent, len(agentsForService)+len(agentsForNode))
		for _, agent := range agentsForService {
			agents[agent.ID] = agent
		}
		for _, agent := range agentsForNode {
			agents[agent.ID] = agent
		}
		for _, agent := range agents {
			switch agent.Type {
			case models.MySQLdExporterAgentType:
				a := models.MySQLdExporter{ID: agent.ID}
				if err = tx.Reload(&a); err != nil {
					return errors.WithStack(err)
				}
				if svc.MySQLdExporterPath != "" {
					if err = svc.Supervisor.Stop(ctx, a.NameForSupervisor()); err != nil {
						return err
					}
				}

			case models.RDSExporterAgentType:
				a := models.RDSExporter{ID: agent.ID}
				if err = tx.Reload(&a); err != nil {
					return errors.WithStack(err)
				}
				if svc.RDSExporterPath != "" {
					// update rds_exporter configuration
					config, err := svc.updateRDSExporterConfig(tx, &service)
					if err != nil {
						return err
					}

					// stop or restart rds_exporter
					name := a.NameForSupervisor()
					if err = svc.Supervisor.Stop(ctx, name); err != nil {
						return err
					}
					if len(config.Instances) > 0 {
						if err = svc.Supervisor.Start(ctx, svc.rdsExporterServiceConfig(name)); err != nil {
							return err
						}
					}
				}

			case models.QanAgentAgentType:
				a := models.QanAgent{ID: agent.ID}
				if err = tx.Reload(&a); err != nil {
					return errors.WithStack(err)
				}
				if svc.QAN != nil {
					if err = svc.QAN.RemoveMySQL(ctx, &a); err != nil {
						return err
					}
				}
			}
		}

		// remove agents
		for _, agent := range agents {
			if err = tx.Delete(&agent); err != nil {
				return errors.WithStack(err)
			}
		}

		if err = tx.Delete(&service); err != nil {
			return errors.WithStack(err)
		}
		if err = tx.Delete(&node); err != nil {
			return errors.WithStack(err)
		}

		return svc.ApplyPrometheusConfiguration(ctx, tx.Querier)
	})
}
