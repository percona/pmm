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
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/api"
)

type DemoServer struct{}

func (s *DemoServer) Error(ctx context.Context, req *api.DemoErrorRequest) (*api.DemoErrorResponse, error) {
	switch {
	case req.Code == 0:
		return &api.DemoErrorResponse{}, nil
	case req.Code < 20:
		code := codes.Code(req.Code)
		return nil, status.Error(code, code.String())
	case req.Code < 40:
		return nil, errors.Errorf("pkg/errors error with code %d", req.Code)
	case req.Code < 60:
		return nil, fmt.Errorf("raw error with code %d", req.Code)
	case req.Code < 80:
		panic(errors.Errorf("pkg/errors panic with code %d", req.Code))
	case req.Code < 100:
		panic(fmt.Errorf("error panic with code %d", req.Code))
	default:
		panic(fmt.Sprintf("string panic with code %d", req.Code))
	}
}

// check interfaces
var (
	_ api.DemoServer = (*DemoServer)(nil)
)
