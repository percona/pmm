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
		cfg, configFilePath, err := config.Get(l)
		if err != nil {
			l.Fatalf("Failed to load configuration: %s.", err)
		}
		config.ConfigureLogger(cfg)
		l.Debugf("Loaded configuration: %+v", cfg)

		for appCtx.Err() == nil {
			ctx, cancel := context.WithCancel(appCtx)
			supervisor := supervisor.NewSupervisor(ctx, &cfg.Paths, &cfg.Ports)
			connectionChecker := connectionchecker.New(ctx)
			client := client.New(cfg, supervisor, connectionChecker)
			localServer := agentlocal.NewServer(cfg, supervisor, client, configFilePath)

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
