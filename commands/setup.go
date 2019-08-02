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

package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/percona/pmm/api/agentlocalpb/json/client/agent_local"
	"github.com/percona/pmm/api/managementpb/json/client/node"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"

	"github.com/percona/pmm-agent/config"
)

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
	cfg, configFilePath, err := config.Get(l)
	if _, ok := err.(config.ErrConfigFileDoesNotExist); ok {
		err = nil
	}
	if err != nil {
		fmt.Printf("Failed to load configuration: %s.\n", err)
		os.Exit(1)
	}

	setLocalTransport(cfg.ListenPort, l)

	configFilePath, running := checkStatus(configFilePath, l)

	if cfg.ID == "" && cfg.Setup.SkipRegistration {
		fmt.Printf("Can't skip registration: pmm-agent ID is empty.\n")
		os.Exit(1)
	}
	if !cfg.Setup.SkipRegistration {
		register(cfg, l)
	}

	if err = config.SaveToFile(configFilePath, cfg, "Updated by `pmm-agent setup`."); err != nil {
		fmt.Printf("Failed to write configuration file %s: %s.\n", configFilePath, err)
		os.Exit(1)
	}
	fmt.Printf("Configuration file %s updated.\n", configFilePath)

	if !running {
		fmt.Printf("Please start pmm-agent: `pmm-agent --config-file=%s`.\n", configFilePath)
		return
	}

	reload(l)

	checkStatus(configFilePath, l)
}

func checkStatus(configFilePath string, l *logrus.Entry) (string, bool) {
	fmt.Printf("Checking local pmm-agent status...\n")
	status, err := localStatus()
	l.Debugf("Status error: %#v", err)
	switch err := err.(type) {
	case nil:
		if status.ConfigFilePath == "" {
			fmt.Printf("pmm-agent is running but does not read configuration from the file. " +
				"Please restart it with --config-file flag.\n")
			os.Exit(1)
		}
		if configFilePath != "" && status.ConfigFilePath != configFilePath {
			fmt.Printf("pmm-agent is running and reads configuration from %s. "+
				"Please re-run `pmm-agent setup` without --config-file flag.\n", status.ConfigFilePath)
			os.Exit(1)
		}
		fmt.Printf("pmm-agent is running.\n")
		return status.ConfigFilePath, true

	case *agent_local.StatusDefault:
		msg := fmt.Sprintf("HTTP code %d", err.Code())
		if err.Payload != nil {
			msg = fmt.Sprintf("%s (gRPC code %d, HTTP code %d)", err.Payload.Error, err.Payload.Code, err.Code())
		}
		fmt.Printf("pmm-agent is running, but status check failed: %s.\n", msg)
		os.Exit(1)
		panic("not reached")

	default:
		if configFilePath == "" {
			fmt.Printf("pmm-agent is not running. Please re-run `pmm-agent setup` with --config-file flag.\n")
			os.Exit(1)
		}
		fmt.Printf("pmm-agent is not running.\n")
		return configFilePath, false
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
	agentID, err := serverRegister(&cfg.Setup)
	l.Debugf("Register error: %#v", err)
	if err != nil {
		msg := err.Error()
		if e, _ := err.(*node.RegisterDefault); e != nil {
			msg = e.Payload.Error + ""
			if e.Code() == 401 || e.Code() == 403 {
				msg += ".\nPlease check username and password"
			}
		}
		if _, ok := err.(errFromNginx); ok {
			msg += ".\nPlease check pmm-managed logs"
		}

		fmt.Printf("Failed to register pmm-agent on PMM Server: %s.\n", msg)
		os.Exit(1)
	}

	fmt.Printf("Registered.\n")
	cfg.ID = agentID
}

func reload(l *logrus.Entry) {
	fmt.Printf("Reloading pmm-agent configuration...\n")

	// sync error handling with Reload API method
	err := localReload()
	l.Debugf("Reload error: %#v", err)
	if err, _ := err.(*agent_local.ReloadDefault); err != nil && err.Code() == int(codes.FailedPrecondition) {
		fmt.Printf("Failed to reload configuration: %s.\n", err.Payload.Error)
		os.Exit(1)
	}

	// wait up to 5 seconds for pmm-agent to reload itself
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)
		_, err = localStatus()
		l.Debugf("Status error: %#v", err)
		if err == nil {
			fmt.Printf("Configuration reloaded.\n")
			return
		}
	}
}
