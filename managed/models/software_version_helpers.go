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
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/reform.v1"
)

// SoftwareVersionsOrderBy is a type used for results ordering either by next check time or service id.
type SoftwareVersionsOrderBy int

// Supported ordering of service software versions entries.
const (
	SoftwareVersionsOrderByNextCheckAt SoftwareVersionsOrderBy = iota
	SoftwareVersionsOrderByServiceID
)

// CreateServiceSoftwareVersionsParams are params for creating a new service software versions entry.
type CreateServiceSoftwareVersionsParams struct {
	ServiceID        string
	ServiceType      ServiceType
	SoftwareVersions []SoftwareVersion
	NextCheckAt      time.Time
}

// Validate validates params used for creating a service software versions entry.
func (p *CreateServiceSoftwareVersionsParams) Validate() error {
	if p.ServiceID == "" {
		return NewInvalidArgumentError("service_id shouldn't be empty")
	}

	switch p.ServiceType {
	case MySQLServiceType,
		MongoDBServiceType,
		PostgreSQLServiceType,
		ProxySQLServiceType,
		HAProxyServiceType,
		ExternalServiceType:
	default:
		return NewInvalidArgumentError("invalid service type %q", p.ServiceType)
	}

	for _, sv := range p.SoftwareVersions {
		switch sv.Name {
		case MysqldSoftwareName:
		case XtrabackupSoftwareName:
		case XbcloudSoftwareName:
		case QpressSoftwareName:
		case MongoDBSoftwareName:
		case PBMSoftwareName:
		default:
			return NewInvalidArgumentError("invalid software name %q", sv.Name)
		}

		if sv.Version == "" {
			return NewInvalidArgumentError("empty version for software name %q", sv.Name)
		}
	}

	return nil
}

// CreateServiceSoftwareVersions creates service software versions entry in DB.
func CreateServiceSoftwareVersions(q *reform.Querier, params CreateServiceSoftwareVersionsParams) (*ServiceSoftwareVersions, error) {
	err := params.Validate()
	if err != nil {
		return nil, err
	}

	row := &ServiceSoftwareVersions{
		ServiceID:        params.ServiceID,
		ServiceType:      params.ServiceType,
		SoftwareVersions: params.SoftwareVersions,
		NextCheckAt:      params.NextCheckAt,
	}

	err = q.Insert(row)
	if err != nil {
		return nil, fmt.Errorf("failed to insert service software versions: %w", err)
	}

	return row, nil
}

// UpdateServiceSoftwareVersionsParams represents params for updating service software versions entity.
type UpdateServiceSoftwareVersionsParams struct {
	NextCheckAt      *time.Time
	SoftwareVersions []SoftwareVersion
}

// Validate validates params used for updating a service software versions entry.
func (u *UpdateServiceSoftwareVersionsParams) Validate() error {
	for _, sv := range u.SoftwareVersions {
		switch sv.Name {
		case MysqldSoftwareName:
		case XtrabackupSoftwareName:
		case XbcloudSoftwareName:
		case QpressSoftwareName:
		case MongoDBSoftwareName:
		case PBMSoftwareName:
		default:
			return NewInvalidArgumentError("invalid software name %q", sv.Name)
		}

		if sv.Version == "" {
			return NewInvalidArgumentError("empty version for software name %q", sv.Name)
		}
	}
	return nil
}

// UpdateServiceSoftwareVersions updates existing service software versions.
func UpdateServiceSoftwareVersions(
	q *reform.Querier,
	serviceID string,
	params UpdateServiceSoftwareVersionsParams,
) (*ServiceSoftwareVersions, error) {
	err := params.Validate()
	if err != nil {
		return nil, err
	}

	row, err := FindServiceSoftwareVersionsByServiceID(q, serviceID)
	if err != nil {
		return nil, err
	}

	if params.NextCheckAt != nil {
		row.NextCheckAt = *params.NextCheckAt
	}

	if params.SoftwareVersions != nil {
		row.SoftwareVersions = params.SoftwareVersions
	}

	err = q.Update(row)
	if err != nil {
		return nil, fmt.Errorf("failed to update service software versions: %w", err)
	}

	return row, nil
}

// FindServiceSoftwareVersionsByServiceID returns service software versions entry by given service ID if found,
// returns wrapped ErrNotFound if not found.
func FindServiceSoftwareVersionsByServiceID(q *reform.Querier, serviceID string) (*ServiceSoftwareVersions, error) {
	if serviceID == "" {
		return nil, errors.New("service id is empty")
	}

	versions := &ServiceSoftwareVersions{ServiceID: serviceID}
	err := q.Reload(versions)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, fmt.Errorf("service software versions by service id '%s': %w", serviceID, ErrNotFound)
		}
		return nil, err
	}

	return versions, nil
}

// FindServicesSoftwareVersionsFilter represents a filter for finding service software versions.
type FindServicesSoftwareVersionsFilter struct {
	Limit       *int
	ServiceType *ServiceType
}

// FindServicesSoftwareVersions returns all services software versions sorted by specified param in ascending order
// if limit is not specified, and limited number of entries otherwise.
func FindServicesSoftwareVersions(
	q *reform.Querier,
	filter FindServicesSoftwareVersionsFilter,
	orderBy SoftwareVersionsOrderBy,
) ([]*ServiceSoftwareVersions, error) {
	var args []any
	var tail strings.Builder
	idx := 1

	var err error
	if filter.ServiceType != nil {
		err = ValidateServiceType(*filter.ServiceType)
		if err != nil {
			return nil, err
		}
		_, err = fmt.Fprintf(&tail, "WHERE service_type = %s ", q.Placeholder(idx))
		if err != nil {
			return nil, err
		}
		args = append(args, string(*filter.ServiceType))
		idx++
	}

	if orderBy == SoftwareVersionsOrderByServiceID {
		tail.WriteString("ORDER BY service_id ")
	} else {
		tail.WriteString("ORDER BY next_check_at ")
	}

	if filter.Limit != nil {
		_, err = fmt.Fprintf(&tail, "LIMIT %s", q.Placeholder(idx))
		if err != nil {
			return nil, err
		}
		args = append(args, *filter.Limit)
	}

	structs, err := q.SelectAllFrom(ServiceSoftwareVersionsTable, tail.String(), args...)
	if err != nil {
		return nil, err
	}

	versions := make([]*ServiceSoftwareVersions, len(structs))
	for i, s := range structs {
		versions[i] = s.(*ServiceSoftwareVersions) //nolint:forcetypeassert
	}

	return versions, nil
}

// DeleteServiceSoftwareVersions removes entry from the DB by service ID.
func DeleteServiceSoftwareVersions(q *reform.Querier, serviceID string) error {
	_, err := FindServiceSoftwareVersionsByServiceID(q, serviceID)
	if err != nil {
		return err
	}

	err = q.Delete(&ServiceSoftwareVersions{ServiceID: serviceID})
	if err != nil {
		return fmt.Errorf("failed to delete services software versions by service id '%s': %w", serviceID, err)
	}
	return nil
}
