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

package main

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"github.com/percona/pmm/version"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/grpclog"

	"github.com/percona/pmm-agent/agentlocal"
	"github.com/percona/pmm-agent/agents/supervisor"
	"github.com/percona/pmm-agent/client"
	"github.com/percona/pmm-agent/config"
	"github.com/percona/pmm-agent/utils/logger"
)

func main() {
	// empty version breaks much of pmm-managed logic
	if version.Version == "" {
		panic("pmm-agent version is not set during build.")
	}

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
		cfg, err := config.Parse(logrus.WithField("component", "config"))
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Debugf("Loaded configuration: %+v", cfg)

		logrus.SetLevel(logrus.InfoLevel)
		logrus.SetReportCaller(false)
		if cfg.Debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		if cfg.Trace {
			logrus.SetLevel(logrus.TraceLevel)
			logrus.SetReportCaller(true)
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
			localServer := agentlocal.NewServer(cfg, supervisor)
			client := client.New(cfg, supervisor)

			go func() {
				_ = client.Run(ctx, localServer)
				cancel()
			}()

			err = localServer.Run(ctx, client)
			cancel()

			<-client.Done()
			if err == agentlocal.ErrReload {
				break
			}
		}
	}
}
