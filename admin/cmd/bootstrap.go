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

// Package cmd holds common logic used by commands.
package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/alecthomas/kong"
	kongcompletion "github.com/jotaen/kong-completion"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/cli"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/commands/base"
	"github.com/percona/pmm/admin/commands/management"
	"github.com/percona/pmm/admin/pkg/flags"
	"github.com/percona/pmm/admin/pkg/logger"
	"github.com/percona/pmm/utils/nodeinfo"
	"github.com/percona/pmm/version"
)

// Bootstrap is used to initialize the application.
func Bootstrap(opts any) {
	var kongCtx *kong.Context
	var kongParser *kong.Kong
	var parsedOpts any

	switch o := opts.(type) {
	case cli.PMMAdminCommands:
		kongParser = kong.Must(&o, getDefaultKongOptions("pmm-admin")...)
		parsedOpts = &o
	case cli.PMMCommands:
		kongParser = kong.Must(&o, getDefaultKongOptions("pmm")...)
		parsedOpts = &o
	}

	kongcompletion.Register(kongParser)
	kongCtx, err := kongParser.Parse(expandArgs(os.Args[1:]))
	kongParser.FatalIfErrorf(err)

	f, ok := parsedOpts.(cli.GlobalFlagsGetter)
	if !ok {
		logrus.Panic("Cannot assert parsedOpts to GlobalFlagsGetter")
	}

	globalFlags := f.GetGlobalFlags()

	configureLogger(globalFlags)
	finishBootstrap(globalFlags)

	err = kongCtx.Run(globalFlags)
	processFinalError(err, bool(globalFlags.JSON))
}

func configureLogger(opts *flags.GlobalFlags) {
	logrus.SetFormatter(&logger.TextFormatter{}) // with levels and timestamps for debug and trace
	if opts.JSON {
		logrus.SetFormatter(&logrus.JSONFormatter{}) //nolint:exhaustruct // with levels and timestamps always present
	}
	if opts.EnableDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if opts.EnableTrace {
		logrus.SetLevel(logrus.TraceLevel)
		logrus.SetReportCaller(true) // https://github.com/sirupsen/logrus/issues/954
	}
}

func expandArgs(args []string) []string {
	var argsResult []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "@") {
			flagsFile := arg[1:]
			logrus.Debugf("Expanding with flags file [%s]", flagsFile)
			readFile, err := os.Open(flagsFile) //nolint:gosec
			if err != nil {
				logrus.Panicf("Failed to parse flags file [%s]: %s", flagsFile, err)
			}
			fileScanner := bufio.NewScanner(readFile)
			fileScanner.Split(bufio.ScanLines)
			for fileScanner.Scan() {
				next := fileScanner.Text()
				if len(next) != 0 {
					logrus.Debugf("Adding arg: %s", next)
					argsResult = append(argsResult, next)
				}
			}

			err = readFile.Close()
			if err != nil {
				logrus.Panicf("Failed to close flags file [%s]: %s.", flagsFile, err)
			}
		} else {
			argsResult = append(argsResult, arg)
		}
	}

	return argsResult
}

func getDefaultKongOptions(appName string) []kong.Option {
	// Detect defaults
	nodeinfo := nodeinfo.Get()
	nodeTypeDefault := "generic"
	if nodeinfo.Container {
		nodeTypeDefault = "container"
	}

	hostname, _ := os.Hostname()

	var defaultMachineID string
	if nodeinfo.MachineID != "" {
		defaultMachineID = nodeinfo.MachineID
	}

	mysqlQuerySources := []string{
		management.MysqlQuerySourceSlowLog,
		management.MysqlQuerySourcePerfSchema,
		management.MysqlQuerySourceNone,
	}

	mongoDBQuerySources := []string{
		management.MongodbQuerySourceProfiler,
		management.MongodbQuerySourceSlowlog,
		management.MongodbQuerySourceNone,
	}

	return []kong.Option{
		kong.Name(appName),
		kong.Description(fmt.Sprintf("Version %s", version.Version)),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:             true,
			NoExpandSubcommands: true,
		}),
		kong.Vars{
			"defaultListenPort":            fmt.Sprintf("%d", agentlocal.DefaultPMMAgentListenPort),
			"nodeIp":                       nodeinfo.PublicAddress,
			"nodeTypeDefault":              nodeTypeDefault,
			"hostname":                     hostname,
			"serviceTypesEnum":             strings.Join(management.AllServiceTypesKeys, ", "),
			"defaultMachineID":             defaultMachineID,
			"distro":                       nodeinfo.Distro,
			"metricsModesEnum":             strings.Join(management.MetricsModes, ", "),
			"mysqlQuerySourcesEnum":        strings.Join(mysqlQuerySources, ", "),
			"mysqlQuerySourceDefault":      mysqlQuerySources[0],
			"mongoDbQuerySourcesEnum":      strings.Join(mongoDBQuerySources, ", "),
			"mongoDbQuerySourceDefault":    mongoDBQuerySources[0],
			"externalDefaultServiceName":   management.DefaultServiceNameSuffix,
			"externalDefaultGroupExporter": management.DefaultGroupExternalExporter,
		},
	}
}

func finishBootstrap(globalFlags *flags.GlobalFlags) {
	ctx, cancel := context.WithCancel(context.Background())

	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		logrus.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal))) //nolint:forcetypeassert
		cancel()
	}()

	agentlocal.SetTransport(ctx, globalFlags.EnableDebug || globalFlags.EnableTrace, globalFlags.PMMAgentListenPort)

	// pmm-admin status command don't connect to PMM Server.
	if commands.SetupClientsEnabled {
		base.SetupClients(ctx, globalFlags)
	}

	commands.CLICtx = ctx
}

func processFinalError(err error, isJSON bool) {
	if err != nil {
		if isJSON {
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
