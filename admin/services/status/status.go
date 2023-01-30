// Copyright 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package status implements status API.
package status

import (
	"context"

	"github.com/percona/pmm/api/updatepb"
)

type Server struct {
	updatepb.UnimplementedStatusServer
}

// New returns new instance of Server.
func New() *Server {
	return &Server{}
}

// Available returns no error if the daemon is available.
func (s *Server) Available(ctx context.Context, req *updatepb.AvailableRequest) (*updatepb.AvailableResponse, error) {
	return &updatepb.AvailableResponse{}, nil
}
