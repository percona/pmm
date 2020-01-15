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

package models

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

func checkServiceUniqueID(q *reform.Querier, id string) error {
	if id == "" {
		panic("empty Service ID")
	}

	row := &Service{ServiceID: id}
	switch err := q.Reload(row); err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Service with ID %q already exists.", id)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

func checkServiceUniqueName(q *reform.Querier, name string) error {
	_, err := q.FindOneFrom(ServiceTable, "service_name", name)
	switch err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Service with name %q already exists.", name)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

// FindAllServices returns all Services.
func FindAllServices(q *reform.Querier) ([]*Service, error) {
	structs, err := q.SelectAllFrom(ServiceTable, "ORDER BY service_id")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	services := make([]*Service, len(structs))
	for i, s := range structs {
		services[i] = s.(*Service)
	}

	return services, nil
}

// FindServiceByID finds Service by ID.
func FindServiceByID(q *reform.Querier, id string) (*Service, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Service ID.")
	}

	row := &Service{ServiceID: id}
	switch err := q.Reload(row); err {
	case nil:
		return row, nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "Service with ID %q not found.", id)
	default:
		return nil, errors.WithStack(err)
	}
}

// FindServicesByIDs finds Services by IDs.
func FindServicesByIDs(q *reform.Querier, ids []string) ([]*Service, error) {
	if len(ids) == 0 {
		return []*Service{}, nil
	}

	p := strings.Join(q.Placeholders(1, len(ids)), ", ")                   //nolint:gomnd
	tail := fmt.Sprintf("WHERE service_id IN (%s) ORDER BY service_id", p) //nolint:gosec
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	structs, err := q.SelectAllFrom(ServiceTable, tail, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]*Service, len(structs))
	for i, s := range structs {
		res[i] = s.(*Service)
	}
	return res, nil
}

// FindServiceByName finds Service by Name.
func FindServiceByName(q *reform.Querier, name string) (*Service, error) {
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Service Name.")
	}

	service := new(Service)
	err := q.FindOneTo(service, "service_name", name)
	switch err {
	case nil:
		return service, nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "Service with name %q not found.", name)
	default:
		return nil, errors.WithStack(err)
	}
}

// ServicesForNode returns all Services for Node with given ID.
func ServicesForNode(q *reform.Querier, nodeID string) ([]*Service, error) {
	tail := fmt.Sprintf("WHERE node_id = %s ORDER BY service_id", q.Placeholder(1)) //nolint:gosec
	structs, err := q.SelectAllFrom(ServiceTable, tail, nodeID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Services")
	}

	res := make([]*Service, len(structs))
	for i, s := range structs {
		res[i] = s.(*Service)
	}
	return res, nil
}

// AddDBMSServiceParams contains parameters for adding DBMS (MySQL, PostgreSQL, MongoDB) Services.
type AddDBMSServiceParams struct {
	ServiceName    string
	NodeID         string
	Environment    string
	Cluster        string
	ReplicationSet string
	CustomLabels   map[string]string
	Address        *string
	Port           *uint16
}

// AddNewService adds new service to storage.
func AddNewService(q *reform.Querier, serviceType ServiceType, params *AddDBMSServiceParams) (*Service, error) {
	id := "/service_id/" + uuid.New().String()
	if err := checkServiceUniqueID(q, id); err != nil {
		return nil, err
	}
	if err := checkServiceUniqueName(q, params.ServiceName); err != nil {
		return nil, err
	}

	if _, err := FindNodeByID(q, params.NodeID); err != nil {
		return nil, err
	}

	row := &Service{
		ServiceID:      id,
		ServiceType:    serviceType,
		ServiceName:    params.ServiceName,
		NodeID:         params.NodeID,
		Environment:    params.Environment,
		Cluster:        params.Cluster,
		ReplicationSet: params.ReplicationSet,
		Address:        params.Address,
		Port:           params.Port,
	}
	if err := row.SetCustomLabels(params.CustomLabels); err != nil {
		return nil, err
	}
	if err := q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	return row, nil
}

// RemoveService removes single Service.
func RemoveService(q *reform.Querier, id string, mode RemoveMode) error {
	s, err := FindServiceByID(q, id)
	if err != nil {
		return err
	}
	// check/remove Agents
	structs, err := q.FindAllFrom(AgentTable, "service_id", id)
	if err != nil {
		return errors.Wrap(err, "failed to select Agent IDs")
	}
	if len(structs) != 0 {
		switch mode {
		case RemoveRestrict:
			return status.Errorf(codes.FailedPrecondition, "Service with ID %q has agents.", id)
		case RemoveCascade:
			for _, str := range structs {
				agentID := str.(*Agent).AgentID
				if _, err = RemoveAgent(q, agentID, RemoveCascade); err != nil {
					return err
				}
			}
		default:
			panic(fmt.Errorf("unhandled RemoveMode %v", mode))
		}
	}

	return errors.Wrap(q.Delete(s), "failed to delete Service")
}
