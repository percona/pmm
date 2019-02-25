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

// Package mysql contains business logic of working with Remote MySQL instances.
package mysql

/*
import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/go-sql-driver/mysql"
	servicelib "github.com/percona/kardianos-service"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services"
	"github.com/percona/pmm-managed/services/inventory"
	"github.com/percona/pmm-managed/services/prometheus"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/ports"
)

const (
	defaultMySQLPort uint32 = 3306

	// maximum time for connecting to the database and running all queries
	sqlCheckTimeout = 5 * time.Second
)

var versionRegexp = regexp.MustCompile(`([\d\.]+)-.*`)

type ServiceConfig struct {
	MySQLdExporterPath string

	Prometheus    *prometheus.Service
	Supervisor    services.Supervisor
	DB            *reform.DB
	PortsRegistry *ports.Registry
}

// Service is responsible for interactions with AWS RDS.
type Service struct {
	*ServiceConfig
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
		pmmServerNode: &node,
	}
	return svc, nil
}

type Instance struct {
	Node    models.RemoteNode
	Service models.MySQLService
}

func (svc *Service) ApplyPrometheusConfiguration(ctx context.Context, q *reform.Querier) error {
	mySQLHR := &prometheus.ScrapeConfig{
		JobName:        "remote-mysql-hr",
		ScrapeInterval: "1s",
		ScrapeTimeout:  "1s",
		MetricsPath:    "/metrics-hr",
		RelabelConfigs: []prometheus.RelabelConfig{{
			TargetLabel: "job",
			Replacement: "mysql",
		}},
	}
	mySQLMR := &prometheus.ScrapeConfig{
		JobName:        "remote-mysql-mr",
		ScrapeInterval: "5s",
		ScrapeTimeout:  "1s",
		MetricsPath:    "/metrics-mr",
		RelabelConfigs: []prometheus.RelabelConfig{{
			TargetLabel: "job",
			Replacement: "mysql",
		}},
	}
	mySQLLR := &prometheus.ScrapeConfig{
		JobName:        "remote-mysql-lr",
		ScrapeInterval: "60s",
		ScrapeTimeout:  "5s",
		MetricsPath:    "/metrics-lr",
		RelabelConfigs: []prometheus.RelabelConfig{{
			TargetLabel: "job",
			Replacement: "mysql",
		}},
	}

	nodes, err := q.FindAllFrom(models.RemoteNodeTable, "type", models.RemoteNodeType)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, n := range nodes {
		node := n.(*models.RemoteNode)

		mySQLServices, err := q.SelectAllFrom(models.MySQLServiceTable, "WHERE node_id = ?", node.ID)
		if err != nil {
			return errors.WithStack(err)
		}
		if len(mySQLServices) != 1 {
			return errors.Errorf("expected to fetch 1 record, fetched %d. %v", len(mySQLServices), mySQLServices)
		}
		service := mySQLServices[0].(*models.MySQLService)
		if service.Type != models.MySQLServiceType {
			continue
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
				logger.Get(ctx).WithField("component", "mysql").Infof("%s %s %s %d", a.Type, node.Name, *node.Region, *a.ListenPort)

				sc := prometheus.StaticConfig{
					Targets: []string{fmt.Sprintf("127.0.0.1:%d", *a.ListenPort)},
					Labels: []prometheus.LabelPair{
						{Name: "instance", Value: node.Name},
					},
				}
				mySQLHR.StaticConfigs = append(mySQLHR.StaticConfigs, sc)
				mySQLMR.StaticConfigs = append(mySQLMR.StaticConfigs, sc)
				mySQLLR.StaticConfigs = append(mySQLLR.StaticConfigs, sc)
			}
		}
	}

	// sort by instance
	sorterFor := func(sc []prometheus.StaticConfig) func(int, int) bool {
		return func(i, j int) bool {
			return sc[i].Labels[0].Value < sc[j].Labels[0].Value
		}
	}
	sort.Slice(mySQLHR.StaticConfigs, sorterFor(mySQLHR.StaticConfigs))
	sort.Slice(mySQLMR.StaticConfigs, sorterFor(mySQLMR.StaticConfigs))
	sort.Slice(mySQLLR.StaticConfigs, sorterFor(mySQLLR.StaticConfigs))

	return svc.Prometheus.SetScrapeConfigs(ctx, false, mySQLHR, mySQLMR, mySQLLR)
}

func (svc *Service) List(ctx context.Context) ([]Instance, error) {
	var res []Instance
	err := svc.DB.InTransaction(func(tx *reform.TX) error {
		structs, e := tx.SelectAllFrom(models.RemoteNodeTable, "WHERE type = ? ORDER BY id", models.RemoteNodeType)
		if e != nil {
			return e
		}
		nodes := make([]models.RemoteNode, len(structs))
		for i, str := range structs {
			nodes[i] = *str.(*models.RemoteNode)
		}

		structs, e = tx.SelectAllFrom(models.MySQLServiceTable, "WHERE type = ? ORDER BY id", models.MySQLServiceType)
		if e != nil {
			return e
		}
		services := make([]models.MySQLService, len(structs))
		for i, str := range structs {
			services[i] = *str.(*models.MySQLService)
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

func (svc *Service) addMySQLdExporter(ctx context.Context, tx *reform.TX, service *models.MySQLService, username, password string) error {
	// insert mysqld_exporter agent and association
	port, err := svc.PortsRegistry.Reserve()
	if err != nil {
		return err
	}
	agent := &models.MySQLdExporter{
		ID:           inventory.MakeID(),
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

	// check connection and a number of tables
	var tableCount int
	dsn := agent.DSN(service)
	db, err := sql.Open("mysql", dsn)
	if err == nil {
		sqlCtx, cancel := context.WithTimeout(ctx, sqlCheckTimeout)
		err = db.QueryRowContext(sqlCtx, "SELECT COUNT(*) FROM information_schema.tables").Scan(&tableCount)
		cancel()
		db.Close()
		agent.MySQLDisableTablestats = pointer.ToBool(tableCount > 1000)
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
		cfg := svc.mysqlExporterCfg(agent, dsn)
		if err = svc.Supervisor.Start(ctx, cfg); err != nil {
			return err
		}
	}

	return nil
}

func (svc *Service) mysqlExporterCfg(agent *models.MySQLdExporter, dsn string) *servicelib.Config {
	name := models.NameForSupervisor(agent.Type, *agent.ListenPort)

	arguments := []string{
		"-collect.binlog_size",
		"-collect.global_status",
		"-collect.global_variables",
		"-collect.info_schema.innodb_metrics",
		"-collect.info_schema.processlist",
		"-collect.info_schema.query_response_time",
		"-collect.info_schema.userstats",
		"-collect.perf_schema.eventswaits",
		"-collect.perf_schema.file_events",
		"-collect.slave_status",
		fmt.Sprintf("-web.listen-address=127.0.0.1:%d", *agent.ListenPort),
	}
	if agent.MySQLDisableTablestats == nil || !*agent.MySQLDisableTablestats {
		// enable tablestats and a few related collectors just like pmm-admin
		// https://github.com/percona/pmm-client/blob/e94b61ed0e5482a27039f0d1b0b36076731f0c29/pmm/plugin/mysql/metrics/metrics.go#L98-L105
		arguments = append(arguments, "-collect.auto_increment.columns")
		arguments = append(arguments, "-collect.info_schema.tables")
		arguments = append(arguments, "-collect.info_schema.tablestats")
		arguments = append(arguments, "-collect.perf_schema.indexiowaits")
		arguments = append(arguments, "-collect.perf_schema.tableiowaits")
		arguments = append(arguments, "-collect.perf_schema.tablelocks")
	}
	sort.Strings(arguments)

	return &servicelib.Config{
		Name:        name,
		DisplayName: name,
		Description: name,
		Executable:  svc.MySQLdExporterPath,
		Arguments:   arguments,
		Environment: []string{fmt.Sprintf("DATA_SOURCE_NAME=%s", dsn)},
	}
}

func (svc *Service) Add(ctx context.Context, name, address string, port uint32, username, password string) (string, error) {
	address = strings.TrimSpace(address)
	username = strings.TrimSpace(username)
	name = strings.TrimSpace(name)
	if address == "" {
		return "", status.Error(codes.InvalidArgument, "MySQL instance host is not given.")
	}
	if username == "" {
		return "", status.Error(codes.InvalidArgument, "Username is not given.")
	}
	if port == 0 {
		port = defaultMySQLPort
	}
	if name == "" {
		name = address
	}

	var id string
	err := svc.DB.InTransaction(func(tx *reform.TX) error {
		// insert node
		node := &models.RemoteNode{
			ID:     inventory.MakeID(),
			Type:   models.RemoteNodeType,
			Name:   name,
			Region: pointer.ToString(models.RemoteNodeRegion),
		}
		if err := tx.Insert(node); err != nil {
			if err, ok := err.(*mysql.MySQLError); ok && err.Number == 0x426 {
				return status.Errorf(codes.AlreadyExists, "MySQL instance %q already exists.",
					node.Name)
			}
			return errors.WithStack(err)
		}
		id = node.ID

		engine, engineVersion, err := svc.engineAndEngineVersion(ctx, address, port, username, password)
		if err != nil {
			return errors.WithStack(err)
		}

		// insert service
		service := &models.MySQLService{
			ID:     inventory.MakeID(),
			Type:   models.MySQLServiceType,
			Name:   name,
			NodeID: node.ID,

			Address:       &address,
			Port:          pointer.ToUint16(uint16(port)),
			Engine:        &engine,
			EngineVersion: &engineVersion,
		}
		if err = tx.Insert(service); err != nil {
			return errors.WithStack(err)
		}

		if err = svc.addMySQLdExporter(ctx, tx, service, username, password); err != nil {
			return err
		}

		return svc.ApplyPrometheusConfiguration(ctx, tx.Querier)
	})

	return id, err
}

func (svc *Service) Remove(ctx context.Context, id string) error {
	var err error
	return svc.DB.InTransaction(func(tx *reform.TX) error {
		var node models.RemoteNode
		if err = tx.SelectOneTo(&node, "WHERE type = ? AND id = ?", models.RemoteNodeType, id); err != nil {
			if err == reform.ErrNoRows {
				return status.Errorf(codes.NotFound, "MySQL instance with ID %q not found.", id)
			}
			return errors.WithStack(err)
		}

		var service models.MySQLService
		if err = tx.SelectOneTo(&service, "WHERE node_id = ? and type = ?", node.ID, models.MySQLServiceType); err != nil {
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
		agents := make(map[string]models.Agent, len(agentsForService)+len(agentsForNode))
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
					if err = svc.Supervisor.Stop(ctx, models.NameForSupervisor(a.Type, *a.ListenPort)); err != nil {
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

// Restore configuration from database.
func (svc *Service) Restore(ctx context.Context, tx *reform.TX) error {
	nodes, err := tx.FindAllFrom(models.RemoteNodeTable, "type", models.RemoteNodeType)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, n := range nodes {
		node := n.(*models.RemoteNode)

		mySQLServices, err := tx.SelectAllFrom(models.MySQLServiceTable, "WHERE node_id = ?", node.ID)
		if err != nil {
			return errors.WithStack(err)
		}
		if len(mySQLServices) != 1 {
			return errors.Errorf("expected to fetch 1 record, fetched %d. %v", len(mySQLServices), mySQLServices)
		}
		service := mySQLServices[0].(*models.MySQLService)
		if service.Type != models.MySQLServiceType {
			continue
		}

		agents, err := models.AgentsForServiceID(tx.Querier, service.ID)
		if err != nil {
			return err
		}
		for _, agent := range agents {
			switch agent.Type {
			case models.MySQLdExporterAgentType:
				a := &models.MySQLdExporter{ID: agent.ID}
				if err = tx.Reload(a); err != nil {
					return errors.WithStack(err)
				}
				if svc.MySQLdExporterPath != "" {
					name := models.NameForSupervisor(a.Type, *a.ListenPort)
					err := svc.Supervisor.Status(ctx, name)
					if err == nil {
						if err = svc.Supervisor.Stop(ctx, name); err != nil {
							return err
						}
					}

					dsn := a.DSN(service)
					cfg := svc.mysqlExporterCfg(a, dsn)
					if err = svc.Supervisor.Start(ctx, cfg); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (svc *Service) engineAndEngineVersion(ctx context.Context, host string, port uint32, username string, password string) (string, string, error) {
	var version string
	var versionComment string
	agent := models.MySQLdExporter{
		ServiceUsername: pointer.ToString(username),
		ServicePassword: pointer.ToString(password),
	}
	service := &models.MySQLService{
		Address: &host,
		Port:    pointer.ToUint16(uint16(port)),
	}
	dsn := agent.DSN(service)
	db, err := sql.Open("mysql", dsn)
	if err == nil {
		sqlCtx, cancel := context.WithTimeout(ctx, sqlCheckTimeout)
		err = db.QueryRowContext(sqlCtx, "SELECT @@version, @@version_comment").Scan(&version, &versionComment)
		cancel()
		db.Close()
	}
	if err != nil {
		return "", "", errors.WithStack(err)
	}
	return normalizeEngineAndEngineVersion(versionComment, version)
}

func normalizeEngineAndEngineVersion(engine string, engineVersion string) (string, string, error) {
	if versionRegexp.MatchString(engineVersion) {
		submatch := versionRegexp.FindStringSubmatch(engineVersion)
		engineVersion = submatch[1]
	}

	lowerEngine := strings.ToLower(engine)
	switch {
	case strings.Contains(lowerEngine, "mariadb"):
		return "MariaDB", engineVersion, nil
	case strings.Contains(lowerEngine, "percona"):
		return "Percona Server", engineVersion, nil
	default:
		return "MySQL", engineVersion, nil
	}
}
*/
