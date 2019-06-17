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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/percona/pmm/nodeinfo"
	"github.com/percona/pmm/version"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v2"
)

// Server represents PMM Server configuration.
type Server struct {
	Address     string `yaml:"address"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	InsecureTLS bool   `yaml:"insecure-tls"`
}

// URL returns base PMM Server URL for JSON APIs.
func (s *Server) URL() *url.URL {
	if s.Address == "" {
		return nil
	}

	var user *url.Userinfo
	switch {
	case s.Password != "":
		user = url.UserPassword(s.Username, s.Password)
	case s.Username != "":
		user = url.User(s.Username)
	}
	return &url.URL{
		Scheme: "https",
		User:   user,
		Host:   s.Address,
		Path:   "/",
	}
}

// Paths represents binaries paths configuration.
type Paths struct {
	NodeExporter     string `yaml:"node_exporter"`
	MySQLdExporter   string `yaml:"mysqld_exporter"`
	MongoDBExporter  string `yaml:"mongodb_exporter"`
	PostgresExporter string `yaml:"postgres_exporter"`
	ProxySQLExporter string `yaml:"proxysql_exporter"`
	PtSummary        string `yaml:"pt_summary"`
	PtMySQLSummary   string `yaml:"pt_mysql_summary"`
	TempDir          string `yaml:"tempdir"`

	SlowLogFilePrefix string `yaml:"slowlog_file_prefix,omitempty"` // for development and testing
}

// lookup replaces paths with absolute paths.
func (p *Paths) lookup() {
	p.NodeExporter, _ = exec.LookPath(p.NodeExporter)
	p.MySQLdExporter, _ = exec.LookPath(p.MySQLdExporter)
	p.MongoDBExporter, _ = exec.LookPath(p.MongoDBExporter)
	p.PostgresExporter, _ = exec.LookPath(p.PostgresExporter)
	p.ProxySQLExporter, _ = exec.LookPath(p.ProxySQLExporter)
	p.PtSummary, _ = exec.LookPath(p.PtSummary)
	p.PtMySQLSummary, _ = exec.LookPath(p.PtMySQLSummary)
}

// Ports represents ports configuration.
type Ports struct {
	Min uint16 `yaml:"min"`
	Max uint16 `yaml:"max"`
}

// Setup contains `pmm-agent setup` flag values.
// It is never stored in configuration file.
type Setup struct {
	NodeType      string
	NodeName      string
	MachineID     string
	Distro        string
	ContainerID   string
	ContainerName string
	NodeModel     string
	Region        string
	Az            string
	// TODO CustomLabels  string
	Address string

	Force bool
}

// Config represents pmm-agent's configuration.
//nolint:maligned
type Config struct {
	// no config file there

	ID         string `yaml:"id"`
	ListenPort uint16 `yaml:"listen-port"`

	Server Server `yaml:"server"`
	Paths  Paths  `yaml:"paths"`
	Ports  Ports  `yaml:"ports"`

	Debug bool `yaml:"debug"`
	Trace bool `yaml:"trace"`

	Setup Setup `yaml:"-"`
}

// ErrConfigFileDoesNotExist error is returned from Get method if configuration file is expected,
// but does not exist.
type ErrConfigFileDoesNotExist string

func (e ErrConfigFileDoesNotExist) Error() string {
	return fmt.Sprintf("configuration file %s does not exist", string(e))
}

// Get parses command-line flags, environment variables and configuration file
// (if --config-file/PMM_AGENT_CONFIG_FILE is defined).
// It returns configuration, configuration file path (value of -config-file/PMM_AGENT_CONFIG_FILE, may be empty),
// and any encountered error. That error may be ErrConfigFileDoesNotExist if configuration file path is not empty,
// but file itself does not exist. Configuration from command-line flags and environment variables
// is still returned in this case.
func Get(l *logrus.Entry) (*Config, string, error) {
	cfg, configFileF, err := get(os.Args[1:], l)
	if cfg != nil {
		cfg.Paths.lookup()
	}
	return cfg, configFileF, err
}

// get is Get for unit tests: parses args instead of command-line, and does not lookups paths.
func get(args []string, l *logrus.Entry) (*Config, string, error) {
	// parse command-line flags and environment variables
	cfg := new(Config)
	app, configFileF := Application(cfg)
	if _, err := app.Parse(args); err != nil {
		return nil, "", err
	}
	if *configFileF == "" {
		return cfg, "", nil
	}

	absConfigFileF, err := filepath.Abs(*configFileF)
	if err != nil {
		return nil, "", err
	}
	*configFileF = absConfigFileF
	l.Debugf("Loading configuration file %s.", *configFileF)
	fileCfg, err := loadFromFile(*configFileF)
	if _, ok := err.(ErrConfigFileDoesNotExist); ok {
		return cfg, *configFileF, err
	}
	if err != nil {
		return nil, "", err
	}

	// re-parse flags into configuration from file
	app, _ = Application(fileCfg)
	if _, err = app.Parse(args); err != nil {
		return nil, "", err
	}
	return fileCfg, *configFileF, nil
}

// Application returns kingpin application that will parse command-line flags and environment variables
// into cfg except --config-file/PMM_AGENT_CONFIG_FILE that is returned separately.
func Application(cfg *Config) (*kingpin.Application, *string) {
	app := kingpin.New("pmm-agent", fmt.Sprintf("Version %s", version.Version))
	app.HelpFlag.Short('h')
	app.Version(version.FullInfo())

	app.Command("run", "Run pmm-agent (default command)").Default()

	// this flags has to be optional and has empty default value for `pmm-agent setup`
	configFileF := app.Flag("config-file", "Configuration file path [PMM_AGENT_CONFIG_FILE]").
		Envar("PMM_AGENT_CONFIG_FILE").PlaceHolder("</path/to/pmm-agent.yaml>").String()

	app.Flag("id", "ID of this pmm-agent [PMM_AGENT_ID]").
		Envar("PMM_AGENT_ID").PlaceHolder("</agent_id/...>").StringVar(&cfg.ID)
	app.Flag("listen-port", "Agent local API port [PMM_AGENT_LISTEN_PORT]").
		Envar("PMM_AGENT_LISTEN_PORT").Default("7777").Uint16Var(&cfg.ListenPort)

	app.Flag("server-address", "PMM Server address [PMM_AGENT_SERVER_ADDRESS]").
		Envar("PMM_AGENT_SERVER_ADDRESS").PlaceHolder("<host:port>").StringVar(&cfg.Server.Address)
	app.Flag("server-username", "HTTP BasicAuth username to connect to PMM Server [PMM_AGENT_SERVER_USERNAME]").
		Envar("PMM_AGENT_SERVER_USERNAME").StringVar(&cfg.Server.Username)
	app.Flag("server-password", "HTTP BasicAuth password to connect to PMM Server [PMM_AGENT_SERVER_PASSWORD]").
		Envar("PMM_AGENT_SERVER_PASSWORD").StringVar(&cfg.Server.Password)
	app.Flag("server-insecure-tls", "Skip PMM Server TLS certificate validation [PMM_AGENT_SERVER_INSECURE_TLS]").
		Envar("PMM_AGENT_SERVER_INSECURE_TLS").BoolVar(&cfg.Server.InsecureTLS)

	app.Flag("paths-node_exporter", "Path to node_exporter to use [PMM_AGENT_PATHS_NODE_EXPORTER]").
		Envar("PMM_AGENT_PATHS_NODE_EXPORTER").Default("node_exporter").StringVar(&cfg.Paths.NodeExporter)
	app.Flag("paths-mysqld_exporter", "Path to mysqld_exporter to use [PMM_AGENT_PATHS_MYSQLD_EXPORTER]").
		Envar("PMM_AGENT_PATHS_MYSQLD_EXPORTER").Default("mysqld_exporter").StringVar(&cfg.Paths.MySQLdExporter)
	app.Flag("paths-mongodb_exporter", "Path to mongodb_exporter to use [PMM_AGENT_PATHS_MONGODB_EXPORTER]").
		Envar("PMM_AGENT_PATHS_MONGODB_EXPORTER").Default("mongodb_exporter").StringVar(&cfg.Paths.MongoDBExporter)
	app.Flag("paths-postgres_exporter", "Path to postgres_exporter to use [PMM_AGENT_PATHS_POSTGRES_EXPORTER]").
		Envar("PMM_AGENT_PATHS_POSTGRES_EXPORTER").Default("postgres_exporter").StringVar(&cfg.Paths.PostgresExporter)
	app.Flag("paths-proxysql_exporter", "Path to proxysql_exporter to use [PMM_AGENT_PATHS_PROXYSQL_EXPORTER]").
		Envar("PMM_AGENT_PATHS_PROXYSQL_EXPORTER").Default("proxysql_exporter").StringVar(&cfg.Paths.ProxySQLExporter)
	app.Flag("paths-pt-summary", "Path to pt-summary to use [PMM_AGENT_PATHS_PT_SUMMARY]").
		Envar("PMM_AGENT_PATHS_PT_SUMMARY").Default("pt-summary").StringVar(&cfg.Paths.PtSummary)
	app.Flag("paths-pt-mysql-summary", "Path to pt-mysql-summary to use [PMM_AGENT_PATHS_PT_MYSQL_SUMMARY]").
		Envar("PMM_AGENT_PATHS_PT_MYSQL_SUMMARY").Default("pt-mysql-summary").StringVar(&cfg.Paths.PtMySQLSummary)
	app.Flag("paths-tempdir", "Temporary directory for exporters [PMM_AGENT_PATHS_TEMPDIR]").
		Envar("PMM_AGENT_PATHS_TEMPDIR").Default(os.TempDir()).StringVar(&cfg.Paths.TempDir)
	// no flag for SlowLogFilePrefix - it is only for development and testing

	// TODO read defaults from /proc/sys/net/ipv4/ip_local_port_range ?
	app.Flag("ports-min", "Minimal allowed port number for listening sockets [PMM_AGENT_PORTS_MIN]").
		Envar("PMM_AGENT_PORTS_MIN").Default("32768").Uint16Var(&cfg.Ports.Min)
	app.Flag("ports-max", "Maximal allowed port number for listening sockets [PMM_AGENT_PORTS_MAX]").
		Envar("PMM_AGENT_PORTS_MAX").Default("60999").Uint16Var(&cfg.Ports.Max)

	app.Flag("debug", "Enable debug output [PMM_AGENT_DEBUG]").
		Envar("PMM_AGENT_DEBUG").BoolVar(&cfg.Debug)
	app.Flag("trace", "Enable trace output (implies debug) [PMM_AGENT_TRACE]").
		Envar("PMM_AGENT_TRACE").BoolVar(&cfg.Trace)

	setupCmd := app.Command("setup", "Configure local pmm-agent")
	nodeinfo := nodeinfo.Get()

	if nodeinfo.PublicAddress == "" {
		setupCmd.Arg("node-address", "Node address").Required().StringVar(&cfg.Setup.Address)
	} else {
		help := fmt.Sprintf("Node address (autodetected default: %s)", nodeinfo.PublicAddress)
		setupCmd.Arg("node-address", help).Default(nodeinfo.PublicAddress).StringVar(&cfg.Setup.Address)
	}

	nodeTypeKeys := []string{"generic", "container"}
	nodeTypeDefault := nodeTypeKeys[0]
	nodeTypeHelp := fmt.Sprintf("Node type, one of: %s (default: %s)", strings.Join(nodeTypeKeys, ", "), nodeTypeDefault)
	setupCmd.Arg("node-type", nodeTypeHelp).Default(nodeTypeDefault).EnumVar(&cfg.Setup.NodeType, nodeTypeKeys...)

	hostname, _ := os.Hostname()
	nodeNameHelp := fmt.Sprintf("Node name (autodetected default: %s)", hostname)
	setupCmd.Arg("node-name", nodeNameHelp).Default(hostname).StringVar(&cfg.Setup.NodeName)

	setupCmd.Flag("machine-id", "Node machine-id (default is autodetected)").Default(nodeinfo.MachineID).StringVar(&cfg.Setup.MachineID)
	setupCmd.Flag("distro", "Node OS distribution (default is autodetected)").Default(nodeinfo.Distro).StringVar(&cfg.Setup.Distro)
	setupCmd.Flag("container-id", "Container ID").StringVar(&cfg.Setup.ContainerID)
	setupCmd.Flag("container-name", "Container name").StringVar(&cfg.Setup.ContainerName)
	setupCmd.Flag("node-model", "Node model").StringVar(&cfg.Setup.NodeModel)
	setupCmd.Flag("region", "Node region").StringVar(&cfg.Setup.Region)
	setupCmd.Flag("az", "Node availability zone").StringVar(&cfg.Setup.Az)
	// TODO setupCmd.Flag("custom-labels", "Custom user-assigned labels").StringVar(&cfg.Setup.CustomLabels)

	setupCmd.Flag("force", "Remove Node with that name with all dependent Services and Agents if one exist").BoolVar(&cfg.Setup.Force)

	return app, configFileF
}

// loadFromFile loads configuration from file.
// As a special case, if file does not exist, it returns ErrConfigFileDoesNotExist.
// Other errors are returned if file exists, but configuration can't be loaded due to permission problems,
// YAML parsing problems, etc.
func loadFromFile(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, ErrConfigFileDoesNotExist(path)
	}

	b, err := ioutil.ReadFile(path) //nolint:gosec
	if err != nil {
		return nil, err
	}
	cfg := new(Config)
	if err = yaml.Unmarshal(b, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// SaveToFile saves configuration to file.
// No special cases.
func SaveToFile(path string, cfg *Config, comment string) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	var res []byte
	if comment != "" {
		res = []byte("# " + comment + "\n")
	}
	res = append(res, "---\n"...)
	res = append(res, b...)
	return ioutil.WriteFile(path, res, 0640)
}
