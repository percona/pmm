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

package handlers

import (
	"context"

	api "github.com/percona/pmm/api/inventory"

	"github.com/percona/pmm-managed/services/inventory"
)

type ServicesServer struct {
	Services *inventory.ServicesService
}

func (s *ServicesServer) ListServices(ctx context.Context, req *api.ListServicesRequest) (*api.ListServicesResponse, error) {
	panic("not implemented")
}

func (s *ServicesServer) GetService(ctx context.Context, req *api.GetServiceRequest) (*api.GetServiceResponse, error) {
	panic("not implemented")
}

func (s *ServicesServer) AddMySQLService(ctx context.Context, req *api.AddMySQLServiceRequest) (*api.AddMySQLServiceResponse, error) {
	panic("not implemented")
}

func (s *ServicesServer) RemoveService(ctx context.Context, req *api.RemoveServiceRequest) (*api.RemoveServiceResponse, error) {
	panic("not implemented")
}

// check interfaces
var (
	_ api.ServicesServer = (*ServicesServer)(nil)
)
