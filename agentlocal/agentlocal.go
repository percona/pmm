// pmm-admin
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

// Package agentlocal provides facilities for accessing local pmm-agent API.
package agentlocal

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

	"github.com/AlekSi/pointer"
	httptransport "github.com/go-openapi/runtime/client"
	agentlocal "github.com/percona/pmm/api/agentlocalpb/json/client"
	"github.com/sirupsen/logrus"
)

// SetTransport configures transport for accessing local pmm-agent API.
func SetTransport(ctx context.Context, debug bool) {
	// use JSON APIs over HTTP/1.1
	transport := httptransport.New("127.0.0.1:7777", "/", []string{"http"})
	transport.SetLogger(logrus.WithField("component", "agentlocal-transport"))
	transport.SetDebug(debug)
	transport.Context = ctx

	// disable HTTP/2
	httpTransport := transport.Transport.(*http.Transport)
	httpTransport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}

	agentlocal.Default.SetTransport(transport)
}

// ErrNotSetUp is returned by GetStatus when pmm-agent is running, but not set up.
var ErrNotSetUp = fmt.Errorf("pmm-agent is running, but not set up")

// Status represents pmm-agent status.
type Status struct {
	AgentID string `json:"agent_id"`
	NodeID  string `json:"node_id"`

	ServerURL         *url.URL `json:"server_url"`
	ServerInsecureTLS bool     `json:"server_insecure_tls"`
	ServerVersion     string   `json:"server_version"`

	Agents []AgentStatus `json:"agents"`
	// TODO latency / last ping time / connection status
}

type AgentStatus struct {
	AgentID   string `json:"agent_id"`
	AgentType string `json:"agent_type"`
	Status    string `json:"status"`
}

// GetStatus returns local pmm-agent status.
// As a special case, if pmm-agent is running, but not set up, ErrNotSetUp is returned.
func GetStatus() (*Status, error) {
	res, err := agentlocal.Default.AgentLocal.Status(nil)
	if err != nil {
		return nil, err
	}

	p := res.Payload
	if p.AgentID == "" || p.RunsOnNodeID == "" || p.ServerInfo == nil {
		return nil, ErrNotSetUp
	}

	u, err := url.Parse(p.ServerInfo.URL)
	if err != nil {
		return nil, err
	}

	agents := make([]AgentStatus, len(p.AgentsInfo))
	for i, a := range p.AgentsInfo {
		agents[i] = AgentStatus{
			AgentID:   a.AgentID,
			AgentType: pointer.GetString(a.AgentType),
			Status:    pointer.GetString(a.Status),
		}
	}

	return &Status{
		AgentID: p.AgentID,
		NodeID:  p.RunsOnNodeID,

		ServerURL:         u,
		ServerInsecureTLS: p.ServerInfo.InsecureTLS,
		ServerVersion:     p.ServerInfo.Version,

		Agents: agents,
	}, nil
}
