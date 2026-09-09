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

//nolint:forbidigo
package commands

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"

	"github.com/percona/pmm/agent/config"
	agent_local "github.com/percona/pmm/api/agentlocal/v1/json/client/agent_local_service"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

// registrationState is what `pmm-agent setup` knows about the Node registration the Agent holds.
type registrationState int

const (
	// PMM Server could not be asked, or answered something this pmm-agent could not interpret, and the
	// registration is kept. It is the zero value so that anything which does not reach a verdict of its
	// own leaves the Node alone: only a clear answer is grounds for registering it again.
	registrationUnverified registrationState = iota
	// The Node has to be registered: the Agent holds no ID, it is pointed at another PMM Server,
	// --force was given, or PMM Server does not know the Agent on this Node.
	registrationMissing
	// PMM Server knows the Agent on this Node.
	registrationConfirmed
	// PMM Server has the Agent on a Node with another name, so registering would add a second Node
	// instead of replacing that one. Only the operator can say which of the two is meant.
	registrationConflict
)

// registrationCheck asks PMM Server about the registration of the Agent described by cfg.
type registrationCheck func(cfg *config.Config, l *logrus.Entry) registrationState

// agentLookup returns the Node which PMM Server has the given Agent registered on.
type agentLookup func(agentID string) (serverNode, error)

// checkRegistrationOnServer asks PMM Server whether it knows this Agent on this Node.
func checkRegistrationOnServer(cfg *config.Config, l *logrus.Entry) registrationState {
	u := cfg.Server.URL()
	if u == nil {
		// register reports the missing server address with an actionable message
		return registrationMissing
	}
	setServerTransport(u, cfg.Server.InsecureTLS, l)

	return checkRegistration(cfg, serverNodeOfAgent)
}

// checkRegistration turns what PMM Server says about the Agent into the state of its registration. The
// server may have been reinstalled, or restored from a backup taken before the Agent was registered,
// leaving the Agent with an ID nothing recognizes. Anything short of a clear answer is not an answer: an
// Agent has to be able to start while PMM Server has no leader yet, and a reply this pmm-agent cannot
// interpret says as little as no reply at all, so the registration is kept in both cases.
func checkRegistration(cfg *config.Config, lookup agentLookup) registrationState {
	node, err := lookup(cfg.ID)
	switch {
	case errors.Is(err, errAgentNotFound):
		fmt.Printf("PMM Server at %s does not know pmm-agent %s, registering the Node again.\n", cfg.Server.Address, cfg.ID)
		return registrationMissing
	case errors.Is(err, errCredentialsRejected):
		fmt.Printf("PMM Server at %s does not accept the credentials of pmm-agent %s, registering the Node again.\n",
			cfg.Server.Address, cfg.ID)
		return registrationMissing
	case err != nil:
		fmt.Printf("Failed to check the registration of pmm-agent %s with %s: %s.\n", cfg.ID, cfg.Server.Address, err)
		return registrationUnverified
	case cfg.Setup.NodeName != "" && node.Name != cfg.Setup.NodeName:
		// Registering would not replace that Node. Without --region the address is not checked for
		// uniqueness, so PMM Server would create a second Node under the given name and leave the
		// registered one behind with every Service on it, monitored by nothing. A configuration file
		// copied from another host looks like this too, and there --force is the answer.
		fmt.Printf("PMM Server at %s has pmm-agent %s on Node %s, not on %s.\n"+
			"Re-run with %s as the Node name argument to keep that Node together with its Services.\n"+
			"Use --force to register %s as a new Node instead: Node %s and its Services stay on PMM Server,"+
			" monitored by nothing.\n",
			cfg.Server.Address, cfg.ID, node.Name, cfg.Setup.NodeName, node.Name, cfg.Setup.NodeName, node.Name)
		return registrationConflict
	default:
		reportRegisteredNode(cfg, node)
		return registrationConfirmed
	}
}

// reportRegisteredNode reports what PMM Server holds about the Node where it no longer matches what
// setup was given. Both the address and the type of a Node are only set when it is registered, and
// nothing updates them afterwards, so a difference is kept rather than applied.
func reportRegisteredNode(cfg *config.Config, node serverNode) {
	// The address is what PMM Server keeps scraping in pull metrics mode.
	if node.Address != "" && cfg.Setup.Address != "" && node.Address != cfg.Setup.Address {
		fmt.Printf("Node %s is registered with address %s, not %s. The registered address is kept;"+
			" it is the address PMM Server scrapes in pull metrics mode.\n"+
			"Use --force to register the Node with %s, which removes the registered Node together with every"+
			" Service on it.\n",
			node.Name, node.Address, cfg.Setup.Address, cfg.Setup.Address)
	}

	// The type is a positional argument with a default, so it is always given and cannot be reported as
	// an unapplied flag. Saying nothing would let an operator read the exit code as the type having
	// changed.
	if node.Type != "" && cfg.Setup.NodeType != "" && node.Type != cfg.Setup.NodeType {
		fmt.Printf("Node %s is registered as a %s Node, not %s. The registered type is kept.\n"+
			"Use --force to register the Node as %s, which removes the registered Node together with every"+
			" Service on it.\n",
			node.Name, node.Type, cfg.Setup.NodeType, cfg.Setup.NodeType)
	}
}

