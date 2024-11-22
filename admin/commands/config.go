// Copyright (C) 2023 Percona LLC
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

	"github.com/percona/pmm/admin/pkg/flags"
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

// ConfigCommand is used by Kong for CLI flags and commands.
//
//nolint:lll
type ConfigCommand struct {
	NodeAddress       string   `arg:"" default:"${nodeIp}" help:"Node address (autodetected, default: ${default})"`
	NodeType          string   `arg:"" enum:"generic,container" default:"${nodeTypeDefault}" help:"Node type. One of: [${enum}]. Default: ${default}"`
	NodeName          string   `arg:"" default:"${hostname}" help:"Node name (autodetected, default: ${default})"`
	NodeModel         string   `help:"Node model"`
	Region            string   `help:"Node region"`
	Az                string   `help:"Node availability zone"`
	AgentPassword     string   `help:"Custom password for /metrics endpoint"`
	Force             bool     `help:"Remove Node with that name with all dependent Services and Agents if one exist"`
	DisableCollectors []string `help:"Comma-separated list of collector names to exclude from exporter"`
	CustomLabels      string   `placeholder:"KEY=VALUE,KEY=VALUE,..." help:"Custom user-assigned labels"`
	BasePath          string   `name:"paths-base" help:"Base path where all binaries, tools and collectors of PMM client are located"`
	LogLinesCount     uint     `help:"Take and return N most recent log lines in logs.zip for each: server, every configured exporters and agents" default:"1024"`

	flags.MetricsModeFlags
	flags.LogLevelFatalFlags
}

func (cmd *ConfigCommand) args(globals *flags.GlobalFlags) ([]string, bool) {
	port := globals.ServerURL.Port()
	if port == "" {
		port = "443"
	}

	var switchedToTLS bool
	var res []string

	if globals.ServerURL.Scheme == "http" {
		port = "443"
		switchedToTLS = true
		globals.SkipTLSCertificateCheck = true
	}

	res = append(res, fmt.Sprintf("--server-address=%s:%s", globals.ServerURL.Hostname(), port))

	if globals.ServerURL.User != nil {
		res = append(res, fmt.Sprintf("--server-username=%s", globals.ServerURL.User.Username()))
		password, ok := globals.ServerURL.User.Password()
		if ok {
			res = append(res, fmt.Sprintf("--server-password=%s", password))
		}
	}

	if globals.PMMAgentListenPort != 0 {
		res = append(res, fmt.Sprintf("--listen-port=%d", globals.PMMAgentListenPort))
	}

	if globals.SkipTLSCertificateCheck {
		res = append(res, "--server-insecure-tls")
	}

	if cmd.LogLevelFatalFlags.LogLevel != "" {
		res = append(res, fmt.Sprintf("--log-level=%s", cmd.LogLevelFatalFlags.LogLevel))
	}
	if globals.EnableDebug {
		res = append(res, "--debug")
	}
	if globals.EnableTrace {
		res = append(res, "--trace")
	}

	if cmd.LogLinesCount > 0 {
		res = append(res, fmt.Sprintf("--log-lines-count=%d", cmd.LogLinesCount))
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

	if cmd.MetricsModeFlags.MetricsMode != "" {
		res = append(res, fmt.Sprintf("--metrics-mode=%s", cmd.MetricsModeFlags.MetricsMode))
	}

	if len(cmd.DisableCollectors) != 0 {
		res = append(res, fmt.Sprintf("--disable-collectors=%s", strings.Join(cmd.DisableCollectors, ",")))
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

	return res, switchedToTLS
}

// RunCmd runs config command.
func (cmd *ConfigCommand) RunCmd(globals *flags.GlobalFlags) (Result, error) {
	args, switchedToTLS := cmd.args(globals)
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
