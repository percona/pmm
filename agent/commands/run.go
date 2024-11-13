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
	"github.com/percona/pmm/agent/serviceinfobroker"
	"github.com/percona/pmm/agent/tailog"
	"github.com/percona/pmm/agent/versioner"
	"github.com/percona/pmm/api/inventorypb"
)

// Run implements `pmm-agent run` default command.
func Run() {
	const initServerLogsMaxLength = 32 // store logs before load configuration
	logStore := tailog.NewStore(initServerLogsMaxLength)
	logrus.SetOutput(io.MultiWriter(os.Stderr, logStore))
	l := logrus.WithField("component", "main")
	rootCtx, rootCancel := context.WithCancel(context.Background())

	defer l.Info("Done.")

	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		l.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal))) //nolint:forcetypeassert
		rootCancel()
	}()

	v := versioner.New(&versioner.RealExecFunctions{})
	configStorage, configFilepath := prepareConfig(l)

	for {
		ctx, cancel := context.WithCancel(rootCtx)
		cfg := configStorage.Get()

		prepareLogger(cfg, logStore, l)

		supervisor := supervisor.NewSupervisor(ctx, v, configStorage)
		connectionChecker := connectionchecker.New(configStorage)
		serviceInfoBroker := serviceinfobroker.New(configStorage)
		r := runner.New(cfg.RunnerCapacity, cfg.RunnerMaxConnectionsPerService)
		client := client.New(configStorage, supervisor, r, connectionChecker, v, serviceInfoBroker, prepareConnectionService(ctx, cfg), logStore)
		localServer := agentlocal.NewServer(configStorage, supervisor, client, configFilepath, logStore)

		logrus.Infof("Window check connection time is %.2f hour(s)", cfg.WindowConnectedTime.Hours())

		var wg sync.WaitGroup
		wg.Add(3)
		reloadCh := make(chan bool, 1)
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
			localServer.Run(ctx, reloadCh)
			cancel()
		}()

		processClientUntilCancel(ctx, client, reloadCh)

		cleanupTmp(cfg.Paths.TempDir, l)
		wg.Wait()
		select {
		case <-rootCtx.Done():
			return
		default:
		}
	}
}

func processClientUntilCancel(ctx context.Context, client *client.Client, reloadCh chan bool) {
	for {
		clientCtx, cancelClientCtx := context.WithCancel(ctx)
		err := client.Run(clientCtx)
		if err != nil {
			logrus.Errorf("Client error: %s", err)
		}

		cancelClientCtx()
		<-client.Done()

		select {
		case <-reloadCh:
			return
		case <-ctx.Done():
			return
		default:
		}
	}
}

func prepareConfig(l *logrus.Entry) (*config.Storage, string) {
	configStorage := config.NewStorage(nil)
	configFilepath, err := configStorage.Reload(l)
	if err != nil {
		l.Fatalf("Failed to load configuration: %s.", err)
	}

	return configStorage, configFilepath
}

func prepareLogger(cfg *config.Config, logStore *tailog.Store, l *logrus.Entry) {
	config.ConfigureLogger(cfg)
	logStore.Resize(cfg.LogLinesCount)
	l.Debugf("Loaded configuration: %+v", cfg)
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

func prepareConnectionService(ctx context.Context, cfg *config.Config) *connectionuptime.Service {
	connectionUptimeService := connectionuptime.NewService(cfg.WindowConnectedTime)
	connectionUptimeService.RunCleanupGoroutine(ctx)

	return connectionUptimeService
}
