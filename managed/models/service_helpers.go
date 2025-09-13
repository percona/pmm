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
)

// ErrInvalidServiceType is returned when unsupported service type is provided.
var ErrInvalidServiceType = errors.New("provided service type not defined")

func checkServiceUniqueID(q *reform.Querier, id string) error {
	if id == "" {
		panic("empty Service ID")
	}

	row := &Service{ServiceID: id}
	err := q.Reload(row)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return errors.WithStack(err)
	}

	return status.Errorf(codes.AlreadyExists, "Service with ID %q already exists.", id)
}

func checkServiceUniqueName(q *reform.Querier, name string) error {
	_, err := q.FindOneFrom(ServiceTable, "service_name", name)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return errors.WithStack(err)
	}

	return status.Errorf(codes.AlreadyExists, "Service with name %q already exists.", name)
}

func validateDBConnectionOptions(socket, host *string, port *uint16) error {
	if host == nil && socket == nil {
		return status.Error(codes.InvalidArgument, "Neither socket nor address passed.")
	}

	if host != nil {
		if socket != nil {
			return status.Error(codes.InvalidArgument, "Socket and address cannot be specified together.")
		}

		if port == nil {
			return status.Errorf(codes.InvalidArgument, "Port is expected to be passed along with the host address.")
		}
	}

	if socket != nil && port != nil {
		return status.Error(codes.InvalidArgument, "Socket and port cannot be specified together.")
	}
	return nil
}

// ServiceFilters represents filters for services list.
type ServiceFilters struct {
	// Return only Services runs on that Node.
	NodeID string
	// Return only Services with provided type.
	ServiceType *ServiceType
	// Return only Services with given external group.
	ExternalGroup string
	// Return only Services in the given cluster
	Cluster string
}

// FindServices returns Services by filters.
func FindServices(q *reform.Querier, filters ServiceFilters) ([]*Service, error) {
	var conditions []string
	var args []interface{}
	idx := 1
	if filters.NodeID != "" {
		conditions = append(conditions, fmt.Sprintf("node_id = %s", q.Placeholder(idx)))
		args = append(args, filters.NodeID)
		idx++
	}
	if filters.ExternalGroup != "" {
		conditions = append(conditions, fmt.Sprintf("external_group = %s", q.Placeholder(idx)))
		args = append(args, filters.ExternalGroup)
		idx++
	}
	if filters.ServiceType != nil {
		conditions = append(conditions, fmt.Sprintf("service_type = %s", q.Placeholder(idx)))
		args = append(args, filters.ServiceType)
		idx++
	}
	if filters.Cluster != "" {
		conditions = append(conditions, fmt.Sprintf("cluster = %s", q.Placeholder(idx)))
		args = append(args, filters.Cluster)
	}
	var whereClause string
	if len(conditions) != 0 {
		whereClause = fmt.Sprintf("WHERE %s", strings.Join(conditions, " AND "))
	}
	structs, err := q.SelectAllFrom(ServiceTable, fmt.Sprintf("%s ORDER BY service_id", whereClause), args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	services := make([]*Service, len(structs))
	for i, s := range structs {
		services[i] = s.(*Service) //nolint:forcetypeassert
	}

	return services, nil
}

// FindActiveServiceTypes returns all active Service Types.
func FindActiveServiceTypes(q *reform.Querier) ([]ServiceType, error) {
	query := fmt.Sprintf(`SELECT DISTINCT service_type FROM %s`, ServiceTable.s.SQLName)
	rows, err := q.Query(query)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer func() {
		if rowsErr := rows.Close(); rowsErr != nil {
			logrus.Debug(rowsErr)
		}
	}()

	var res []ServiceType
	for rows.Next() {
		var serviceType ServiceType
		if err = rows.Scan(&serviceType); err != nil {
			return nil, err
		}

		res = append(res, serviceType)
	}

	return res, nil
}

// FindServiceByID searches Service by ID.
func FindServiceByID(q *reform.Querier, id string) (*Service, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Service ID.")
	}

	row := &Service{ServiceID: id}
	err := q.Reload(row)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Service with ID %q not found.", id)
		}
		return nil, errors.WithStack(err)
	}

	return row, nil
}

