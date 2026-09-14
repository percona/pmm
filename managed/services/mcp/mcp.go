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

package mcp

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/version"
)

// serverName is the MCP implementation name reported to clients on initialize.
const serverName = "pmm-mcp"

// Instructions is sent to MCP clients on initialize. It tells a model how the
// tools fit together so that a triage runs in the right order.
const Instructions = `PMM (Percona Monitoring and Management) exposes read-only database
diagnostics through these tools. Recommended order for a slow-query triage:

1. pmm_version - confirm the connection and the caller's token work.
2. pmm_inventory - list monitored services (service_id, service_name, engine,
   version) and pick the one to investigate.
3. pmm_top_queries - rank the worst queries for that service over a time window;
   each row carries a queryid.
4. pmm_query_detail - fetch metrics, schema and an example statement for one
   queryid.
5. pmm_get_explain / pmm_get_schema - fetch the execution plan and the table
   DDL for the query's tables.
6. pmm_get_config - read the server's numeric configuration variables.

All tools are read-only. EXPLAIN is never EXPLAIN ANALYZE. Errors come back as
"error: <code>" followed by a remediation message; act on the message.`

// Service serves PMM's MCP endpoint.
type Service struct {
	l       *logrus.Entry
	enabled func() bool
	api     pmmAPI
	handler http.Handler

	// dsUID caches the uid of the Grafana metrics datasource.
	dsMu  sync.Mutex
	dsUID string
}

// Params holds the dependencies and configuration of the MCP service.
type Params struct {
	// Enabled reports whether the endpoint is switched on. When it returns
	// false, the handler answers 404 so that the feature is invisible.
	Enabled func() bool
	// LoopbackURL is the base URL of PMM's REST API as seen from inside the
	// server (nginx's plain-HTTP listener). Defaults to DefaultLoopbackURL.
	LoopbackURL string
	// API overrides the PMM API client; tests use it to inject fakes.
	API pmmAPI
}

// New creates a new MCP service.
func New(params Params) (*Service, error) {
	s := &Service{
		l:       logrus.WithField("component", "mcp"),
		enabled: params.Enabled,
		api:     params.API,
	}
	if s.enabled == nil {
		s.enabled = func() bool { return true }
	}
	if s.api == nil {
		loopback := params.LoopbackURL
		if loopback == "" {
			loopback = DefaultLoopbackURL
		}
		c, err := newClient(loopback, s.l)
		if err != nil {
			return nil, fmt.Errorf("mcp: %w", err)
		}
		s.api = c
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Title:   "Percona Monitoring and Management",
		Version: version.Version,
	}, &mcp.ServerOptions{
		Instructions: Instructions,
	})
	s.registerTools(server)

	s.handler = mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		// No session table: every request carries its own credentials and is
		// authorized on its own by nginx, so nothing has to be replicated in HA
		// mode and restarts are transparent to clients.
		Stateless: true,
		// nginx terminates the client connection and proxies to 127.0.0.1:7772
		// with the upstream name as Host header. The SDK's DNS-rebinding guard
		// would reject every such request; nginx auth_request is the guard here.
		DisableLocalhostProtection: true,
	})

	return s, nil
}

// Handler returns the HTTP handler to mount at /mcp and /mcp/.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if !s.enabled() {
			http.NotFound(rw, req)
			return
		}
		s.handler.ServeHTTP(rw, req)
	})
}

// registerTools adds every tool to the server.
func (s *Service) registerTools(server *mcp.Server) {
	s.registerInventoryTools(server)
}
