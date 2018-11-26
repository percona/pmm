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

package inventory

import (
	"context"
	"fmt"

	"github.com/percona/pmm/api/inventory"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// ServicesService works with inventory API Services.
type ServicesService struct {
	Q *reform.Querier
}

// makeService converts database row to Inventory API Service.
func makeService(row *models.ServiceRow) inventory.Service {
	switch row.Type {
	case models.MySQLServiceType:
		return &inventory.MySQLService{
			Id:         row.ID,
			Name:       "",
			NodeId:     row.NodeID,
			Address:    "",
			Port:       0,
			UnixSocket: "",
		}
	default:
		panic(fmt.Errorf("unhandled ServiceRow type %s", row.Type))
	}
}

// List selects all Services in a stable order.
func (ss *ServicesService) List(ctx context.Context) ([]inventory.Service, error) {
	structs, err := ss.Q.SelectAllFrom(models.ServiceRowTable, "ORDER BY id")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]inventory.Service, len(structs))
	for i, str := range structs {
		row := str.(*models.ServiceRow)
		res[i] = makeService(row)
	}
	return res, nil
}

// Get selects a single Service by ID.
func (ss *ServicesService) Get(ctx context.Context, id uint32) (inventory.Service, error) {
	row := &models.ServiceRow{ID: id}
	if err := ss.Q.Reload(row); err != nil {
		if err == reform.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "Service with ID %d not found.", id)
		}
		return nil, errors.WithStack(err)
	}

	return makeService(row), nil
}

// Remove deletes Service by ID.
func (ss *ServicesService) Remove(ctx context.Context, id uint32) error {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// ID is not 0.

	// TODO check absence of Agents

	err := ss.Q.Delete(&models.ServiceRow{ID: id})
	if err == reform.ErrNoRows {
		return status.Errorf(codes.NotFound, "Service with ID %d not found.", id)
	}
	return errors.WithStack(err)
}
