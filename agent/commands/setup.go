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
	// The Node has to be registered: the Agent holds no ID, it is pointed at another PMM Server,
	// --force was given, or PMM Server does not know the Agent on this Node.
	registrationMissing registrationState = iota
	// PMM Server knows the Agent on this Node.
	registrationConfirmed
	// PMM Server could not be asked, and the registration is kept.
	registrationUnverified
)

// registrationCheck asks PMM Server about the registration of the Agent described by cfg.
type registrationCheck func(cfg *config.Config, l *logrus.Entry) registrationState

// checkRegistrationOnServer asks PMM Server whether it knows this Agent on this Node. The server may
// have been reinstalled, or restored from a backup taken before the Agent was registered, leaving the
// Agent with an ID nothing recognizes. The configuration file may also have been copied from another
// host, whose Node the Agent is registered on. An unreachable server is not an answer: an Agent has to
// be able to start while PMM Server has no leader yet, so its registration is kept in that case.
func checkRegistrationOnServer(cfg *config.Config, l *logrus.Entry) registrationState {
	u := cfg.Server.URL()
	if u == nil {
		// register reports the missing server address with an actionable message
		return registrationMissing
	}
	setServerTransport(u, cfg.Server.InsecureTLS, l)

	serverNode, err := serverNodeOfAgent(cfg.ID)
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
	case cfg.Setup.NodeName != "" && serverNode != cfg.Setup.NodeName:
		fmt.Printf("PMM Server at %s has pmm-agent %s on Node %s, not on %s, registering the Node again.\n",
			cfg.Server.Address, cfg.ID, serverNode, cfg.Setup.NodeName)
		return registrationMissing
	default:
		return registrationConfirmed
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

	return check(cfg, l)
}

// registeredConfig returns the configuration file the Agent runs with, or nil when there is none.
// `pmm-admin config` runs `pmm-agent setup` without --config-file, so the file, whose path is only known
// from the running pmm-agent, has not been loaded yet in that case.
func registeredConfig(configFilepath string, cfg *config.Config) *config.Config {
	fileCfg, err := config.LoadFromFile(configFilepath, &cfg.Encryption)
	var e config.ConfigFileDoesNotExistError
	switch {
	case err == nil:
		return fileCfg
	case errors.As(err, &e):
		return nil
	default:
		fmt.Printf("Failed to read the configuration file %s, registering the Node: %s.\n", configFilepath, err)
		return nil
	}
}

// keepRegistration keeps the credentials the Agent runs with: those given to `pmm-agent setup` only
// serve to register, which replaces them with a service token. It also reports the settings which
// describe the Node on PMM Server, because they are only applied when the Node is registered.
func keepRegistration(cfg, fileCfg *config.Config) {
	if fileCfg.Server.Password != "" {
		cfg.Server.Username = fileCfg.Server.Username
		cfg.Server.Password = fileCfg.Server.Password
	}

	flags := unappliedSetupFlags(&cfg.Setup)
	if len(flags) > 0 {
		fmt.Printf("Settings %s only take effect when the Node is registered.\n", strings.Join(flags, ", "))
	}
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
	configFilepath, err := configStorage.Reload(l)

	var e config.ConfigFileDoesNotExistError
	if err != nil && !errors.As(err, &e) {
		fmt.Printf("Failed to load configuration: %s.\n", err)
		os.Exit(1)
	}

	cfg := configStorage.Get()
	setLocalTransport(cfg.ListenAddress, cfg.ListenPort, l)

	configFilepath, running := checkStatus(configFilepath, l)

	fileCfg := registeredConfig(configFilepath, cfg)
	if cfg.ID == "" && fileCfg != nil {
		cfg.ID = fileCfg.ID
	}

	if cfg.ID == "" && cfg.Setup.SkipRegistration {
		fmt.Printf("Can't skip registration: pmm-agent ID is empty.\n")
		os.Exit(1)
	}

	err = config.IsWritable(configFilepath)
	if err != nil {
		fmt.Printf("Config file %s is not writable: %v.\n", configFilepath, err)
		os.Exit(1)
	}

	if !cfg.Setup.SkipRegistration {
		switch registrationOf(cfg, fileCfg, checkRegistrationOnServer, l) {
		case registrationMissing:
			register(cfg, l)
		case registrationConfirmed:
			fmt.Printf("Node is already registered with %s, pmm-agent ID is %s. Use --force to register it again.\n",
				cfg.Server.Address, cfg.ID)
			keepRegistration(cfg, fileCfg)
		case registrationUnverified:
			fmt.Printf("Keeping the registration, pmm-agent ID is %s. Use --force to register the Node again.\n", cfg.ID)
			keepRegistration(cfg, fileCfg)
		}
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
