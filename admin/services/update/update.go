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

// Package update implements update API.
package update

import (
	"context"

	"github.com/percona/pmm/api/updatepb"
)

type Server struct {
	// Context coming from cli commands. When cancelled, the command has been cancelled.
	cliCtx   context.Context
	upgrader upgrader

	updatepb.UnimplementedUpdateServer
}

// New returns new instance of Server.
func New(ctx context.Context, upgrader upgrader) (*Server, error) {
	return &Server{
		cliCtx:   ctx,
		upgrader: upgrader,
	}, nil
}

// StartUpdate starts PMM Server upgrade.
func (s *Server) StartUpdate(ctx context.Context, req *updatepb.StartUpdateRequest) (*updatepb.StartUpdateResponse, error) {
	logFileName, err := s.upgrader.StartUpgrade(s.cliCtx, req.ContainerId)
	if err != nil {
		return nil, err
	}

	return &updatepb.StartUpdateResponse{LogsToken: logFileName}, nil
}

// UpdateStatus returns PMM Server upgrade status.
func (s *Server) UpdateStatus(ctx context.Context, req *updatepb.UpdateStatusRequest) (*updatepb.UpdateStatusResponse, error) { //nolint:unparam
	res, err := s.upgrader.UpgradeStatus(s.cliCtx, req.LogsToken, req.Offset)
	if err != nil {
		return nil, err
	}

	return &updatepb.UpdateStatusResponse{
		Lines:  res.Lines,
		Offset: res.Offset,
		Done:   res.Done,
	}, nil
}
