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
	"github.com/percona/pmm/agent/defaultsfile"
	"github.com/percona/pmm/agent/runner"
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

	var cfg config.Config
	configFilepath, err := config.Get(&cfg, l)
	if err != nil {
		l.Fatalf("Failed to load configuration: %s.", err)
	}

	cleanupTmp(cfg.Paths.TempDir, l)
	connectionUptimeService := connectionuptime.NewService(cfg.WindowConnectedTime)
	connectionUptimeService.RunCleanupGoroutine(ctx)
	supervisor := supervisor.NewSupervisor(ctx, &cfg.Paths, &cfg.Ports, &cfg.Server, &cfg.LogLinesCount)
	supervisor.Run(ctx)
	connectionChecker := connectionchecker.New(&cfg.Paths)
	defaultsFileParser := defaultsfile.New()
	v := versioner.New(&versioner.RealExecFunctions{})
	r := runner.New(cfg.RunnerCapacity)
	go r.Run(ctx)
	client := client.New(&cfg, supervisor, r, connectionChecker, v, defaultsFileParser, connectionUptimeService, logStore)
	localServer := agentlocal.NewServer(&cfg, supervisor, client, configFilepath, logStore)

	go func() {
		localServer.Run(ctx)
		cancel()
	}()

	for {
		_, err = config.Get(&cfg, l)
		if err != nil {
			l.Fatalf("Failed to load configuration: %s.", err)
		}
		config.ConfigureLogger(&cfg)
		logStore.Resize(cfg.LogLinesCount)
		l.Debugf("Loaded configuration: %+v", cfg)

		logrus.Infof("Window check connection time is %.2f hour(s)", cfg.WindowConnectedTime.Hours())
		connectionUptimeService.SetWindowPeriod(cfg.WindowConnectedTime)

		clientCtx, cancel := context.WithCancel(ctx)

		_ = client.Run(clientCtx)
		cancel()

		<-client.Done()

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
