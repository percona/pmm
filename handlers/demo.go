// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package handlers

import (
	"fmt"

	"golang.org/x/net/context"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/api"
)

type DemoServer struct{}

func (s *DemoServer) Error(ctx context.Context, req *api.DemoErrorRequest) (*api.DemoErrorResponse, error) {
	if req.Code >= 100 {
		panic(fmt.Sprintf("panic with code %d", req.Code))
	}

	code := codes.Code(req.Code)
	switch code {
	case codes.OK:
		return &api.DemoErrorResponse{}, nil
	case codes.InvalidArgument:
		return nil, status.ErrorProto(&spb.Status{
			Code:    int32(codes.InvalidArgument),
			Message: "invalid argument",
		})
	default:
		return nil, status.Errorf(code, "unhandled error: %s", code)
	}
}

// check interfaces
var (
	_ api.DemoServer = (*DemoServer)(nil)
)