// registrationOf reports what `pmm-agent setup` is to do about the Node registration. An Agent which
// already holds an ID is registered with PMM Server, and registering it again makes the server drop
// the Node together with every Service on it. That is only done on demand, when the Agent is being
// pointed at a different PMM Server, or when the server does not know the Agent on this Node.
// A nil fileCfg means that the Agent runs with no configuration file yet.
func registrationOf(cfg, fileCfg *config.Config, check registrationCheck, l *logrus.Entry) registrationState {
	if cfg.ID == "" || cfg.Setup.Force || fileCfg == nil {
		return registrationMissing
	}
	// The file may hold the address without the default port, unlike the parsed configuration.
	if !strings.EqualFold(cfg.Server.NormalizedAddress(), fileCfg.Server.NormalizedAddress()) {
		return registrationMissing
	}

	return check(runningCredentials(cfg, fileCfg), l)
}

// runningCredentials returns cfg holding the credentials the Agent runs with, so that the registration
// is checked with the stored service token rather than with the credentials given to setup. Only then
// does a confirmed registration mean that the Agent can still reach PMM Server on its own: a token the
// server no longer accepts registers the Node again, with the credentials given to setup.
func runningCredentials(cfg, fileCfg *config.Config) *config.Config {
	if fileCfg.Server.Password == "" {
		return cfg
	}

	c := *cfg
	c.Server.Username = fileCfg.Server.Username
	c.Server.Password = fileCfg.Server.Password

	return &c
}

// registeredConfig returns the configuration file the Agent runs with, or nil when there is none.
// `pmm-admin config` runs `pmm-agent setup` without --config-file, so the file, whose path is only known
// from the running pmm-agent, has not been loaded yet in that case. An error means that the file is
// there but could not be read, which says nothing about the registration either way.
func registeredConfig(configFilepath string, cfg *config.Config) (*config.Config, error) {
	fileCfg, err := config.LoadFromFile(configFilepath, &cfg.Encryption)
	var e config.ConfigFileDoesNotExistError
	switch {
	case err == nil:
		return fileCfg, nil
	case errors.As(err, &e):
		return nil, nil //nolint:nilnil
	default:
		return nil, err
	}
}

// keepRegistration keeps the credentials the Agent runs with: those given to `pmm-agent setup` only
// serve to register, which replaces them with a service token. It also reports the settings which
// describe the Node on PMM Server, because they are only applied when the Node is registered.
func keepRegistration(cfg, fileCfg *config.Config) {
	flags := append(unappliedSetupFlags(&cfg.Setup), unappliedCredentials(cfg, fileCfg)...)

	if fileCfg.Server.Password != "" {
		cfg.Server.Username = fileCfg.Server.Username
		cfg.Server.Password = fileCfg.Server.Password
	}

	if len(flags) > 0 {
		fmt.Printf("Settings %s only take effect when the Node is registered.\n", strings.Join(flags, ", "))
	}
}

// unappliedCredentials names the credentials given to setup which the Agent is not going to use, because
// the service token it already holds is what reaches PMM Server. Keeping them is right, and saying so is
// what stops a mistyped password from passing for an accepted one.
func unappliedCredentials(cfg, fileCfg *config.Config) []string {
	if fileCfg.Server.Password == "" || cfg.Server.Password == "" || cfg.Server.Password == fileCfg.Server.Password {
		return nil
	}

	return []string{"--server-username", "--server-password"}
}

// storesConfig reports whether the configuration `pmm-agent setup` assembled is the one to store. It
// holds what the configuration file has only when setup loaded that file, and `pmm-admin config` runs
// setup without --config-file: there the configuration is the flags and the defaults alone, missing
// every setting the file holds beyond them, from the ports range to the /proc/mounts path. Registering
// replaces the file by design, but keeping a registration must leave those settings alone.
func storesConfig(registered, loadedFromFile bool, fileCfg *config.Config) bool {
	return registered || loadedFromFile || fileCfg == nil
}

