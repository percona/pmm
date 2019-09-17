// pmm-agent
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

// Package config provides access to pmm-agent configuration.
package config

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/percona/pmm/utils/nodeinfo"
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

	WithoutTLS bool `yaml:"without-tls,omitempty"` // for development and testing
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
	ExportersBase    string `yaml:"exporters_base"`
	NodeExporter     string `yaml:"node_exporter"`
	MySQLdExporter   string `yaml:"mysqld_exporter"`
	MongoDBExporter  string `yaml:"mongodb_exporter"`
	PostgresExporter string `yaml:"postgres_exporter"`
	ProxySQLExporter string `yaml:"proxysql_exporter"`

	TempDir string `yaml:"tempdir"`

	SlowLogFilePrefix string `yaml:"slowlog_file_prefix,omitempty"` // for development and testing
}

// Ports represents ports configuration.
type Ports struct {
	Min uint16 `yaml:"min"`
	Max uint16 `yaml:"max"`
}

// Setup contains `pmm-agent setup` flag and argument values.
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

	Force            bool
	SkipRegistration bool
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
	return get(os.Args[1:], l)
}

// get is Get for unit tests: it parses args instead of command-line.
func get(args []string, l *logrus.Entry) (cfg *Config, configFileF string, err error) {
	// tweak configuration on exit to cover all return points
	defer func() {
		if cfg == nil {
			return
		}

		// set default values
		if cfg.ListenPort == 0 {
			cfg.ListenPort = 7777
		}
		if cfg.Ports.Min == 0 {
			cfg.Ports.Min = 42000 // for minimal compatibility with PMM Client 1.x firewall rules and documentation
		}
		if cfg.Ports.Max == 0 {
			cfg.Ports.Max = 51999
		}
		for sp, v := range map[*string]string{
			&cfg.Paths.ExportersBase:    "/usr/local/percona/pmm2/exporters",
			&cfg.Paths.NodeExporter:     "node_exporter",
			&cfg.Paths.MySQLdExporter:   "mysqld_exporter",
			&cfg.Paths.MongoDBExporter:  "mongodb_exporter",
			&cfg.Paths.PostgresExporter: "postgres_exporter",
			&cfg.Paths.ProxySQLExporter: "proxysql_exporter",
			&cfg.Paths.TempDir:          os.TempDir(),
		} {
			if *sp == "" {
				*sp = v
			}
		}

		if cfg.Paths.ExportersBase != "" {
			if abs, _ := filepath.Abs(cfg.Paths.ExportersBase); abs != "" {
				cfg.Paths.ExportersBase = abs
			}
		}

		for _, sp := range []*string{
			&cfg.Paths.NodeExporter,
			&cfg.Paths.MySQLdExporter,
			&cfg.Paths.MongoDBExporter,
			&cfg.Paths.PostgresExporter,
			&cfg.Paths.ProxySQLExporter,
		} {
			if cfg.Paths.ExportersBase != "" && !filepath.IsAbs(*sp) {
				*sp = filepath.Join(cfg.Paths.ExportersBase, *sp)
			}
			l.Infof("Using %s", *sp)
		}

		if cfg.Server.Address != "" {
			if _, _, e := net.SplitHostPort(cfg.Server.Address); e != nil {
				host := cfg.Server.Address
				cfg.Server.Address = net.JoinHostPort(host, "443")
				l.Infof("Updating PMM Server address from %q to %q.", host, cfg.Server.Address)
			}
		}
	}()

	// parse command-line flags and environment variables
	cfg = new(Config)
	app, cfgFileF := Application(cfg)
	if _, err = app.Parse(args); err != nil {
		return
	}
	if *cfgFileF == "" {
		return
	}

	if configFileF, err = filepath.Abs(*cfgFileF); err != nil {
		return
	}
	l.Infof("Loading configuration file %s.", configFileF)
	fileCfg, err := loadFromFile(configFileF)
	if err != nil {
		return
	}

	// re-parse flags into configuration from file
	app, _ = Application(fileCfg)
	if _, err = app.Parse(args); err != nil {
		return
	}

	cfg = fileCfg
	return //nolint:nakedret
}

