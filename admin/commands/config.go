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
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm/utils/nodeinfo"
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

type configCommand struct {
	NodeAddress string
	NodeType    string
	NodeName    string

	AgentPassword     string
	NodeModel         string
	Region            string
	Az                string
	MetricsMode       string
	DisableCollectors string
	CustomLabels      string
	BasePath          string
	ListenPort        uint32
	LogLevel          string

	LogLinesCount uint

	Force bool
}

func (cmd *configCommand) args() (res []string, switchedToTLS bool) {
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

func (cmd *configCommand) Run() (Result, error) {
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

// register command
var (
	Config  configCommand
	ConfigC = kingpin.Command("config", "Configure local pmm-agent")
)

func init() {
	nodeinfo := nodeinfo.Get()
	if nodeinfo.PublicAddress == "" {
		ConfigC.Arg("node-address", "Node address").Required().StringVar(&Config.NodeAddress)
	} else {
		help := fmt.Sprintf("Node address (autodetected default: %s)", nodeinfo.PublicAddress)
		ConfigC.Arg("node-address", help).Default(nodeinfo.PublicAddress).StringVar(&Config.NodeAddress)
	}

	configNodeTypeKeys := []string{"generic", "container"} // "remote" Node can't be registered with that API
	nodeTypeDefault := "generic"
	if nodeinfo.Container {
		nodeTypeDefault = "container"
	}
	nodeTypeHelp := fmt.Sprintf("Node type, one of: %s (default: %s)", strings.Join(configNodeTypeKeys, ", "), nodeTypeDefault)
	ConfigC.Arg("node-type", nodeTypeHelp).Default(nodeTypeDefault).EnumVar(&Config.NodeType, configNodeTypeKeys...)

	hostname, _ := os.Hostname()
	nodeNameHelp := fmt.Sprintf("Node name (autodetected default: %s)", hostname)
	ConfigC.Arg("node-name", nodeNameHelp).Default(hostname).StringVar(&Config.NodeName)

	ConfigC.Flag("node-model", "Node model").StringVar(&Config.NodeModel)
	ConfigC.Flag("region", "Node region").StringVar(&Config.Region)
	ConfigC.Flag("az", "Node availability zone").StringVar(&Config.Az)

	ConfigC.Flag("agent-password", "Custom password for /metrics endpoint").StringVar(&Config.AgentPassword)
	ConfigC.Flag("force", "Remove Node with that name with all dependent Services and Agents if one exist").BoolVar(&Config.Force)
	ConfigC.Flag("metrics-mode", "Metrics flow mode for agents node-exporter, can be push - agent will push metrics,"+
		" pull - server scrape metrics from agent  or auto - chosen by server.").Default("auto").EnumVar(&Config.MetricsMode, "auto", "push", "pull")
	ConfigC.Flag("disable-collectors", "Comma-separated list of collector names to exclude from exporter").StringVar(&Config.DisableCollectors)
	ConfigC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&Config.CustomLabels)
	ConfigC.Flag("paths-base", "Base path where all binaries, tools and collectors of PMM client are located").StringVar(&Config.BasePath)
	ConfigC.Flag("log-level", "Logging level").Default("warn").EnumVar(&Config.LogLevel, "debug", "info", "warn", "error", "fatal")
	ConfigC.Flag("log-lines-count", "Take and return N most recent log lines in logs.zip for each: server, every configured exporters and agents").Default("1024").UintVar(&Config.LogLinesCount)
}
