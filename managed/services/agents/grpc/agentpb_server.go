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

// Package grpc contains functionality for agent <-> server connection.
package grpc

import (
	agentpb "github.com/percona/pmm/api/agent/pb"
	"github.com/percona/pmm/managed/services/agents"
)

// agentPBServer provides methods for pmm-agent <-> pmm-managed interactions.
type agentPBServer struct {
	handler *agents.Handler

	agentpb.UnimplementedAgentServer
}

// NewAgentPBServer creates new agent server.
func NewAgentPBServer(r *agents.Handler) agentpb.AgentServer { //nolint:ireturn
	return &agentPBServer{
		handler: r,
	}
}

// Connect establishes two-way communication channel between pmm-agent and pmm-managed.
func (s *agentPBServer) Connect(stream agentpb.Agent_ConnectServer) error {
	return s.handler.Run(stream)
}

// check interfaces.
var (
	_ agentpb.AgentServer = (*agentPBServer)(nil)
)
