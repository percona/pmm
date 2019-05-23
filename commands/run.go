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
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/grpclog"

	"github.com/percona/pmm-agent/agentlocal"
	"github.com/percona/pmm-agent/agents/supervisor"
	"github.com/percona/pmm-agent/client"
	"github.com/percona/pmm-agent/config"
	"github.com/percona/pmm-agent/connectionchecker"
	"github.com/percona/pmm-agent/utils/logger"
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
		logrus.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal)))
		appCancel()
	}()

	var grpclogOnce sync.Once
	for appCtx.Err() == nil {
		cfg, configFilePath, err := config.Get(l)
		if err != nil {
			logrus.Fatalf("Failed to load configuration: %s.", err)
		}
		logrus.Debugf("Loaded configuration: %+v", cfg)

		logrus.SetLevel(logrus.InfoLevel)
		logrus.SetReportCaller(false) // https://github.com/sirupsen/logrus/issues/954
		if cfg.Debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		if cfg.Trace {
			logrus.SetLevel(logrus.TraceLevel)
			logrus.SetReportCaller(true) // https://github.com/sirupsen/logrus/issues/954
		}

		// SetLoggerV2 is not threads safe, can be changed only once before any gRPC activity
		grpclogOnce.Do(func() {
			if cfg.Trace {
				grpclog.SetLoggerV2(&logger.GRPC{Entry: logrus.WithField("component", "grpclog")})
			}
		})

		for appCtx.Err() == nil {
			ctx, cancel := context.WithCancel(appCtx)
			supervisor := supervisor.NewSupervisor(ctx, &cfg.Paths, &cfg.Ports)
			connectionChecker := connectionchecker.New()
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
