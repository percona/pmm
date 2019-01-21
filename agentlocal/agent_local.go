// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package agentlocal

import (
	"context"

	api "github.com/percona/pmm/api/agent_local"
)

type AgentLocalServer struct {
}

func (als *AgentLocalServer) Status(ctx context.Context, req *api.StatusRequest) (*api.StatusResponse, error) {
	panic("not implemented")
}

// check interfaces
var (
	_ api.AgentLocalServer = (*AgentLocalServer)(nil)
)