// unappliedSetupFlags lists the given `pmm-agent setup` flags which describe the Node on PMM Server.
func unappliedSetupFlags(s *config.Setup) []string {
	var flags []string
	for _, f := range []struct {
		name  string
		given bool
	}{
		{"--node-model", s.NodeModel != ""},
		{"--region", s.Region != ""},
		{"--az", s.Az != ""},
		{"--metrics-mode", s.MetricsMode != "" && s.MetricsMode != "auto"},
		{"--disable-collectors", s.DisableCollectors != ""},
		{"--custom-labels", s.CustomLabels != ""},
		{"--agent-password", s.AgentPassword != ""},
		{"--expose-exporter", s.ExposeExporter},
		{"--container-id", s.ContainerID != ""},
		{"--container-name", s.ContainerName != ""},
	} {
		if f.given {
			flags = append(flags, f.name)
		}
	}

	return flags
}

// Setup implements `pmm-agent setup` command.
func Setup() {
	/*
		Cases when this code breaks:

		1. $ pmm-agent run --listen-port=12345  # non-default port, no config file
		   $ pmm-agent setup
		   We should stop setup (pmm-agent is running without config file), but we don't.

		2. $ pmm-agent run --server-address=1.2.3.4:443 --config-file=pmm-agent.yaml
		   $ pmm-agent setup
		   We will use server address from config, not from run's flag.
	*/

	l := logrus.WithField("component", "setup")

	configStorage := config.NewStorage(nil)
	loadedFilepath, err := configStorage.Reload(l)

	var e config.ConfigFileDoesNotExistError
	if err != nil && !errors.As(err, &e) {
		fmt.Printf("Failed to load configuration: %s.\n", err)
		os.Exit(1)
	}

	cfg := configStorage.Get()
	setLocalTransport(cfg.ListenAddress, cfg.ListenPort, l)

	configFilepath, running := checkStatus(loadedFilepath, l)

	fileCfg, err := registeredConfig(configFilepath, cfg)
	if err != nil {
		// A file encrypted with a key setup was not given reads exactly like a damaged one, and the format
		// carries nothing to tell them apart. --force answers neither: not knowing what the file says is
		// not knowing whether this Agent is registered, so registering would take the Node over together
		// with every Service on it, and it would store a plaintext file over one which may be encrypted,
		// which the Agent, started with its key, then fails to load.
		fmt.Printf("Failed to read the configuration file %s: %s.\n"+
			"If the file is encrypted, re-run with --config-file-key-file, or set PMM_AGENT_CONFIG_FILE_KEY_FILE."+
			" Otherwise repair the file, or remove it to register this Node afresh.\n",
			configFilepath, err)
		os.Exit(1)
	}

	if cfg.ID == "" && cfg.Setup.SkipRegistration {
		fmt.Printf("Can't skip registration: pmm-agent ID is empty.\n")
		os.Exit(1)
	}

	// The ID the Agent runs with, so that its registration can be checked. It is taken after the
	// --skip-registration guard, which asks about the ID given to setup: reading one from the file would
	// let that command through to store a configuration assembled without the file it came from.
	if cfg.ID == "" && fileCfg != nil {
		cfg.ID = fileCfg.ID
	}

	err = config.IsWritable(configFilepath)
	if err != nil {
		fmt.Printf("Config file %s is not writable: %v.\n", configFilepath, err)
		os.Exit(1)
	}

	registered := false
	if !cfg.Setup.SkipRegistration {
		switch registrationOf(cfg, fileCfg, checkRegistrationOnServer, l) {
		case registrationMissing:
			register(cfg, l)
			registered = true
		case registrationConfirmed:
			fmt.Printf("Node is already registered with %s, pmm-agent ID is %s. Use --force to register it again.\n",
				cfg.Server.Address, cfg.ID)
			keepRegistration(cfg, fileCfg)
		case registrationUnverified:
			// Deliberately still a success: an Agent has to be able to set itself up while PMM Server has
			// no leader yet, and failing here would take an installation down with the outage. It goes to
			// stderr, and says what was not done, so that it does not read as a confirmed registration to
			// whoever is watching the output.
			fmt.Fprintf(os.Stderr, "WARNING: PMM Server at %s did not confirm that pmm-agent %s is registered."+
				" Nothing was verified and nothing was changed; the existing registration is kept."+
				" Re-run once PMM Server answers, or use --force to register the Node again.\n",
				cfg.Server.Address, cfg.ID)
			keepRegistration(cfg, fileCfg)
		case registrationConflict:
			// checkRegistrationOnServer reported which Node PMM Server has and what to do about it.
			os.Exit(1)
		}
	}

	if !storesConfig(registered, loadedFilepath != "", fileCfg) {
		fmt.Printf("Configuration file %s is left unchanged.\n", configFilepath)
		return
	}

	cfg.ProcMountsPath = cfg.Setup.ProcMountsPath

	err = config.SaveToFile(configFilepath, cfg, "Updated by `pmm-agent setup`.")
	if err != nil {
		fmt.Printf("Failed to write configuration file %s: %s.\n", configFilepath, err)
		os.Exit(1)
	}
	fmt.Printf("Configuration file %s updated.\n", configFilepath)

	if !running {
		fmt.Printf("Please start pmm-agent: `pmm-agent --config-file=%s`.\n", configFilepath)
		return
	}

	reload(l)

	checkStatus(configFilepath, l)
}

