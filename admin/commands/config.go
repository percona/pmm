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

package commands

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

type configResult struct {
	Warning string `json:"warning"`
	Output  string `json:"output"`
}

func (res *configResult) Result() {}

func (res *configResult) String() string {
	s := res.Output
	if res.Warning != "" {
		s = res.Warning + "\n" + s
	}
	return s
}

type ConfigCommand struct {
	NodeAddress       string `arg:"" default:"${nodeIp}" help:"Node address (autodetected default: ${nodeIp})"`
	NodeType          string `arg:"" enum:"generic,container" default:"${nodeTypeDefault}" help:"Node type, one of: generic, container (default: ${nodeTypeDefault})"`
	NodeName          string `arg:"" default:"${hostname}" help:"Node name (autodetected default: ${hostname})"`
	NodeModel         string `help:"Node model"`
	Region            string `help:"Node region"`
	Az                string `help:"Node availability zone"`
	AgentPassword     string `help:"Custom password for /metrics endpoint"`
	Force             bool   `help:"Remove Node with that name with all dependent Services and Agents if one exist"`
	MetricsMode       string `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode for agents node-exporter, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	DisableCollectors string `help:"Comma-separated list of collector names to exclude from exporter"`
	CustomLabels      string `help:"Custom user-assigned labels"`
	BasePath          string `name:"paths-base" help:"Base path where all binaries, tools and collectors of PMM client are located"`
	LogLevel          string `enum:"debug,info,warn,error,fatal" default:"warn" help:"Logging level"`
}

func (cmd *ConfigCommand) args() (res []string, switchedToTLS bool) {
	port := GlobalFlags.ServerURL.Port()
	if port == "" {
		port = "443"
	}
	if GlobalFlags.ServerURL.Scheme == "http" {
		port = "443"
		switchedToTLS = true
		GlobalFlags.ServerInsecureTLS = true
	}
	res = append(res, fmt.Sprintf("--server-address=%s:%s", GlobalFlags.ServerURL.Hostname(), port))

	if GlobalFlags.ServerURL.User != nil {
		res = append(res, fmt.Sprintf("--server-username=%s", GlobalFlags.ServerURL.User.Username()))
		password, ok := GlobalFlags.ServerURL.User.Password()
		if ok {
			res = append(res, fmt.Sprintf("--server-password=%s", password))
		}
	}

	if GlobalFlags.PMMAgentListenPort != 0 {
		res = append(res, fmt.Sprintf("--listen-port=%d", GlobalFlags.PMMAgentListenPort))
	}

	if GlobalFlags.ServerInsecureTLS {
		res = append(res, "--server-insecure-tls")
	}

	if cmd.LogLevel != "" {
		res = append(res, fmt.Sprintf("--log-level=%s", cmd.LogLevel))
	}
	if GlobalFlags.Debug {
		res = append(res, "--debug")
	}
	if GlobalFlags.Trace {
		res = append(res, "--trace")
	}

	res = append(res, "setup")
	if cmd.NodeModel != "" {
		res = append(res, fmt.Sprintf("--node-model=%s", cmd.NodeModel))
	}
	if cmd.Region != "" {
		res = append(res, fmt.Sprintf("--region=%s", cmd.Region))
	}
	if cmd.Az != "" {
		res = append(res, fmt.Sprintf("--az=%s", cmd.Az))
	}
	if cmd.Force {
		res = append(res, "--force")
	}

	if cmd.MetricsMode != "" {
		res = append(res, fmt.Sprintf("--metrics-mode=%s", cmd.MetricsMode))
	}

	if cmd.DisableCollectors != "" {
		res = append(res, fmt.Sprintf("--disable-collectors=%s", cmd.DisableCollectors))
	}

	if cmd.CustomLabels != "" {
		res = append(res, fmt.Sprintf("--custom-labels=%s", cmd.CustomLabels))
	}

	if cmd.BasePath != "" {
		res = append(res, fmt.Sprintf("--paths-base=%s", cmd.BasePath))
	}

	if cmd.AgentPassword != "" {
		res = append(res, fmt.Sprintf("--agent-password=%s", cmd.AgentPassword))
	}

	res = append(res, cmd.NodeAddress, cmd.NodeType, cmd.NodeName)

	return //nolint:nakedret
}

func (cmd *ConfigCommand) RunCmd() (Result, error) {
	args, switchedToTLS := cmd.args()
	c := exec.Command("pmm-agent", args...) //nolint:gosec
	logrus.Debugf("Running: %s", strings.Join(c.Args, " "))
	b, err := c.Output() // hide pmm-agent's stderr logging
	res := &configResult{
		Output: strings.TrimSpace(string(b)),
	}
	if switchedToTLS {
		res.Warning = `Warning: PMM Server requires TLS communications with client.`
	}
	return res, err
}
