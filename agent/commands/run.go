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

package commands

import (
	"context"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm/agent/agentlocal"
	"github.com/percona/pmm/agent/agents/supervisor"
	"github.com/percona/pmm/agent/client"
	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/connectionchecker"
	"github.com/percona/pmm/agent/connectionuptime"
	"github.com/percona/pmm/agent/credentialssource"
	"github.com/percona/pmm/agent/tailog"
	"github.com/percona/pmm/agent/versioner"
	"github.com/percona/pmm/api/inventorypb"
)

// Run implements `pmm-agent run` default command.
func Run() {
	l := logrus.WithField("component", "main")
	ctx, cancel := context.WithCancel(context.Background())
	defer l.Info("Done.")

	const initServerLogsMaxLength = 32 // store logs before load configuration
	logStore := tailog.NewStore(initServerLogsMaxLength)
	logrus.SetOutput(io.MultiWriter(os.Stderr, logStore))

	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		l.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal)))
		cancel()
	}()

	var connectionUptimeService *connectionuptime.Service
	for {
		cfg, configFilepath, err := config.Get(l)
		if err != nil {
			l.Fatalf("Failed to load configuration: %s.", err)
		}
		config.ConfigureLogger(cfg)
		logStore.Resize(cfg.LogLinesCount)
		l.Debugf("Loaded configuration: %+v", cfg)

		cleanupTmp(cfg.Paths.TempDir, l)

		logrus.Infof("Window check connection time is %.2f hour(s)", cfg.WindowConnectedTime.Hours())
		if connectionUptimeService == nil {
			connectionUptimeService = connectionuptime.NewService(cfg.WindowConnectedTime)
			connectionUptimeService.RunCleanupGoroutine(ctx)
		} else {
			connectionUptimeService.SetWindowPeriod(cfg.WindowConnectedTime)
		}

		run(ctx, cfg, configFilepath, connectionUptimeService, logStore)

		if ctx.Err() != nil {
			return
		}
	}
}

func cleanupTmp(tmpRoot string, log *logrus.Entry) {
	for k := range inventorypb.AgentType_name {
		agentType := inventorypb.AgentType(k)
		if agentType == inventorypb.AgentType_AGENT_TYPE_INVALID {
			continue
		}

		agentTmp := filepath.Join(tmpRoot, strings.ToLower(agentType.String()))
		err := os.RemoveAll(agentTmp)
		if err != nil {
			log.Warnf("Failed to cleanup directory '%s': %s", agentTmp, err.Error())
		}
	}
}

// run runs all pmm-agent components with given configuration until ctx is cancellled.
// See documentation for NewXXX, Run, and Done
func run(ctx context.Context, cfg *config.Config, configFilepath string, cs *connectionuptime.Service, logStore *tailog.Store) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	// Actions runner is currently created inside client.New.
	// It should be created separately.
	// TODO https://jira.percona.com/browse/PMM-7206

	supervisor := supervisor.NewSupervisor(ctx, &cfg.Paths, &cfg.Ports, &cfg.Server, cfg.LogLinesCount)
	connectionChecker := connectionchecker.New(&cfg.Paths)
	credentialsSourceParser := credentialssource.New()
	v := versioner.New(&versioner.RealExecFunctions{})
	client := client.New(cfg, supervisor, connectionChecker, v, credentialsSourceParser, cs, logStore)
	localServer := agentlocal.NewServer(cfg, supervisor, client, configFilepath, logStore)

	go func() {
		_ = client.Run(ctx)
		cancel()
	}()

	localServer.Run(ctx)
	cancel()

	<-client.Done()
}
