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
	"golang.org/x/net/context"

	"github.com/percona/pmm-managed/api"
	"github.com/percona/pmm-managed/services/logs"
)

// check interface
var _ api.LogsServer = (*LogsServer)(nil)

func NewLogsServer(logs *logs.Logs) *LogsServer {
	return &LogsServer{
		logs: logs,
	}
}

type LogsServer struct {
	logs *logs.Logs
}

func (s *LogsServer) All(ctx context.Context, req *api.LogsAllRequest) (*api.LogsAllResponse, error) {
	var resp api.LogsAllResponse
	files := s.logs.Files()
	for i := range files {
		resp.Logs[files[i].Name] = &api.Log{
			Name: files[i].Name,
			Data: files[i].Data,
		}
	}

	return &resp, nil
}
