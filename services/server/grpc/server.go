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

// Package grpc implements gRPC APIs of pmm-managed Server API.
package grpc

import (
	"context"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/version"

	"github.com/percona/pmm-managed/utils/logger"
)

type server struct {
}

// NewServer returns new server for Server service.
func NewServer() serverpb.ServerServer {
	return new(server)
}

// Version returns PMM Server version.
func (s *server) Version(ctx context.Context, req *serverpb.VersionRequest) (*serverpb.VersionResponse, error) {
	res := &serverpb.VersionResponse{
		Version:          version.Version,
		PmmManagedCommit: version.FullCommit,
	}

	sec, err := strconv.ParseInt(version.Timestamp, 10, 64)
	if err == nil {
		res.Timestamp, err = ptypes.TimestampProto(time.Unix(sec, 0))
	}
	if err != nil {
		logger.Get(ctx).Warn(err)
	}

	return res, nil
}
