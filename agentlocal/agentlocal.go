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
	"net/http"
	"net/url"

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
	transport.Transport.(*http.Transport).TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}

	agentlocal.Default.SetTransport(transport)
}

// Status represents pmm-agent status.
type Status struct {
	AgentID string
	NodeID  string

	ServerURL         *url.URL
	ServerInsecureTLS bool
}

// GetStatus returns local pmm-agent status.
func GetStatus() (*Status, error) {
	res, err := agentlocal.Default.AgentLocal.Status(nil)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(res.Payload.ServerInfo.URL)
	if err != nil {
		return nil, err
	}

	return &Status{
		AgentID:           res.Payload.AgentID,
		NodeID:            res.Payload.RunsOnNodeID,
		ServerURL:         u,
		ServerInsecureTLS: res.Payload.ServerInfo.InsecureTLS,
	}, nil
}