func checkStatus(configFilepath string, l *logrus.Entry) (string, bool) {
	fmt.Printf("Checking local pmm-agent status...\n")
	status, err := localStatus()
	l.Debugf("Status error: %#v", err)
	switch err := err.(type) { //nolint:errorlint
	case nil:
		if status.ConfigFilepath == "" {
			fmt.Printf("pmm-agent is running but does not read configuration from the file. " +
				"Please restart it with --config-file flag.\n")
			os.Exit(1)
		}
		if configFilepath != "" && status.ConfigFilepath != configFilepath {
			fmt.Printf("pmm-agent is running and reads configuration from %s. "+
				"Please re-run `pmm-agent setup` without --config-file flag.\n", status.ConfigFilepath)
			os.Exit(1)
		}
		fmt.Printf("pmm-agent is running.\n")
		return status.ConfigFilepath, true

	case *agent_local.StatusDefault:
		msg := fmt.Sprintf("HTTP code %d", err.Code())
		if err.Payload != nil {
			msg = fmt.Sprintf("%s (gRPC code %d, HTTP code %d)", err.Payload.Message, err.Payload.Code, err.Code())
		}
		fmt.Printf("pmm-agent is running, but status check failed: %s.\n", msg)
		os.Exit(1)
		panic("not reached")

	default:
		if configFilepath == "" {
			fmt.Printf("pmm-agent is not running. Please re-run `pmm-agent setup` with --config-file flag.\n")
			os.Exit(1)
		}
		fmt.Printf("pmm-agent is not running.\n")
		return configFilepath, false
	}
}

func register(cfg *config.Config, l *logrus.Entry) {
	fmt.Printf("Registering pmm-agent on PMM Server...\n")

	u := cfg.Server.URL()
	if u == nil {
		fmt.Printf("Can't construct PMM Server URL. Please re-run with --server-address flag.\n")
		os.Exit(1)
	}

	setServerTransport(u, cfg.Server.InsecureTLS, l)
	agentID, token, err := serverRegister(&cfg.Setup)
	l.Debugf("Register error: %#v", err)
	if err != nil {
		msg := err.Error()
		e, ok := errors.AsType[*mservice.RegisterNodeDefault](err)
		if ok {
			msg = e.Payload.Message + ""
			switch e.Code() {
			case http.StatusConflict:
				msg += " If you want override node, use --force option"
			case http.StatusUnauthorized, http.StatusForbidden:
				msg += "\nPlease check username and password"
			}
		}
		if _, ok := err.(nginxError); ok { //nolint:errorlint
			msg += ".\nPlease check pmm-managed logs."
		}

		fmt.Printf("Failed to register pmm-agent on PMM Server: %s.\n", msg)
		os.Exit(1)
	}
	cfg.ID = agentID
	if token != "" {
		cfg.Server.Username = "service_token"
		cfg.Server.Password = token
	} else {
		l.Info("PMM Server responded with an empty service token. Consider upgrading PMM Server to the latest version.")
	}
	fmt.Printf("Registered.\n")
}

func reload(l *logrus.Entry) {
	fmt.Printf("Reloading pmm-agent configuration...\n")

	// sync error handling with Reload API method
	err := localReload()
	l.Debugf("Reload error: %#v", err)
	reloadErr, ok := errors.AsType[*agent_local.ReloadDefault](err)
	if ok && reloadErr.Code() == int(codes.FailedPrecondition) {
		fmt.Printf("Failed to reload configuration: %s.\n", reloadErr.Payload.Message)
		os.Exit(1)
	}

	// wait up to 5 seconds for pmm-agent to reload itself
	for range 5 {
		time.Sleep(time.Second)
		_, err = localStatus()
		l.Debugf("Status error: %#v", err)
		if err == nil {
			fmt.Printf("Configuration reloaded.\n")
			return
		}
	}
}
