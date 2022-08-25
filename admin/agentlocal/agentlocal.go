// Copyright 2019 Percona LLC
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

// Package agentlocal provides facilities for accessing local pmm-agent API.
package agentlocal

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/AlekSi/pointer"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/api/agentlocalpb/json/client"
	agentlocal "github.com/percona/pmm/api/agentlocalpb/json/client/agent_local"
)

// DefaultClient is http client configured either for socket or tcp connection to local pmm-agent.
// It shall be used to talk to the local pmm-agent.
var DefaultClient = http.DefaultClient

// SetTransport configures transport for accessing local pmm-agent API.
func SetTransport(ctx context.Context, debug bool, port uint32, socket string) {
	// use JSON APIs over HTTP/1.1
	transport := configureTransport(port, socket)
	transport.SetLogger(logrus.WithField("component", "agentlocal-transport"))
	transport.SetDebug(debug)
	transport.Context = ctx

	// disable HTTP/2
	httpTransport, ok := transport.Transport.(*http.Transport)
	if !ok {
		panic("Cannot assert transport to *http.Transport")
	}
	httpTransport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)

	client.Default.SetTransport(transport)
}

// configureTransport configures transport based on provided socket or port.
func configureTransport(port uint32, socket string) *httptransport.Runtime {
	if socket != "" {
		// In order to connect via socket, we need to override DialContext.
		// The rest of the configuration is from http.DefaultTransport.
		tr, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			panic("Cannot assert http.DefaultTransport to *http.Transport")
		}

		t := tr.Clone()
		t.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			dialer := net.Dialer{}
			return dialer.DialContext(ctx, "unix", socket)
		}
		cl := &http.Client{
			Transport: t,
		}
		transport := httptransport.NewWithClient(GetHostname("", 0, socket), "/", []string{"http"}, cl)
		DefaultClient = cl

		return transport
	}

	transport := httptransport.New(GetHostname(Localhost, port, ""), "/", []string{"http"})

	return transport
}

type NetworkInfo bool

const (
	RequestNetworkInfo        NetworkInfo = true
	DoNotRequestNetworkInfo   NetworkInfo = false
	Localhost                             = "127.0.0.1"
	DefaultPMMAgentListenPort             = 7777
)

// ErrNotSetUp is returned by GetStatus when pmm-agent is running, but not set up.
var ErrNotSetUp = fmt.Errorf("pmm-agent is running, but not set up")

// ErrNotConnected is returned by GetStatus when pmm-agent is running and set up, but not connected to PMM Server.
var ErrNotConnected = fmt.Errorf("pmm-agent is not connected to PMM Server")

// Status represents pmm-agent status.
type Status struct {
	AgentID  string `json:"agent_id"`
	NodeID   string `json:"node_id"`
	NodeName string `json:"node_name"`

	ServerURL         string `json:"server_url"`
	ServerInsecureTLS bool   `json:"server_insecure_tls"`
	ServerVersion     string `json:"server_version"`
	AgentVersion      string `json:"agent_version"`

	Agents []AgentStatus `json:"agents"`

	Connected        bool          `json:"connected"`
	ServerClockDrift time.Duration `json:"server_clock_drift,omitempty"`
	ServerLatency    time.Duration `json:"server_latency,omitempty"`

	ConnectionUptime float32 `json:"connection_uptime"`
}

type AgentStatus struct {
	AgentID   string `json:"agent_id"`
	AgentType string `json:"agent_type"`
	Status    string `json:"status"`
	Port      int64  `json:"listen_port,omitempty"`
}

// GetRawStatus returns raw local pmm-agent status. No special cases.
// Most callers should use GetStatus instead.
func GetRawStatus(ctx context.Context, requestNetworkInfo NetworkInfo) (*agentlocal.StatusOKBody, error) {
	params := &agentlocal.StatusParams{
		Body: agentlocal.StatusBody{
			GetNetworkInfo: bool(requestNetworkInfo),
		},
		Context: ctx,
	}

	res, err := client.Default.AgentLocal.Status(params)
	if err != nil {
		if res == nil {
			return nil, err
		}
		return res.Payload, err
	}
	return res.Payload, nil
}

// GetStatus returns local pmm-agent status.
// As a special case, if pmm-agent is running, but not set up, ErrNotSetUp is returned.
// If pmm-agent is set up, but not connected ErrNotConnected is returned.
func GetStatus(requestNetworkInfo NetworkInfo) (*Status, error) {
	var err error
	p, err := GetRawStatus(context.TODO(), requestNetworkInfo)
	if err != nil {
		return nil, err
	}

	if p.AgentID == "" || p.ServerInfo == nil {
		return nil, ErrNotSetUp
	}

	u, err := url.Parse(p.ServerInfo.URL)
	if err != nil {
		return nil, err
	}

	if p.RunsOnNodeID == "" {
		// set error but not return it immediately because we want
		// in this case to get some information from agent
		err = ErrNotConnected
	}

	agents := make([]AgentStatus, len(p.AgentsInfo))
	for i, a := range p.AgentsInfo {
		agents[i] = AgentStatus{
			AgentID:   a.AgentID,
			AgentType: pointer.GetString(a.AgentType),
			Status:    pointer.GetString(a.Status),
			Port:      a.ListenPort,
		}
	}
	var clockDrift time.Duration
	var latency time.Duration
	if bool(requestNetworkInfo) && p.ServerInfo.Connected {
		clockDrift, err = time.ParseDuration(p.ServerInfo.ClockDrift)
		if err != nil {
			return nil, err
		}
		latency, err = time.ParseDuration(p.ServerInfo.Latency)
		if err != nil {
			return nil, err
		}
	}

	agentVersion := p.AgentVersion
	if agentVersion == "" {
		agentVersion = "unknown"
	}

	return &Status{
		AgentID:  p.AgentID,
		NodeID:   p.RunsOnNodeID,
		NodeName: p.NodeName,

		ServerURL:         u.String(),
		ServerInsecureTLS: p.ServerInfo.InsecureTLS,
		ServerVersion:     p.ServerInfo.Version,
		AgentVersion:      agentVersion,

		Agents: agents,

		Connected:        p.ServerInfo.Connected,
		ServerClockDrift: clockDrift,
		ServerLatency:    latency,

		ConnectionUptime: p.ConnectionUptime,
	}, err
}

// GetHostname returns hostname for HTTP request depending on socket or host/port arguments.
func GetHostname(host string, port uint32, socket string) string {
	if socket != "" {
		return "unix-socket"
	}

	return net.JoinHostPort(host, strconv.Itoa(int(port)))
}
