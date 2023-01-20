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
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm/agent/agentlocal"
	"github.com/percona/pmm/agent/agents/supervisor"
	"github.com/percona/pmm/agent/client"
	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/connectionchecker"
	"github.com/percona/pmm/agent/connectionuptime"
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

	configFilepath, err := config.Reload(l)
	if err != nil {
		l.Fatalf("Failed to load configuration: %s.", err)
	}

	cfg := config.Get()

	cleanupTmp(cfg.Paths.TempDir, l)
	connectionUptimeService := connectionuptime.NewService(cfg.WindowConnectedTime)
	connectionUptimeService.RunCleanupGoroutine(ctx)
	supervisor := supervisor.NewSupervisor(ctx, config.Get)
	connectionChecker := connectionchecker.New(config.Get)
	v := versioner.New(&versioner.RealExecFunctions{})
	r := runner.New(cfg.RunnerCapacity)
	client := client.New(config.Get, supervisor, r, connectionChecker, v, connectionUptimeService, logStore)
	localServer := agentlocal.NewServer(config.Get, supervisor, client, configFilepath, logStore)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		supervisor.Run(ctx)
		cancel()
	}()
	go func() {
		defer wg.Done()
		r.Run(ctx)
		cancel()
	}()
	go func() {
		defer wg.Done()
		localServer.Run(ctx)
		cancel()
	}()

	for {
		_, err = config.Reload(l)
		if err != nil {
			l.Fatalf("Failed to load configuration: %s.", err)
		}

		cfg := config.Get()

		config.ConfigureLogger(cfg)
		logStore.Resize(cfg.LogLinesCount)
		l.Debugf("Loaded configuration: %+v", cfg)

		logrus.Infof("Window check connection time is %.2f hour(s)", cfg.WindowConnectedTime.Hours())
		connectionUptimeService.SetWindowPeriod(cfg.WindowConnectedTime)

		clientCtx, cancelClientCtx := context.WithCancel(ctx)

		_ = client.Run(clientCtx)
		cancelClientCtx()

		<-client.Done()

		if ctx.Err() != nil {
			break
		}
	}
	wg.Wait()
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
