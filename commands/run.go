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

package commands

import (
	"context"
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm-agent/agentlocal"
	"github.com/percona/pmm-agent/agents/supervisor"
	"github.com/percona/pmm-agent/client"
	"github.com/percona/pmm-agent/config"
	"github.com/percona/pmm-agent/connectionchecker"
)

// Run implements `pmm-agent run` default command.
func Run() {
	l := logrus.WithField("component", "main")
	appCtx, appCancel := context.WithCancel(context.Background())
	defer l.Info("Done.")

	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		l.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal)))
		appCancel()
	}()

	for appCtx.Err() == nil {
		cfg, configFilepath, err := config.Get(l)
		if err != nil {
			l.Fatalf("Failed to load configuration: %s.", err)
		}
		config.ConfigureLogger(cfg)
		l.Debugf("Loaded configuration: %+v", cfg)

		for appCtx.Err() == nil {
			ctx, cancel := context.WithCancel(appCtx)
			supervisor := supervisor.NewSupervisor(ctx, &cfg.Paths, &cfg.Ports, &cfg.Server)
			connectionChecker := connectionchecker.New(ctx, &cfg.Paths)
			client := client.New(cfg, supervisor, connectionChecker)
			localServer := agentlocal.NewServer(cfg, supervisor, client, configFilepath)

			go func() {
				_ = client.Run(ctx)
				cancel()
			}()

			err = localServer.Run(ctx)
			cancel()

			<-client.Done()
			if err == agentlocal.ErrReload {
				break
			}
		}
	}
}
