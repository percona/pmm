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
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"

	"github.com/percona/pmm/agent/config"
	agent_local "github.com/percona/pmm/api/agentlocal/v1/json/client/agent_local_service"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

// registrationCheck reports whether PMM Server no longer knows the Agent described by cfg,
// so that the Node has to be registered again.
type registrationCheck func(cfg *config.Config, l *logrus.Entry) bool

// mustRegisterOnServer reports whether PMM Server no longer knows this Agent. The server may have
// been reinstalled, or restored from a backup taken before the Agent was registered, leaving the
// Agent with an ID nothing recognizes. An unreachable server is not an answer: an Agent has to be
// able to start while PMM Server has no leader yet, so its registration is kept in that case.
func mustRegisterOnServer(cfg *config.Config, l *logrus.Entry) bool {
	u := cfg.Server.URL()
	if u == nil {
		// register reports the missing server address with an actionable message
		return true
	}
	setServerTransport(u, cfg.Server.InsecureTLS, l)

	known, err := serverKnowsAgent(cfg.ID)
	if err != nil {
		l.Warnf("Failed to check the registration of pmm-agent %s with %s, keeping it: %s", cfg.ID, cfg.Server.Address, err)
		return false
	}
	if !known {
		fmt.Printf("PMM Server at %s does not know pmm-agent %s, registering the Node again.\n", cfg.Server.Address, cfg.ID)
	}

	return !known
}

// skipRegistration reports whether `pmm-agent setup` can leave the Node registration as it is.
// An Agent which already holds an ID is registered with PMM Server, and registering it again makes
// the server drop the Node together with every Service on it. That is only done on demand, when the
// Agent is being pointed at a different PMM Server, or when the server no longer knows the Agent.
func skipRegistration(cfg *config.Config, configFilepath string, mustRegister registrationCheck, l *logrus.Entry) bool {
	if cfg.Setup.SkipRegistration {
		return true
	}
	if cfg.ID == "" || cfg.Setup.Force {
		return false
	}

	fileCfg, err := config.LoadFromFile(configFilepath, &cfg.Encryption)
	if err != nil {
		l.Warnf("Failed to read the configuration file %s, registering the Node: %s", configFilepath, err)
		return false
	}
	if fileCfg.Server.Address != cfg.Server.Address {
		return false
	}

	return !mustRegister(cfg, l)
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

	if cfg.ID == "" && cfg.Setup.SkipRegistration {
		fmt.Printf("Can't skip registration: pmm-agent ID is empty.\n")
		os.Exit(1)
	}

	err = config.IsWritable(configFilepath)
	if err != nil {
		fmt.Printf("Config file %s is not writable: %v.\n", configFilepath, err)
		os.Exit(1)
	}

	if skipRegistration(cfg, configFilepath, mustRegisterOnServer, l) {
		fmt.Printf("Node is already registered with %s, pmm-agent ID is %s. Use --force to register it again.\n",
			cfg.Server.Address, cfg.ID)
	} else {
		register(cfg, l)
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
