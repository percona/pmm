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

	"github.com/percona/pmm/api/inventory"

	"github.com/percona/pmm-managed/services/agents"
)

type ServicesServer struct {
	Store *agents.Store
}

func (s *ServicesServer) ListServices(ctx context.Context, req *inventory.ListServicesRequest) (*inventory.ListServicesResponse, error) {
	panic("not implemented")
}

func (s *ServicesServer) GetService(ctx context.Context, req *inventory.GetServiceRequest) (*inventory.GetServiceResponse, error) {
	panic("not implemented")
}

func (s *ServicesServer) AddMySQLService(ctx context.Context, req *inventory.AddMySQLServiceRequest) (*inventory.AddMySQLServiceResponse, error) {
	panic("not implemented")
}

func (s *ServicesServer) RemoveService(ctx context.Context, req *inventory.RemoveServiceRequest) (*inventory.RemoveServiceResponse, error) {
	panic("not implemented")
}

// check interfaces
var (
	_ inventory.ServicesServer = (*ServicesServer)(nil)
)
