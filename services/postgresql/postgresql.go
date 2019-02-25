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

// Package postgresql contains business logic of working with Remote PostgreSQL instances.
package postgresql

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
	"github.com/lib/pq"
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
	defaultPostgreSQLPort uint32 = 5432

	// maximum time for connecting to the database and running all queries
	sqlCheckTimeout = 5 * time.Second
)

// regexps to extract version numbers from the `SELECT version()` output
var (
	postgresDBRegexp  = regexp.MustCompile(`PostgreSQL ([\d\.]+)`)
	cockroachDBRegexp = regexp.MustCompile(`CockroachDB CCL (v[\d\.]+)`)
)

type ServiceConfig struct {
	PostgresExporterPath string

	Prometheus    *prometheus.Service
	Supervisor    services.Supervisor
	DB            *reform.DB
	PortsRegistry *ports.Registry
}

// Service is responsible for interactions with PostgreSQL.
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
		&config.PostgresExporterPath,
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

// ApplyPrometheusConfiguration Adds postgres to prometheus configuration and applies it
func (svc *Service) ApplyPrometheusConfiguration(ctx context.Context, q *reform.Querier) error {
	postgreSQLConfig := &prometheus.ScrapeConfig{
		JobName:        "remote-postgresql",
		ScrapeInterval: "1s",
		ScrapeTimeout:  "1s",
		MetricsPath:    "/metrics",
		RelabelConfigs: []prometheus.RelabelConfig{{
			TargetLabel: "job",
			Replacement: "postgresql",
		}},
	}

	nodes, err := q.FindAllFrom(models.RemoteNodeTable, "type", models.RemoteNodeType)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, n := range nodes {
		node := n.(*models.RemoteNode)

		postgreSQLServices, e := q.SelectAllFrom(models.PostgreSQLServiceTable, "WHERE node_id = ?", node.ID)
		if e != nil {
			return errors.WithStack(e)
		}
		if len(postgreSQLServices) != 1 {
			return errors.Errorf("expected to fetch 1 record, fetched %d. %v", len(postgreSQLServices), postgreSQLServices)
		}
		service := postgreSQLServices[0].(*models.PostgreSQLService)
		if service.Type != models.PostgreSQLServiceType {
			continue
		}

		agents, err := models.AgentsForServiceID(q, service.ID)
		if err != nil {
			return err
		}
		for _, agent := range agents {
			switch agent.Type {
			case models.PostgresExporterAgentType:
				a := models.PostgresExporter{ID: agent.ID}
				if e := q.Reload(&a); e != nil {
					return errors.WithStack(e)
				}
				logger.Get(ctx).WithField("component", "postgresql").Infof("%s %s %d", a.Type, node.Name, *a.ListenPort)

				sc := prometheus.StaticConfig{
					Targets: []string{fmt.Sprintf("127.0.0.1:%d", *a.ListenPort)},
					Labels: []prometheus.LabelPair{
						{Name: "instance", Value: node.Name},
					},
				}
				postgreSQLConfig.StaticConfigs = append(postgreSQLConfig.StaticConfigs, sc)
			}
		}
	}

	// sort by instance
	sorterFor := func(sc []prometheus.StaticConfig) func(int, int) bool {
		return func(i, j int) bool {
			return sc[i].Labels[0].Value < sc[j].Labels[0].Value
		}
	}
	sort.Slice(postgreSQLConfig.StaticConfigs, sorterFor(postgreSQLConfig.StaticConfigs))

	return svc.Prometheus.SetScrapeConfigs(ctx, false, postgreSQLConfig)
}

