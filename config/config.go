// pmm-agent
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

// Package config provides access to pmm-agent configuration.
package config

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/percona/pmm/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Paths represents binaries paths configuration.
type Paths struct {
	NodeExporter    string
	MySQLdExporter  string
	MongoDBExporter string
	TempDir         string
}

// Lookup replaces paths with absolute paths.
func (p *Paths) Lookup() {
	p.NodeExporter, _ = exec.LookPath(p.NodeExporter)
	p.MySQLdExporter, _ = exec.LookPath(p.MySQLdExporter)
	p.MongoDBExporter, _ = exec.LookPath(p.MongoDBExporter)
}

// Ports represents ports configuration.
type Ports struct {
	Min uint16
	Max uint16
}

// Config represents pmm-agent's static configuration.
type Config struct {
	ID      string
	Address string

	Debug       bool
	InsecureTLS bool

	Paths Paths
	Ports Ports
}

func Application(cfg *Config) *kingpin.Application {
	app := kingpin.New("pmm-agent", fmt.Sprintf("Version %s.", version.Version))
	app.HelpFlag.Short('h')
	app.Version(version.FullInfo())

	app.Flag("id", "ID of this pmm-agent.").Envar("PMM_AGENT_ID").StringVar(&cfg.ID)
	app.Flag("address", "PMM Server address (host:port).").Envar("PMM_AGENT_ADDRESS").StringVar(&cfg.Address)

	app.Flag("debug", "Enable debug output.").Envar("PMM_AGENT_DEBUG").BoolVar(&cfg.Debug)
	app.Flag("insecure-tls", "Skip PMM Server TLS certificate validation.").Envar("PMM_AGENT_INSECURE_TLS").BoolVar(&cfg.InsecureTLS)

	app.Flag("paths.node_exporter", "Path to node_exporter to use.").Envar("PMM_AGENT_PATHS_NODE_EXPORTER").
		Default("node_exporter").StringVar(&cfg.Paths.NodeExporter)
	app.Flag("paths.mysqld_exporter", "Path to mysqld_exporter to use.").Envar("PMM_AGENT_PATHS_MYSQLD_EXPORTER").
		Default("mysqld_exporter").StringVar(&cfg.Paths.MySQLdExporter)
	app.Flag("paths.mongodb_exporter", "Path to mongodb_exporter to use.").Envar("PMM_AGENT_PATHS_MONGODB_EXPORTER").
		Default("mongodb_exporter").StringVar(&cfg.Paths.MongoDBExporter)
	app.Flag("paths.tempdir", "Temporary directory for exporters.").Envar("PMM_AGENT_PATHS_TEMPDIR").
		Default(os.TempDir()).StringVar(&cfg.Paths.TempDir)

	// TODO read defaults from /proc/sys/net/ipv4/ip_local_port_range ?
	app.Flag("ports.min", "Minimal allowed port number for listening sockets.").Envar("PMM_AGENT_PORTS_MIN").
		Default("32768").Uint16Var(&cfg.Ports.Min)
	app.Flag("ports.max", "Maximal allowed port number for listening sockets.").Envar("PMM_AGENT_PORTS_MAX").
		Default("60999").Uint16Var(&cfg.Ports.Max)

	// TODO load configuration from file with kingpin.ExpandArgsFromFile
	// TODO show environment variables in help

	return app
}
