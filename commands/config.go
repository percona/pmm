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

package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/percona/pmm/nodeinfo"
	"github.com/sirupsen/logrus"

	"gopkg.in/alecthomas/kingpin.v2"
)

type configResult struct {
	Output string `json:"output"`
}

func (res *configResult) Result() {}

func (res *configResult) String() string {
	return res.Output
}

type configCommand struct {
	NodeAddress string
	NodeType    string
	NodeName    string

	NodeModel string
	Region    string
	Az        string

	Force bool
}

func (cmd *configCommand) Run() (Result, error) {
	var args []string

	port := GlobalFlags.ServerURL.Port()
	if port == "" {
		port = "443"
	}
	args = append(args, fmt.Sprintf("--server-address=%s:%s", GlobalFlags.ServerURL.Hostname(), port))

	if GlobalFlags.ServerURL.User != nil {
		args = append(args, fmt.Sprintf("--server-username=%s", GlobalFlags.ServerURL.User.Username()))
		password, ok := GlobalFlags.ServerURL.User.Password()
		if ok {
			args = append(args, fmt.Sprintf("--server-password=%s", password))
		}
	}

	if GlobalFlags.ServerInsecureTLS {
		args = append(args, "--server-insecure-tls")
	}

	if GlobalFlags.Debug {
		args = append(args, "--debug")
	}
	if GlobalFlags.Trace {
		args = append(args, "--trace")
	}

	args = append(args, "setup")
	if cmd.NodeModel != "" {
		args = append(args, fmt.Sprintf("--node-model=%s", cmd.NodeModel))
	}
	if cmd.Region != "" {
		args = append(args, fmt.Sprintf("--region=%s", cmd.Region))
	}
	if cmd.Az != "" {
		args = append(args, fmt.Sprintf("--az=%s", cmd.Az))
	}
	if cmd.Force {
		args = append(args, "--force")
	}
	args = append(args, cmd.NodeAddress, cmd.NodeType, cmd.NodeName)

	c := exec.Command("pmm-agent", args...) //nolint:gosec
	logrus.Debugf("Running: %s", strings.Join(c.Args, " "))
	b, err := c.CombinedOutput()
	res := &configResult{
		Output: string(b),
	}
	return res, err
}

// register command
var (
	Config  = new(configCommand)
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

	nodeTypeKeys := []string{"generic", "container"}
	nodeTypeDefault := "generic"
	if nodeinfo.Container {
		nodeTypeDefault = "container"
	}
	nodeTypeHelp := fmt.Sprintf("Node type, one of: %s (default: %s)", strings.Join(nodeTypeKeys, ", "), nodeTypeDefault)
	ConfigC.Arg("node-type", nodeTypeHelp).Default(nodeTypeDefault).EnumVar(&Config.NodeType, nodeTypeKeys...)

	hostname, _ := os.Hostname()
	nodeNameHelp := fmt.Sprintf("Node name (autodetected default: %s)", hostname)
	ConfigC.Arg("node-name", nodeNameHelp).Default(hostname).StringVar(&Config.NodeName)

	ConfigC.Flag("node-model", "Node model").StringVar(&Config.NodeModel)
	ConfigC.Flag("region", "Node region").StringVar(&Config.Region)
	ConfigC.Flag("az", "Node availability zone").StringVar(&Config.Az)

	ConfigC.Flag("force", "Remove Node with that name with all dependent Services and Agents if one exist").BoolVar(&Config.Force)
}
