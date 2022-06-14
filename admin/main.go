// pmm-admin
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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"

	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/cli"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/logger"
	"github.com/percona/pmm/version"
)

func main() {
	var opts cli.CLIFlags
	kongCtx := kong.Parse(&opts,
		kong.Name("pmm-admin"),
		kong.Description(fmt.Sprintf("Version %s", version.Version)),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Bind(&cli.CLI),
		kong.Vars{
			"defaultListenPort": fmt.Sprintf("%d", agentlocal.DefaultPMMAgentListenPort),
		})

	if opts.Version {
		if opts.JSON {
			fmt.Println(version.FullInfoJSON())
		} else {
			fmt.Println(version.FullInfo())
		}
		os.Exit(0)

		return
	}

	logrus.SetFormatter(&logger.TextFormatter{}) // with levels and timestamps for debug and trace
	if opts.JSON {
		logrus.SetFormatter(&logrus.JSONFormatter{}) // with levels and timestamps always present
	}
	if opts.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if opts.Trace {
		logrus.SetLevel(logrus.TraceLevel)
		logrus.SetReportCaller(true) // https://github.com/sirupsen/logrus/issues/954
	}

	ctx, cancel := context.WithCancel(context.Background())

	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		logrus.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal)))
		cancel()
	}()

	agentlocal.SetTransport(ctx, opts.Debug || opts.Trace, opts.PMMAgentListenPort)

	// pmm-admin status command don't connect to PMM Server.
	if cli.CLI.SetupClients {
		commands.SetupClients(ctx, opts.ServerURL)
	}

	cli.CLI.Ctx = ctx

	err := kongCtx.Run()
	if err != nil {
		if opts.JSON {
			b, jErr := json.Marshal(err.Error())
			if jErr != nil {
				logrus.Infof("Error: %#v.", err)
				logrus.Panicf("Failed to marshal error to JSON.\n%s.\nPlease report this bug.", jErr)
			}
			fmt.Printf("%s\n", b) //nolint:forbidigo
		} else {
			fmt.Println(err) //nolint:forbidigo
		}

		os.Exit(1)
	}
}
