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

// Package handlers implements gRPC API of pmm-managed.
package handlers

import (
	"context"

	"github.com/percona/pmm/api/serverpb"
)

type serverServer struct {
	version string
}

// NewServerServer returns Inventory API handler for managing Server.
func NewServerServer(version string) serverpb.ServerServer {
	return &serverServer{
		version: version,
	}
}

func (s *serverServer) Version(ctx context.Context, req *serverpb.VersionRequest) (*serverpb.VersionResponse, error) {
	return &serverpb.VersionResponse{
		Version: s.version,
	}, nil
}
