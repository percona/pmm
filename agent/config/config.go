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

// Package config provides access to pmm-agent configuration.
package config

import (
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/utils/nodeinfo"
	"github.com/percona/pmm/version"
)

const (
	pathBaseDefault = "/usr/local/percona/pmm"
	agentTmpPath    = "tmp" // temporary directory to keep exporters' config files, relative to pathBase
	agentDataPath   = "data"
	agentPrefix     = "/agent_id/"
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

// FilteredURL returns URL with redacted password.
func (s *Server) FilteredURL() string {
	u := s.URL()
	if u == nil {
		return ""
	}

	if _, ps := u.User.Password(); ps {
		u.User = url.UserPassword(u.User.Username(), "***")
	}

	// unescape ***; url.unescape and url.encodeUserPassword are not exported, so use strings.Replace
	return strings.ReplaceAll(u.String(), ":%2A%2A%2A@", ":***@")
}

// Paths represents binaries paths configuration.
type Paths struct {
	PathsBase        string `yaml:"paths_base"`
	ExportersBase    string `yaml:"exporters_base"`
	NodeExporter     string `yaml:"node_exporter"`
	MySQLdExporter   string `yaml:"mysqld_exporter"`
	MongoDBExporter  string `yaml:"mongodb_exporter"`
	PostgresExporter string `yaml:"postgres_exporter"`
	ProxySQLExporter string `yaml:"proxysql_exporter"`
	RDSExporter      string `yaml:"rds_exporter"`
	AzureExporter    string `yaml:"azure_exporter"`

	VMAgent string `yaml:"vmagent"`
	Nomad   string `yaml:"nomad"`

	TempDir      string `yaml:"tempdir"`
	NomadDataDir string `yaml:"nomad_data_dir"`

	PTSummary        string `yaml:"pt_summary"`
	PTPGSummary      string `yaml:"pt_pg_summary"`
	PTMySQLSummary   string `yaml:"pt_mysql_summary"`
	PTMongoDBSummary string `yaml:"pt_mongodb_summary"`

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
	NodeType          string
	NodeName          string
	MachineID         string
	Distro            string
	ContainerID       string
	ContainerName     string
	NodeModel         string
	Region            string
	Az                string
	Address           string
	MetricsMode       string
	DisableCollectors string
	CustomLabels      string
	AgentPassword     string

	Force            bool
	SkipRegistration bool
	ExposeExporter   bool
}

// Config represents pmm-agent's configuration.
//
//nolint:maligned
type Config struct { //nolint:musttag
	// no config file there

	ID                             string `yaml:"id"`
	ListenAddress                  string `yaml:"listen-address"`
	ListenPort                     uint16 `yaml:"listen-port"`
	RunnerCapacity                 uint16 `yaml:"runner-capacity,omitempty"`
	RunnerMaxConnectionsPerService uint16 `yaml:"runner-max-connections-per-service,omitempty"`

	Server Server `yaml:"server"`
	Paths  Paths  `yaml:"paths"`
	Ports  Ports  `yaml:"ports"`

	LogLevel string `yaml:"log-level"`
	Debug    bool   `yaml:"debug"`
	Trace    bool   `yaml:"trace"`

	LogLinesCount uint `json:"log-lines-count"`

	WindowConnectedTime time.Duration `yaml:"window-connected-time"`

	Setup Setup `yaml:"-"`
}

// ConfigFileDoesNotExistError error is returned from Get method if configuration file is expected,
// but does not exist.
type ConfigFileDoesNotExistError string //nolint:revive

func (e ConfigFileDoesNotExistError) Error() string {
	return fmt.Sprintf("configuration file %s does not exist", string(e))
}

// getFromCmdLine parses command-line flags, environment variables and configuration file
// (if --config-file/PMM_AGENT_CONFIG_FILE is defined).
// It returns configuration, configuration file path (value of -config-file/PMM_AGENT_CONFIG_FILE, may be empty),
// and any encountered error. That error may be ConfigFileDoesNotExistError if configuration file path is not empty,
// but file itself does not exist. Configuration from command-line flags and environment variables
// is still returned in this case.
func getFromCmdLine(cfg *Config, l *logrus.Entry) (string, error) {
	return get(os.Args[1:], cfg, l)
}

// get is Get for unit tests: it parses args instead of command-line.
func get(args []string, cfg *Config, l *logrus.Entry) (string, error) { //nolint:cyclop
	var configFileF string
	var err error
	// tweak configuration on exit to cover all return points
	defer func() {
		if cfg == nil {
			return
		}

		// set default values
		if strings.HasPrefix(cfg.ID, agentPrefix) {
			l.Warnf("The agent ID '%s' contains a legacy prefix '%s'. It will be used without it.", cfg.ID, agentPrefix)
			cfg.ID, _ = strings.CutPrefix(cfg.ID, agentPrefix)
		}
		if cfg.ListenAddress == "" {
			cfg.ListenAddress = "127.0.0.1"
		}
		if cfg.ListenPort == 0 {
			cfg.ListenPort = 7777
		}
		if cfg.Ports.Min == 0 {
			cfg.Ports.Min = 42000 // for minimal compatibility with PMM Client 1.x firewall rules and documentation
		}
		if cfg.Ports.Max == 0 {
			cfg.Ports.Max = 51999
		}
		if cfg.WindowConnectedTime == 0 {
			cfg.WindowConnectedTime = time.Hour
		}

		for sp, v := range map[*string]string{
			&cfg.Paths.NodeExporter:     "node_exporter",
			&cfg.Paths.MySQLdExporter:   "mysqld_exporter",
			&cfg.Paths.MongoDBExporter:  "mongodb_exporter",
			&cfg.Paths.PostgresExporter: "postgres_exporter",
			&cfg.Paths.ProxySQLExporter: "proxysql_exporter",
			&cfg.Paths.RDSExporter:      "rds_exporter",
			&cfg.Paths.AzureExporter:    "azure_exporter",
			&cfg.Paths.VMAgent:          "vmagent",
			&cfg.Paths.PTSummary:        "tools/pt-summary",
			&cfg.Paths.PTPGSummary:      "tools/pt-pg-summary",
			&cfg.Paths.PTMongoDBSummary: "tools/pt-mongodb-summary",
			&cfg.Paths.PTMySQLSummary:   "tools/pt-mysql-summary",
			&cfg.Paths.Nomad:            "tools/nomad",
		} {
			if *sp == "" {
				*sp = v
			}
		}

		if cfg.Paths.PathsBase == "" {
			cfg.Paths.PathsBase = pathBaseDefault
		}
		if cfg.Paths.ExportersBase == "" {
			cfg.Paths.ExportersBase = filepath.Join(cfg.Paths.PathsBase, "exporters")
		}

		if abs, _ := filepath.Abs(cfg.Paths.PathsBase); abs != "" {
			cfg.Paths.PathsBase = abs
		}
		if abs, _ := filepath.Abs(cfg.Paths.ExportersBase); abs != "" {
			cfg.Paths.ExportersBase = abs
		}

		if cfg.Paths.TempDir == "" {
			cfg.Paths.TempDir = filepath.Join(cfg.Paths.PathsBase, agentTmpPath)
			l.Infof("Temporary directory is not configured and will be set to %s", cfg.Paths.TempDir)
		}

		if cfg.Paths.NomadDataDir == "" {
			cfg.Paths.NomadDataDir = filepath.Join(cfg.Paths.PathsBase, agentTmpPath, "nomad")
			l.Infof("Nomad data directory will default to %s", cfg.Paths.NomadDataDir)
		}

		if !filepath.IsAbs(cfg.Paths.TempDir) {
			cfg.Paths.TempDir = filepath.Join(cfg.Paths.PathsBase, cfg.Paths.TempDir)
			l.Debugf("Temporary directory is configured as %s", cfg.Paths.TempDir)
		}

		for n, sp := range map[string]*string{
			"Percona Toolkit pt-summary":         &cfg.Paths.PTSummary,
			"Percona Toolkit pt-pg-summary":      &cfg.Paths.PTPGSummary,
			"Percona Toolkit pt-mongodb-summary": &cfg.Paths.PTMongoDBSummary,
			"Percona Toolkit pt-mysql-summary":   &cfg.Paths.PTMySQLSummary,
			"Nomad binary":                       &cfg.Paths.Nomad,
		} {
			if !filepath.IsAbs(*sp) {
				*sp = filepath.Join(cfg.Paths.PathsBase, *sp)
				l.Infof("Using %s as a path to %s", *sp, n)
			}
		}

		for n, sp := range map[string]*string{
			"node_exporter":     &cfg.Paths.NodeExporter,
			"mysqld_exporter":   &cfg.Paths.MySQLdExporter,
			"mongodb_exporter":  &cfg.Paths.MongoDBExporter,
			"postgres_exporter": &cfg.Paths.PostgresExporter,
			"proxysql_exporter": &cfg.Paths.ProxySQLExporter,
			"rds_exporter":      &cfg.Paths.RDSExporter,
			"azure_exporter":    &cfg.Paths.AzureExporter,
			"vmagent":           &cfg.Paths.VMAgent,
		} {
			if cfg.Paths.ExportersBase != "" && !filepath.IsAbs(*sp) {
				*sp = filepath.Join(cfg.Paths.ExportersBase, *sp)
			}
			l.Infof("Using %s as a path to %s", *sp, n)
		}

		if cfg.Server.Address != "" {
			if _, _, e := net.SplitHostPort(cfg.Server.Address); e != nil {
				host := cfg.Server.Address
				cfg.Server.Address = net.JoinHostPort(host, "443")
				l.Infof("Updating PMM Server address from %q to %q.", host, cfg.Server.Address)
			}
		}

		// enabled cross-component PMM_DEBUG and PMM_TRACE take priority
		if b, _ := strconv.ParseBool(os.Getenv("PMM_DEBUG")); b {
			cfg.Debug = true
		}
		if b, _ := strconv.ParseBool(os.Getenv("PMM_TRACE")); b {
			cfg.Trace = true
		}
	}()

	// parse command-line flags and environment variables
	app, cfgFileF := Application(cfg)
	if _, err = app.Parse(args); err != nil {
		return configFileF, err
	}
	if *cfgFileF == "" {
		return configFileF, err
	}

	if configFileF, err = filepath.Abs(*cfgFileF); err != nil {
		return configFileF, err
	}
	l.Infof("Loading configuration file %s.", configFileF)
	fileCfg, err := loadFromFile(configFileF)
	if err != nil {
		return configFileF, err
	}

	// re-parse flags into configuration from file
	app, _ = Application(fileCfg)
	if _, err = app.Parse(args); err != nil {
		return configFileF, err
	}

	*cfg = *fileCfg
	return configFileF, nil
}

// Application returns kingpin application that will parse command-line flags and environment variables
// (but not configuration file) into cfg except --config-file/PMM_AGENT_CONFIG_FILE that is returned separately.
func Application(cfg *Config) (*kingpin.Application, *string) {
	app := kingpin.New("pmm-agent", fmt.Sprintf("Version %s", version.Version))
	app.HelpFlag.Short('h')

	app.Command("run", "Run pmm-agent (default command)").Default()

	// All `app` flags should be optional and should not have non-zero default values for:
	// * `pmm-agent setup` to work;
	// * correct configuration file loading.
	// See `get` above for the actual default values.

	configFileF := app.Flag("config-file", "Configuration file path [PMM_AGENT_CONFIG_FILE]").
		Envar("PMM_AGENT_CONFIG_FILE").PlaceHolder("</path/to/pmm-agent.yaml>").String()

	app.Flag("id", "ID of this pmm-agent [PMM_AGENT_ID]").
		Envar("PMM_AGENT_ID").StringVar(&cfg.ID)
	app.Flag("listen-address", "Agent local API address [PMM_AGENT_LISTEN_ADDRESS]").
		Envar("PMM_AGENT_LISTEN_ADDRESS").StringVar(&cfg.ListenAddress)
	app.Flag("listen-port", "Agent local API port [PMM_AGENT_LISTEN_PORT]").
		Envar("PMM_AGENT_LISTEN_PORT").Uint16Var(&cfg.ListenPort)
	app.Flag("runner-capacity", "Agent internal actions/jobs runner capacity [PMM_AGENT_RUNNER_CAPACITY]").
		Envar("PMM_AGENT_RUNNER_CAPACITY").Uint16Var(&cfg.RunnerCapacity)
	app.Flag("runner-max-connections-per-service", "Agent internal action/job runner connection limit per DB instance").
		Envar("PMM_AGENT_RUNNER_MAX_CONNECTIONS_PER_SERVICE").Uint16Var(&cfg.RunnerMaxConnectionsPerService)

	app.Flag("server-address", "PMM Server address [PMM_AGENT_SERVER_ADDRESS]").
		Envar("PMM_AGENT_SERVER_ADDRESS").PlaceHolder("<host:port>").StringVar(&cfg.Server.Address)
	app.Flag("server-username", "Username to connect to PMM Server [PMM_AGENT_SERVER_USERNAME]").
		Envar("PMM_AGENT_SERVER_USERNAME").StringVar(&cfg.Server.Username)
	app.Flag("server-password", "Password to connect to PMM Server [PMM_AGENT_SERVER_PASSWORD]").
		Envar("PMM_AGENT_SERVER_PASSWORD").StringVar(&cfg.Server.Password)
	app.Flag("server-insecure-tls", "Skip PMM Server TLS certificate validation [PMM_AGENT_SERVER_INSECURE_TLS]").
		Envar("PMM_AGENT_SERVER_INSECURE_TLS").BoolVar(&cfg.Server.InsecureTLS)
	// no flag for WithoutTLS - it is only for development and testing

	app.Flag("paths-base", "Base path for exporters/collectors/tools to use [PMM_AGENT_PATHS_BASE]").
		Envar("PMM_AGENT_PATHS_BASE").StringVar(&cfg.Paths.PathsBase)
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
	app.Flag("paths-azure_exporter", "Path to azure_exporter to use [PMM_AGENT_PATHS_AZURE_EXPORTER]").
		Envar("PMM_AGENT_PATHS_AZURE_EXPORTER").StringVar(&cfg.Paths.AzureExporter)
	app.Flag("paths-pt-summary", "Path to pt summary to use [PMM_AGENT_PATHS_PT_SUMMARY]").
		Envar("PMM_AGENT_PATHS_PT_SUMMARY").StringVar(&cfg.Paths.PTSummary)
	app.Flag("paths-pt-pg-summary", "Path to pt-pg-summary to use [PMM_AGENT_PATHS_PT_PG_SUMMARY]").
		Envar("PMM_AGENT_PATHS_PT_PG_SUMMARY").StringVar(&cfg.Paths.PTPGSummary)
	app.Flag("paths-pt-mongodb-summary", "Path to pt mongodb summary to use [PMM_AGENT_PATHS_PT_MONGODB_SUMMARY]").
		Envar("PMM_AGENT_PATHS_PT_MONGODB_SUMMARY").StringVar(&cfg.Paths.PTMongoDBSummary)
	app.Flag("paths-pt-mysql-summary", "Path to pt my sql summary to use [PMM_AGENT_PATHS_PT_MYSQL_SUMMARY]").
		Envar("PMM_AGENT_PATHS_PT_MYSQL_SUMMARY").StringVar(&cfg.Paths.PTMySQLSummary)
	app.Flag("paths-nomad", "Path to nomad binary. Can be overridden using [PMM_AGENT_PATHS_NOMAD]").
		Envar("PMM_AGENT_PATHS_NOMAD").StringVar(&cfg.Paths.Nomad)
	app.Flag("paths-nomad-data-dir", "Nomad data directory [PMM_AGENT_PATHS_NOMAD_DATA_DIR]").
		Envar("PMM_AGENT_PATHS_NOMAD_DATA_DIR").StringVar(&cfg.Paths.NomadDataDir)
	app.Flag("paths-tempdir", "Temporary directory for exporters [PMM_AGENT_PATHS_TEMPDIR]").
		Envar("PMM_AGENT_PATHS_TEMPDIR").StringVar(&cfg.Paths.TempDir)
	// no flag for SlowLogFilePrefix - it is only for development and testing

	app.Flag("ports-min", "Minimal allowed port number for listening sockets [PMM_AGENT_PORTS_MIN]").
		Envar("PMM_AGENT_PORTS_MIN").Uint16Var(&cfg.Ports.Min)
	app.Flag("ports-max", "Maximal allowed port number for listening sockets [PMM_AGENT_PORTS_MAX]").
		Envar("PMM_AGENT_PORTS_MAX").Uint16Var(&cfg.Ports.Max)
	app.Flag("window-connected-time", "Window time for which we track the status of connection between agent and server").
		Envar("PMM_AGENT_WINDOW_CONNECTED_TIME").DurationVar(&cfg.WindowConnectedTime)

	app.Flag("log-level", "Set logging level [PMM_AGENT_LOG_LEVEL]").
		Envar("PMM_AGENT_LOG_LEVEL").EnumVar(&cfg.LogLevel, "debug", "info", "warn", "error", "fatal")
	app.Flag("debug", "Enable debug output [PMM_AGENT_DEBUG]").
		Envar("PMM_AGENT_DEBUG").BoolVar(&cfg.Debug)
	app.Flag("trace", "Enable trace output (implies debug) [PMM_AGENT_TRACE]").
		Envar("PMM_AGENT_TRACE").BoolVar(&cfg.Trace)
	app.Flag("log-lines-count",
		"Take and return N most recent log lines in logs.zip for each: server, every configured exporters and agents [PMM_AGENT_LOG_LINES_COUNT]").
		Envar("PMM_AGENT_LOG_LINES_COUNT").Default("1024").UintVar(&cfg.LogLinesCount)
	jsonF := app.Flag("json", "Enable JSON output").Action(func(*kingpin.ParseContext) error {
		logrus.SetFormatter(&logrus.JSONFormatter{}) // with levels and timestamps always present
		return nil
	}).Bool()

	app.Flag("version", "Show application version").Short('v').Action(func(*kingpin.ParseContext) error {
		// We use fmt instead of log package to provide proper output for --json flag.
		if *jsonF {
			fmt.Println(version.FullInfoJSON()) //nolint:forbidigo
		} else {
			fmt.Println(version.FullInfo()) //nolint:forbidigo
		}
		os.Exit(0)

		return nil
	}).Bool()

	setupCmd := app.Command("setup", "Configure local pmm-agent")
	nodeinfo := nodeinfo.Get()

	if nodeinfo.PublicAddress == "" {
		help := "Node address [PMM_AGENT_SETUP_NODE_ADDRESS]"
		setupCmd.Arg("node-address", help).Required().
			Envar("PMM_AGENT_SETUP_NODE_ADDRESS").StringVar(&cfg.Setup.Address)
	} else {
		help := fmt.Sprintf("Node address (autodetected default: %s) [PMM_AGENT_SETUP_NODE_ADDRESS]", nodeinfo.PublicAddress)
		setupCmd.Arg("node-address", help).Default(nodeinfo.PublicAddress).
			Envar("PMM_AGENT_SETUP_NODE_ADDRESS").StringVar(&cfg.Setup.Address)
	}

	nodeTypeKeys := []string{"generic", "container"}
	nodeTypeDefault := "generic"
	if nodeinfo.Container {
		nodeTypeDefault = "container"
	}
	nodeTypeHelp := fmt.Sprintf("Node type, one of: %s (default: %s) [PMM_AGENT_SETUP_NODE_TYPE]", strings.Join(nodeTypeKeys, ", "), nodeTypeDefault)
	setupCmd.Arg("node-type", nodeTypeHelp).Default(nodeTypeDefault).
		Envar("PMM_AGENT_SETUP_NODE_TYPE").EnumVar(&cfg.Setup.NodeType, nodeTypeKeys...)

	hostname, _ := os.Hostname()
	nodeNameHelp := fmt.Sprintf("Node name (autodetected default: %s) [PMM_AGENT_SETUP_NODE_NAME]", hostname)
	setupCmd.Arg("node-name", nodeNameHelp).Default(hostname).
		Envar("PMM_AGENT_SETUP_NODE_NAME").StringVar(&cfg.Setup.NodeName)

	var defaultMachineID string
	if nodeinfo.MachineID != "" {
		defaultMachineID = nodeinfo.MachineID
	}
	setupCmd.Flag("machine-id", "Node machine-id (default is autodetected) [PMM_AGENT_SETUP_MACHINE_ID]").Default(defaultMachineID).
		Envar("PMM_AGENT_SETUP_MACHINE_ID").StringVar(&cfg.Setup.MachineID)
	setupCmd.Flag("distro", "Node OS distribution (default is autodetected) [PMM_AGENT_SETUP_DISTRO]").Default(nodeinfo.Distro).
		Envar("PMM_AGENT_SETUP_DISTRO").StringVar(&cfg.Setup.Distro)
	setupCmd.Flag("container-id", "Container ID [PMM_AGENT_SETUP_CONTAINER_ID]").
		Envar("PMM_AGENT_SETUP_CONTAINER_ID").StringVar(&cfg.Setup.ContainerID)
	setupCmd.Flag("container-name", "Container name [PMM_AGENT_SETUP_CONTAINER_NAME]").
		Envar("PMM_AGENT_SETUP_CONTAINER_NAME").StringVar(&cfg.Setup.ContainerName)
	setupCmd.Flag("node-model", "Node model [PMM_AGENT_SETUP_NODE_MODEL]").
		Envar("PMM_AGENT_SETUP_NODE_MODEL").StringVar(&cfg.Setup.NodeModel)
	setupCmd.Flag("region", "Node region [PMM_AGENT_SETUP_REGION]").
		Envar("PMM_AGENT_SETUP_REGION").StringVar(&cfg.Setup.Region)
	setupCmd.Flag("az", "Node availability zone [PMM_AGENT_SETUP_AZ]").
		Envar("PMM_AGENT_SETUP_AZ").StringVar(&cfg.Setup.Az)

	setupCmd.Flag("force", "Remove Node with that name with all dependent Services and Agents if one exist [PMM_AGENT_SETUP_FORCE]").
		Envar("PMM_AGENT_SETUP_FORCE").BoolVar(&cfg.Setup.Force)
	setupCmd.Flag("skip-registration", "Skip registration on PMM Server [PMM_AGENT_SETUP_SKIP_REGISTRATION]").
		Envar("PMM_AGENT_SETUP_SKIP_REGISTRATION").BoolVar(&cfg.Setup.SkipRegistration)
	setupCmd.Flag("metrics-mode", "Metrics flow mode for agents node-exporter, can be push - agent will push metrics,"+
		"pull - server scrape metrics from agent  or auto - chosen by server. [PMM_AGENT_SETUP_METRICS_MODE]").
		Envar("PMM_AGENT_SETUP_METRICS_MODE").Default("auto").EnumVar(&cfg.Setup.MetricsMode, "auto", "push", "pull")
	setupCmd.Flag("disable-collectors", "Comma-separated list of collector names to exclude from exporter. [PMM_AGENT_SETUP_METRICS_MODE]").
		Envar("PMM_AGENT_SETUP_DISABLE_COLLECTORS").Default("").StringVar(&cfg.Setup.DisableCollectors)
	setupCmd.Flag("custom-labels", "Custom labels [PMM_AGENT_SETUP_CUSTOM_LABELS]").
		Envar("PMM_AGENT_SETUP_CUSTOM_LABELS").StringVar(&cfg.Setup.CustomLabels)
	setupCmd.Flag("agent-password", "Custom password for /metrics endpoint [PMM_AGENT_SETUP_NODE_PASSWORD]").
		Envar("PMM_AGENT_SETUP_NODE_PASSWORD").StringVar(&cfg.Setup.AgentPassword)
	setupCmd.Flag("expose-exporter", "Expose the address of the agent's node-exporter publicly on 0.0.0.0").
		Envar("PMM_AGENT_EXPOSE_EXPORTER").BoolVar(&cfg.Setup.ExposeExporter)

	return app, configFileF
}

// loadFromFile loads configuration from file.
// As a special case, if file does not exist, it returns ConfigFileDoesNotExistError.
// Other errors are returned if file exists, but configuration can't be loaded due to permission problems,
// YAML parsing problems, etc.
func loadFromFile(path string) (*Config, error) {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil, ConfigFileDoesNotExistError(path)
	}

	b, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
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
	return os.WriteFile(path, res, 0o640) //nolint:gosec
}

// IsWritable checks if specified path is writable.
func IsWritable(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		// File doesn't exist, check if folder is writable.
		if errors.Is(err, fs.ErrNotExist) {
			return unix.Access(filepath.Dir(path), unix.W_OK)
		}
		return err
	}
	return unix.Access(path, unix.W_OK)
}