// FindServicesByIDs finds Services by IDs.
func FindServicesByIDs(q *reform.Querier, ids []string) (map[string]*Service, error) {
	if len(ids) == 0 {
		return make(map[string]*Service), nil
	}

	p := strings.Join(q.Placeholders(1, len(ids)), ", ")
	tail := fmt.Sprintf("WHERE service_id IN (%s) ORDER BY service_id", p)
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	all, err := q.SelectAllFrom(ServiceTable, tail, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	services := make(map[string]*Service, len(all))
	for _, s := range all {
		service := s.(*Service) //nolint:forcetypeassert
		services[service.ServiceID] = service
	}

	return services, nil
}

// FindServiceByName finds Service by Name.
func FindServiceByName(q *reform.Querier, name string) (*Service, error) {
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Service Name.")
	}

	var service Service
	err := q.FindOneTo(&service, "service_name", name)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Service with name %q not found.", name)
		}
		return nil, errors.WithStack(err)
	}

	return &service, nil
}

// AddDBMSServiceParams contains parameters for adding DBMS (MySQL, PostgreSQL, MongoDB, External, and Valkey) Services.
type AddDBMSServiceParams struct {
	ServiceName    string
	NodeID         string
	Environment    string
	Cluster        string
	ReplicationSet string
	CustomLabels   map[string]string
	ExternalGroup  string
	Database       string
	Address        *string
	Port           *uint16
	Socket         *string
}

// AddNewService adds new service to storage.
// Must be performed in transaction.
func AddNewService(q *reform.Querier, serviceType ServiceType, params *AddDBMSServiceParams) (*Service, error) {
	switch serviceType {
	case MySQLServiceType, MongoDBServiceType, PostgreSQLServiceType, ProxySQLServiceType, ValkeyServiceType:
		if err := validateDBConnectionOptions(params.Socket, params.Address, params.Port); err != nil {
			return nil, err
		}
		if params.ExternalGroup != "" {
			return nil, status.Errorf(codes.InvalidArgument, "The external group is not allowed for service type: %q.", serviceType)
		}
	case HAProxyServiceType:
		if params.ExternalGroup != "" {
			return nil, status.Errorf(codes.InvalidArgument, "The external group is not allowed for service type: %q.", serviceType)
		}
	case ExternalServiceType:
		if params.ExternalGroup == "" {
			// Set default value for backward compatibility with an old pmm-admin.
			params.ExternalGroup = "external"
		}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Unknown service type: %q.", serviceType)
	}

	id := uuid.New().String()
	if err := checkServiceUniqueID(q, id); err != nil {
		return nil, err
	}
	if err := checkServiceUniqueName(q, params.ServiceName); err != nil {
		return nil, err
	}

	if _, err := FindNodeByID(q, params.NodeID); err != nil {
		return nil, err
	}

	databaseName := ""
	if serviceType == PostgreSQLServiceType {
		if params.Database == "" {
			databaseName = "postgres"
		} else {
			databaseName = params.Database
		}
	}
	row := &Service{
		ServiceID:      id,
		ServiceType:    serviceType,
		ServiceName:    params.ServiceName,
		DatabaseName:   databaseName,
		NodeID:         params.NodeID,
		Environment:    params.Environment,
		Cluster:        params.Cluster,
		ReplicationSet: params.ReplicationSet,
		Address:        params.Address,
		Port:           params.Port,
		Socket:         params.Socket,
		ExternalGroup:  params.ExternalGroup,
	}
	if err := row.SetCustomLabels(params.CustomLabels); err != nil {
		return nil, err
	}
	if err := q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	if err := initSoftwareVersions(q, id, serviceType); err != nil {
		return nil, err
	}

	return row, nil
}