// Application returns kingpin application that will parse command-line flags and environment variables
// (but not configuration file) into cfg except --config-file/PMM_AGENT_CONFIG_FILE that is returned separately.
func Application(cfg *Config) (*kingpin.Application, *string) {
	app := kingpin.New("pmm-agent", fmt.Sprintf("Version %s", version.Version))
	app.HelpFlag.Short('h')
	app.Version(version.FullInfo())

	app.Command("run", "Run pmm-agent (default command)").Default()

	// All `app` flags should be optional and should not have non-zero default values for:
	// * `pmm-agent setup` to work;
	// * correct configuration file loading.
	// See `get` above for the actual default values.

	configFileF := app.Flag("config-file", "Configuration file path [PMM_AGENT_CONFIG_FILE]").
		Envar("PMM_AGENT_CONFIG_FILE").PlaceHolder("</path/to/pmm-agent.yaml>").String()

	app.Flag("id", "ID of this pmm-agent [PMM_AGENT_ID]").
		Envar("PMM_AGENT_ID").PlaceHolder("</agent_id/...>").StringVar(&cfg.ID)
	app.Flag("listen-port", "Agent local API port [PMM_AGENT_LISTEN_PORT]").
		Envar("PMM_AGENT_LISTEN_PORT").Uint16Var(&cfg.ListenPort)

	app.Flag("server-address", "PMM Server address [PMM_AGENT_SERVER_ADDRESS]").
		Envar("PMM_AGENT_SERVER_ADDRESS").PlaceHolder("<host:port>").StringVar(&cfg.Server.Address)
	app.Flag("server-username", "Username to connect to PMM Server [PMM_AGENT_SERVER_USERNAME]").
		Envar("PMM_AGENT_SERVER_USERNAME").StringVar(&cfg.Server.Username)
	app.Flag("server-password", "Password to connect to PMM Server [PMM_AGENT_SERVER_PASSWORD]").
		Envar("PMM_AGENT_SERVER_PASSWORD").StringVar(&cfg.Server.Password)
	app.Flag("server-insecure-tls", "Skip PMM Server TLS certificate validation [PMM_AGENT_SERVER_INSECURE_TLS]").
		Envar("PMM_AGENT_SERVER_INSECURE_TLS").BoolVar(&cfg.Server.InsecureTLS)
	// no flag for WithoutTLS - it is only for development and testing

	app.Flag("paths-exporters_base", "Base path for exporters to use [PMM_AGENT_PATHS_EXPORTERS_BASE]").
		Envar("PMM_AGENT_PATHS_EXPORTERS_BASE").StringVar(&cfg.Paths.ExportersBase)
	app.Flag("paths-node_exporter", "Path to node_exporter to use [PMM_AGENT_PATHS_NODE_EXPORTER]").
		Envar("PMM_AGENT_PATHS_NODE_EXPORTER").StringVar(&cfg.Paths.NodeExporter)
	app.Flag("paths-mysqld_exporter", "Path to mysqld_exporter to use [PMM_AGENT_PATHS_MYSQLD_EXPORTER]").
		Envar("PMM_AGENT_PATHS_MYSQLD_EXPORTER").StringVar(&cfg.Paths.MySQLdExporter)
	app.Flag("paths-mongodb_exporter", "Path to mongodb_exporter to use [PMM_AGENT_PATHS_MONGODB_EXPORTER]").
		Envar("PMM_AGENT_PATHS_MONGODB_EXPORTER").StringVar(&cfg.Paths.MongoDBExporter)
	app.Flag("paths-postgres_exporter", "Path to postgres_exporter to use [PMM_AGENT_PATHS_POSTGRES_EXPORTER]").
		Envar("PMM_AGENT_PATHS_POSTGRES_EXPORTER").StringVar(&cfg.Paths.PostgresExporter)
	app.Flag("paths-proxysql_exporter", "Path to proxysql_exporter to use [PMM_AGENT_PATHS_PROXYSQL_EXPORTER]").
		Envar("PMM_AGENT_PATHS_PROXYSQL_EXPORTER").StringVar(&cfg.Paths.ProxySQLExporter)
	app.Flag("paths-tempdir", "Temporary directory for exporters [PMM_AGENT_PATHS_TEMPDIR]").
		Envar("PMM_AGENT_PATHS_TEMPDIR").StringVar(&cfg.Paths.TempDir)
	// no flag for SlowLogFilePrefix - it is only for development and testing

	app.Flag("ports-min", "Minimal allowed port number for listening sockets [PMM_AGENT_PORTS_MIN]").
		Envar("PMM_AGENT_PORTS_MIN").Uint16Var(&cfg.Ports.Min)
	app.Flag("ports-max", "Maximal allowed port number for listening sockets [PMM_AGENT_PORTS_MAX]").
		Envar("PMM_AGENT_PORTS_MAX").Uint16Var(&cfg.Ports.Max)

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
	nodeTypeDefault := "generic"
	if nodeinfo.Container {
		nodeTypeDefault = "container"
	}
	nodeTypeHelp := fmt.Sprintf("Node type, one of: %s (default: %s)", strings.Join(nodeTypeKeys, ", "), nodeTypeDefault)
	setupCmd.Arg("node-type", nodeTypeHelp).Default(nodeTypeDefault).EnumVar(&cfg.Setup.NodeType, nodeTypeKeys...)

	hostname, _ := os.Hostname()
	nodeNameHelp := fmt.Sprintf("Node name (autodetected default: %s)", hostname)
	setupCmd.Arg("node-name", nodeNameHelp).Default(hostname).StringVar(&cfg.Setup.NodeName)

	var defaultMachineID string
	if nodeinfo.MachineID != "" {
		defaultMachineID = "/machine_id/" + nodeinfo.MachineID
	}
	setupCmd.Flag("machine-id", "Node machine-id (default is autodetected)").Default(defaultMachineID).StringVar(&cfg.Setup.MachineID)
	setupCmd.Flag("distro", "Node OS distribution (default is autodetected)").Default(nodeinfo.Distro).StringVar(&cfg.Setup.Distro)
	setupCmd.Flag("container-id", "Container ID").StringVar(&cfg.Setup.ContainerID)
	setupCmd.Flag("container-name", "Container name").StringVar(&cfg.Setup.ContainerName)
	setupCmd.Flag("node-model", "Node model").StringVar(&cfg.Setup.NodeModel)
	setupCmd.Flag("region", "Node region").StringVar(&cfg.Setup.Region)
	setupCmd.Flag("az", "Node availability zone").StringVar(&cfg.Setup.Az)
	// TODO setupCmd.Flag("custom-labels", "Custom user-assigned labels").StringVar(&cfg.Setup.CustomLabels)

	setupCmd.Flag("force", "Remove Node with that name with all dependent Services and Agents if one exist").BoolVar(&cfg.Setup.Force)
	setupCmd.Flag("skip-registration", "Skip registration on PMM Server").BoolVar(&cfg.Setup.SkipRegistration)

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