type Instance struct {
	Node    models.RemoteNode
	Service models.PostgreSQLService
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

		structs, e = tx.SelectAllFrom(models.PostgreSQLServiceTable, "WHERE type = ? ORDER BY id", models.PostgreSQLServiceType)
		if e != nil {
			return e
		}
		services := make([]models.PostgreSQLService, len(structs))
		for i, str := range structs {
			services[i] = *str.(*models.PostgreSQLService)
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

// Add new postgreSQL service and start postgres_exporter
func (svc *Service) Add(ctx context.Context, name, address string, port uint32, username, password string) (string, error) {
	address = strings.TrimSpace(address)
	username = strings.TrimSpace(username)
	name = strings.TrimSpace(name)
	if address == "" {
		return "", status.Error(codes.InvalidArgument, "PostgreSQL instance host is not given.")
	}
	if username == "" {
		return "", status.Error(codes.InvalidArgument, "Username is not given.")
	}
	if port == 0 {
		port = defaultPostgreSQLPort
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
				return status.Errorf(codes.AlreadyExists, "PostgreSQL instance %q already exists.",
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
		service := &models.PostgreSQLService{
			ID:     inventory.MakeID(),
			Type:   models.PostgreSQLServiceType,
			Name:   name,
			NodeID: node.ID,

			Address:       &address,
			Port:          pointer.ToUint16(uint16(port)),
			Engine:        &engine,
			EngineVersion: &engineVersion,
		}
		if err := tx.Insert(service); err != nil {
			return errors.WithStack(err)
		}

		if err := svc.addPostgresExporter(ctx, tx, service, username, password); err != nil {
			return err
		}

		return svc.ApplyPrometheusConfiguration(ctx, tx.Querier)
	})

	return id, err
}

func (svc *Service) engineAndEngineVersion(ctx context.Context, host string, port uint32, username string, password string) (string, string, error) {
	var databaseVersion string
	agent := models.PostgresExporter{
		ServiceUsername: pointer.ToString(username),
		ServicePassword: pointer.ToString(password),
	}
	service := &models.PostgreSQLService{
		Address: &host,
		Port:    pointer.ToUint16(uint16(port)),
	}
	dsn := agent.DSN(service)
	db, err := sql.Open("postgres", dsn)
	if err == nil {
		sqlCtx, cancel := context.WithTimeout(ctx, sqlCheckTimeout)
		err = db.QueryRowContext(sqlCtx, "SELECT version()").Scan(&databaseVersion)
		cancel()
		db.Close()
	}
	if err != nil {
		return "", "", errors.WithStack(err)
	}
	engine, engineVersion := svc.engineAndVersionFromPlainText(databaseVersion)
	return engine, engineVersion, nil
}

func (svc *Service) engineAndVersionFromPlainText(databaseVersion string) (string, string) {
	var engine string
	var engineVersion string
	switch {
	case postgresDBRegexp.MatchString(databaseVersion):
		engine = "PostgreSQL"
		submatch := postgresDBRegexp.FindStringSubmatch(databaseVersion)
		engineVersion = submatch[1]
	case cockroachDBRegexp.MatchString(databaseVersion):
		engine = "CockroachDB"
		submatch := cockroachDBRegexp.FindStringSubmatch(databaseVersion)
		engineVersion = submatch[1]
	}
	return engine, engineVersion
}

// Remove stops postgres_exporter and agent and remove agent from db
func (svc *Service) Remove(ctx context.Context, id string) error {
	var err error
	return svc.DB.InTransaction(func(tx *reform.TX) error {
		var node models.RemoteNode
		if err = tx.SelectOneTo(&node, "WHERE type = ? AND id = ?", models.RemoteNodeType, id); err != nil {
			if err == reform.ErrNoRows {
				return status.Errorf(codes.NotFound, "PostgreSQL instance with ID %q not found.", id)
			}
			return errors.WithStack(err)
		}

		var service models.PostgreSQLService
		if err = tx.SelectOneTo(&service, "WHERE node_id = ? and type = ?", node.ID, models.PostgreSQLServiceType); err != nil {
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
			case models.PostgresExporterAgentType:
				a := models.PostgresExporter{ID: agent.ID}
				if err = tx.Reload(&a); err != nil {
					return errors.WithStack(err)
				}
				if svc.PostgresExporterPath != "" {
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

func (svc *Service) addPostgresExporter(ctx context.Context, tx *reform.TX, service *models.PostgreSQLService, username, password string) error {
	// insert postgres_exporter agent and association
	port, err := svc.PortsRegistry.Reserve()
	if err != nil {
		return err
	}
	agent := &models.PostgresExporter{
		ID:           inventory.MakeID(),
		Type:         models.PostgresExporterAgentType,
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
	db, err := sql.Open("postgres", dsn)
	if err == nil {
		sqlCtx, cancel := context.WithTimeout(ctx, sqlCheckTimeout)
		err = db.QueryRowContext(sqlCtx, "SELECT COUNT(*) FROM information_schema.tables").Scan(&tableCount)
		cancel()
		db.Close()
	}
	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			switch err.Code {
			case "42501":
				return status.Error(codes.PermissionDenied, err.Message)
			case "28P01":
				return status.Error(codes.Unauthenticated, err.Message)
			}
		}
		return errors.WithStack(err)
	}

	// start postgres_exporter agent
	if svc.PostgresExporterPath != "" {
		cfg := svc.postgresExporterCfg(agent, dsn)
		if err = svc.Supervisor.Start(ctx, cfg); err != nil {
			return err
		}
	}

	return nil
}

// Restore configuration from database.
func (svc *Service) Restore(ctx context.Context, tx *reform.TX) error {
	nodes, err := tx.FindAllFrom(models.RemoteNodeTable, "type", models.RemoteNodeType)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, n := range nodes {
		node := n.(*models.RemoteNode)

		postgreSQLServices, e := tx.SelectAllFrom(models.PostgreSQLServiceTable, "WHERE node_id = ?", node.ID)
		if e != nil {
			return errors.WithStack(e)
		}
		if len(postgreSQLServices) != 1 {
			return errors.Errorf("expected to fetch 1 record, fetched %d. %v", len(postgreSQLServices), postgreSQLServices)
		}
		service := postgreSQLServices[0].(*models.PostgreSQLService)
		if service.Type != models.PostgreSQLServiceType {
			continue
		}

		agents, err := models.AgentsForServiceID(tx.Querier, service.ID)
		if err != nil {
			return err
		}
		for _, agent := range agents {
			switch agent.Type {
			case models.PostgresExporterAgentType:
				a := &models.PostgresExporter{ID: agent.ID}
				if err = tx.Reload(a); err != nil {
					return errors.WithStack(err)
				}
				if svc.PostgresExporterPath != "" {
					name := models.NameForSupervisor(a.Type, *a.ListenPort)

					err := svc.Supervisor.Status(ctx, name)
					if err == nil {
						if err = svc.Supervisor.Stop(ctx, name); err != nil {
							return err
						}
					}

					dsn := a.DSN(service)
					cfg := svc.postgresExporterCfg(a, dsn)
					if err = svc.Supervisor.Start(ctx, cfg); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (svc *Service) postgresExporterCfg(agent *models.PostgresExporter, dsn string) *servicelib.Config {
	name := models.NameForSupervisor(agent.Type, *agent.ListenPort)

	arguments := []string{
		fmt.Sprintf("-web.listen-address=127.0.0.1:%d", *agent.ListenPort),
	}
	sort.Strings(arguments)

	return &servicelib.Config{
		Name:        name,
		DisplayName: name,
		Description: name,
		Executable:  svc.PostgresExporterPath,
		Arguments:   arguments,
		Environment: []string{fmt.Sprintf("DATA_SOURCE_NAME=%s", dsn)},
	}
}
*/