// RemoveService removes single Service.
// If associated service software versions entry exists it is removed by the ON DELETE CASCADE option.
func RemoveService(q *reform.Querier, id string, mode RemoveMode) error {
	s, err := FindServiceByID(q, id)
	if err != nil {
		return err
	}
	// find agents and artifacts
	agents, err := FindAgents(q, AgentFilters{ServiceID: id})
	if err != nil {
		return errors.Wrap(err, "failed to select Agent IDs")
	}

	artifacts, err := FindArtifacts(q, ArtifactFilters{ServiceID: id})
	if err != nil {
		return errors.Wrap(err, "failed to select artifacts")
	}

	restoreItems, err := FindRestoreHistoryItems(q, RestoreHistoryItemFilters{ServiceID: id})
	if err != nil {
		return errors.Wrap(err, "failed to select restore history items")
	}

	tasks, err := FindScheduledTasks(q, ScheduledTasksFilter{ServiceID: id})
	if err != nil {
		return errors.Wrap(err, "failed to select scheduled tasks")
	}

	switch mode {
	case RemoveRestrict:
		if len(agents) != 0 {
			return status.Errorf(codes.FailedPrecondition, "Service with ID %q has agents.", id)
		}

		if len(artifacts) != 0 {
			return status.Errorf(codes.FailedPrecondition, "Service with ID %q has artifacts.", id)
		}

		if len(restoreItems) != 0 {
			return status.Errorf(codes.FailedPrecondition, "Service with ID %q has restore history items.", id)
		}

		if len(tasks) != 0 {
			return status.Errorf(codes.FailedPrecondition, "Service with ID %q has scheduled tasks.", id)
		}
	case RemoveCascade:
		for _, a := range agents {
			if _, err := RemoveAgent(q, a.AgentID, RemoveCascade); err != nil {
				return err
			}
		}
		for _, a := range artifacts {
			if _, err := UpdateArtifact(q, a.ID, UpdateArtifactParams{
				ServiceID: pointer.ToString(""),
			}); err != nil {
				return err
			}
		}
		for _, i := range restoreItems {
			if err := RemoveRestoreHistoryItem(q, i.ID); err != nil {
				return err
			}
		}
		for _, t := range tasks {
			if err := RemoveScheduledTask(q, t.ID); err != nil {
				return err
			}
		}
	default:
		panic(fmt.Errorf("unhandled RemoveMode %v", mode))
	}

	return errors.Wrap(q.Delete(s), "failed to delete Service")
}

// ValidateServiceType checks argument value is in the list of supported types.
func ValidateServiceType(serviceType ServiceType) error {
	switch serviceType {
	case MySQLServiceType,
		MongoDBServiceType,
		PostgreSQLServiceType,
		ProxySQLServiceType,
		HAProxyServiceType,
		ExternalServiceType:
		return nil
	default:
		return errors.Wrapf(ErrInvalidServiceType, "unknown service type '%s'", string(serviceType))
	}
}

// ChangeStandardLabelsParams contains parameters for changing standard labels for a service.
type ChangeStandardLabelsParams struct {
	ServiceID      string
	Cluster        *string
	Environment    *string
	ReplicationSet *string
	ExternalGroup  *string
}

// ChangeStandardLabels changes standard labels for a service.
func ChangeStandardLabels(q *reform.Querier, serviceID string, labels ServiceStandardLabelsParams) error {
	s, err := FindServiceByID(q, serviceID)
	if err != nil {
		return err
	}

	columns := []string{}

	if labels.Cluster != nil {
		columns = append(columns, "cluster")
		s.Cluster = *labels.Cluster
	}

	if labels.Environment != nil {
		columns = append(columns, "environment")
		s.Environment = *labels.Environment
	}

	if labels.ReplicationSet != nil {
		columns = append(columns, "replication_set")
		s.ReplicationSet = *labels.ReplicationSet
	}

	if labels.ExternalGroup != nil {
		columns = append(columns, "external_group")
		s.ExternalGroup = *labels.ExternalGroup
	}

	// to avoid "reform: nothing to update" error
	if len(columns) == 0 {
		return nil
	}

	if err = q.UpdateColumns(s, columns...); err != nil {
		return err
	}

	return nil
}

func initSoftwareVersions(q *reform.Querier, serviceID string, serviceType ServiceType) error {
	switch serviceType {
	case MySQLServiceType:
		fallthrough
	case MongoDBServiceType:
		if _, err := CreateServiceSoftwareVersions(q, CreateServiceSoftwareVersionsParams{
			ServiceID:        serviceID,
			ServiceType:      serviceType,
			SoftwareVersions: []SoftwareVersion{},
			NextCheckAt:      time.Now(),
		}); err != nil {
			return errors.Wrapf(err, "couldn't initialize software versions for service %s", serviceID)
		}
	default:
		return nil
	}
	return nil
}
