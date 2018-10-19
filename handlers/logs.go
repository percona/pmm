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
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/percona/pmm-managed/api"
	"github.com/percona/pmm-managed/services/logs"
)

type LogsServer struct {
	Logs *logs.Logs
}

// All returns last lines of all log files.
func (s *LogsServer) All(ctx context.Context, req *api.LogsAllRequest) (*api.LogsAllResponse, error) {
	resp := api.LogsAllResponse{
		Logs: make(map[string]*api.Log),
	}

	// fail-safe
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	for _, f := range s.Logs.Files(ctx) {
		lines := strings.Split(string(f.Data), "\n")
		resp.Logs[f.Name] = &api.Log{
			Lines: lines,
		}
	}

	return &resp, nil
}

// check interfaces
var (
	_ api.LogsServer = (*LogsServer)(nil)
)
