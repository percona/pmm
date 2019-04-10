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
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/percona/pmm/version"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v2"
)

// Paths represents binaries paths configuration.
type Paths struct {
	NodeExporter     string `yaml:"node_exporter"`
	MySQLdExporter   string `yaml:"mysqld_exporter"`
	MongoDBExporter  string `yaml:"mongodb_exporter"`
	PostgresExporter string `yaml:"postgres_exporter"`
	TempDir          string `yaml:"tempdir"`
}

// Lookup replaces paths with absolute paths.
func (p *Paths) Lookup() {
	p.NodeExporter, _ = exec.LookPath(p.NodeExporter)
	p.MySQLdExporter, _ = exec.LookPath(p.MySQLdExporter)
	p.MongoDBExporter, _ = exec.LookPath(p.MongoDBExporter)
	p.PostgresExporter, _ = exec.LookPath(p.PostgresExporter)
}

// Ports represents ports configuration.
type Ports struct {
	Min uint16 `yaml:"min"`
	Max uint16 `yaml:"max"`
}

// Config represents pmm-agent's static configuration.
//nolint:maligned
type Config struct {
	ID         string `yaml:"id"`
	Address    string `yaml:"address"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	ListenPort uint16 `yaml:"listen-port"`

	Debug       bool `yaml:"debug"`
	Trace       bool `yaml:"trace"`
	InsecureTLS bool `yaml:"insecure-tls"`

	Paths Paths `yaml:"paths"`
	Ports Ports `yaml:"ports"`
}

// application returns kingpin application that parses all flags and environment variables into cfg
// except --config-file that is returned separately.
func application(cfg *Config) (*kingpin.Application, *string) {
	app := kingpin.New("pmm-agent", fmt.Sprintf("Version %s.", version.Version))
	app.HelpFlag.Short('h')
	app.Version(version.FullInfo())

	// TODO
	app.Command("setup", "Not implemented yet.").Action(func(context *kingpin.ParseContext) error {
		return fmt.Errorf("not implemented yet")
	})

	configFileF := app.Flag("config-file", "Configuration file path. [PMM_AGENT_CONFIG_FILE]").
		Envar("PMM_AGENT_CONFIG_FILE").PlaceHolder("</path/to/pmm-agent.yaml>").String()

	app.Flag("id", "ID of this pmm-agent. [PMM_AGENT_ID]").
		Envar("PMM_AGENT_ID").PlaceHolder("</agent_id/...>").StringVar(&cfg.ID)
	app.Flag("listen-port", "Agent local API port. [PMM_AGENT_LISTEN_PORT]").
		Envar("PMM_AGENT_LISTEN_PORT").Default("7777").Uint16Var(&cfg.ListenPort)
	app.Flag("address", "PMM Server address. [PMM_AGENT_ADDRESS]").
		Envar("PMM_AGENT_ADDRESS").PlaceHolder("<host:port>").StringVar(&cfg.Address)

	app.Flag("debug", "Enable debug output. [PMM_AGENT_DEBUG]").
		Envar("PMM_AGENT_DEBUG").BoolVar(&cfg.Debug)
	app.Flag("trace", "Enable trace output (implies debug). [PMM_AGENT_TRACE]").
		Envar("PMM_AGENT_TRACE").BoolVar(&cfg.Trace)
	app.Flag("insecure-tls", "Skip PMM Server TLS certificate validation. [PMM_AGENT_INSECURE_TLS]").
		Envar("PMM_AGENT_INSECURE_TLS").BoolVar(&cfg.InsecureTLS)
	app.Flag("username", "HTTP BasicAuth username to connect to PMM Server. [PMM_AGENT_USERNAME]").
		Envar("PMM_AGENT_USERNAME").StringVar(&cfg.Username)
	app.Flag("password", "HTTP BasicAuth password to connect to PMM Server. [PMM_AGENT_PASSWORD]").
		Envar("PMM_AGENT_PASSWORD").StringVar(&cfg.Password)

	app.Flag("paths.node_exporter", "Path to node_exporter to use. [PMM_AGENT_PATHS_NODE_EXPORTER]").
		Envar("PMM_AGENT_PATHS_NODE_EXPORTER").Default("node_exporter").StringVar(&cfg.Paths.NodeExporter)
	app.Flag("paths.mysqld_exporter", "Path to mysqld_exporter to use. [PMM_AGENT_PATHS_MYSQLD_EXPORTER]").
		Envar("PMM_AGENT_PATHS_MYSQLD_EXPORTER").Default("mysqld_exporter").StringVar(&cfg.Paths.MySQLdExporter)
	app.Flag("paths.mongodb_exporter", "Path to mongodb_exporter to use. [PMM_AGENT_PATHS_MONGODB_EXPORTER]").
		Envar("PMM_AGENT_PATHS_MONGODB_EXPORTER").Default("mongodb_exporter").StringVar(&cfg.Paths.MongoDBExporter)
	app.Flag("paths.postgres_exporter", "Path to postgres_exporter to use. [PMM_AGENT_PATHS_POSTGRES_EXPORTER]").
		Envar("PMM_AGENT_PATHS_POSTGRES_EXPORTER").Default("postgres_exporter").StringVar(&cfg.Paths.PostgresExporter)
	app.Flag("paths.tempdir", "Temporary directory for exporters. [PMM_AGENT_PATHS_TEMPDIR]").
		Envar("PMM_AGENT_PATHS_TEMPDIR").Default(os.TempDir()).StringVar(&cfg.Paths.TempDir)

	// TODO read defaults from /proc/sys/net/ipv4/ip_local_port_range ?
	app.Flag("ports.min", "Minimal allowed port number for listening sockets. [PMM_AGENT_PORTS_MIN]").
		Envar("PMM_AGENT_PORTS_MIN").Default("32768").Uint16Var(&cfg.Ports.Min)
	app.Flag("ports.max", "Maximal allowed port number for listening sockets. [PMM_AGENT_PORTS_MAX]").
		Envar("PMM_AGENT_PORTS_MAX").Default("60999").Uint16Var(&cfg.Ports.Max)

	return app, configFileF
}

func readConfigFile(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path) //nolint:gosec
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(b, &cfg)
	return &cfg, err
}

// Get parses given command-line arguments and returns configuration.
func Get(args []string, l *logrus.Entry) (*Config, error) {
	// parse flags and environment variables
	cfg := new(Config)
	app, configFileF := application(cfg)
	_, err := app.Parse(args)
	if err != nil {
		return nil, err
	}

	// if config file is given, read and parse it, then re-parse flags into this configuration
	if *configFileF != "" {
		l.Infof("Loading configuration file %s.", *configFileF)
		if cfg, err = readConfigFile(*configFileF); err != nil {
			return nil, err
		}
		app, _ = application(cfg)
		if _, err = app.Parse(args); err != nil {
			return nil, err
		}
	}

	cfg.Paths.Lookup()
	return cfg, nil
}
