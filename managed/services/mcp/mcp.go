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
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

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

All tools are read-only. MySQL EXPLAIN is never EXPLAIN ANALYZE; MongoDB explain
runs the read to collect execution statistics but never modifies data. Errors
come back as "error: <code>" followed by a remediation message; act on the
message.`

// hourWindow is the default look-back for deep links that have no window of their own.
const hourWindow = time.Hour

// maxRequestBodyBytes bounds one MCP request body; tool arguments are tiny.
const maxRequestBodyBytes = 1 << 20

// Service serves PMM's MCP endpoint.
type Service struct {
	l             *logrus.Entry
	enabled       func() bool
	api           pmmAPI
	handler       http.Handler
	rawSQL        func() bool
	actionTimeout func() time.Duration
	publicAddress func(context.Context) string
	now           func() time.Time

	// warnPeer logs the first non-loopback rejection at the default level.
	warnPeer sync.Once
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
	// RawSQL reports whether tool output may include statements with literal
	// values (query examples, the explained statement on EXPLAIN output).
	// When it returns false, only normalized text is emitted. Defaults to false.
	RawSQL func() bool
	// ActionTimeout returns the EXPLAIN / SHOW CREATE TABLE polling deadline.
	// Defaults to DefaultActionTimeout.
	ActionTimeout func() time.Duration
	// PublicAddress returns settings.PMMPublicAddress (may be empty); used for "View in PMM" links.
	PublicAddress func(context.Context) string
}

// isLoopbackPeer reports whether a request's remote address is a loopback IP.
//
// nginx reaches pmm-managed over 127.0.0.1 whatever interface pmm-managed is
// bound to, so a loopback peer is how a request shows it came through nginx -
// and therefore through auth_request. Anything else, an empty or unparseable
// address included, is treated as a direct connection: the conservative
// answer, since getting this wrong exposes /mcp with no authorization in front.
func isLoopbackPeer(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// New creates a new MCP service.
func New(params Params) (*Service, error) {
	s := &Service{
		l:             logrus.WithField("component", "mcp"),
		enabled:       params.Enabled,
		api:           params.API,
		rawSQL:        params.RawSQL,
		actionTimeout: params.ActionTimeout,
		publicAddress: params.PublicAddress,
		now:           time.Now,
	}
	// Both switches default to off when the caller supplies no callback, so a
	// wiring mistake cannot expose the endpoint or un-redact its output. This
	// matches models.MCPEnabledDefault / models.MCPRawSQLDefault.
	if s.enabled == nil {
		s.enabled = func() bool { return false }
	}
	if s.rawSQL == nil {
		s.rawSQL = func() bool { return false }
	}
	if s.actionTimeout == nil {
		s.actionTimeout = func() time.Duration { return DefaultActionTimeout }
	}
	if s.api == nil {
		loopback := params.LoopbackURL
		if loopback == "" {
			loopback = DefaultLoopbackURL
		}
		c, err := newClient(loopback)
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
		// rejects a loopback connection carrying a non-loopback Host, which is
		// every request nginx forwards, so it has to be off. It would not help
		// against direct access anyway: a direct connection arrives on a
		// non-loopback address, and the SDK only checks loopback ones.
		// Handler enforces the property that matters instead: the peer must
		// be loopback, i.e. the request came through nginx and auth_request.
		DisableLocalhostProtection: true,
		MaxRequestBodyBytes:        maxRequestBodyBytes,
	})

	return s, nil
}

// Handler returns the HTTP handler to mount at /mcp and /mcp/.
//
// A request that did not arrive from a loopback peer is answered with 404,
// like a disabled endpoint: PMM_INTERFACE_TO_BIND can bind pmm-managed to a
// routable interface, and /mcp must stay reachable only through nginx, where
// auth_request authorizes the caller. The peer is checked first because it
// costs nothing, while the enabled switch is a settings query.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if !isLoopbackPeer(req.RemoteAddr) {
			// The answer is the 404 of a disabled endpoint, so a front proxy
			// that does not connect over loopback would look like MCP is off.
			l := s.l.WithField("peer", req.RemoteAddr)
			s.warnPeer.Do(func() {
				l.Warn("Rejected /mcp request from a non-loopback peer: /mcp is served only through PMM's nginx; " +
					"further rejections are logged at debug level.")
			})
			l.Debug("Rejected /mcp request from a non-loopback peer.")
			http.NotFound(rw, req)
			return
		}
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
	s.registerQANTools(server)
	s.registerActionTools(server)
	s.registerConfigTools(server)
}
