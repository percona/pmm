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
	"os/exec"
	"os/signal"

	"github.com/percona/pmm/version"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-admin/commands"
	"github.com/percona/pmm-admin/commands/inventory"
	"github.com/percona/pmm-admin/commands/management"
	"github.com/percona/pmm-admin/logger"
)

func main() {
	kingpin.CommandLine.Name = "pmm-admin"
	kingpin.CommandLine.Help = fmt.Sprintf("Version %s", version.Version)
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.CommandLine.Version(version.FullInfo())
	kingpin.CommandLine.UsageTemplate(commands.UsageTemplate)

	serverURLF := kingpin.Flag("server-url", "PMM Server URL in `https://username:password@pmm-server-host/` format").String()
	kingpin.Flag("server-insecure-tls", "Skip PMM Server TLS certificate validation").BoolVar(&commands.GlobalFlags.ServerInsecureTLS)
	kingpin.Flag("debug", "Enable debug logging").BoolVar(&commands.GlobalFlags.Debug)
	kingpin.Flag("trace", "Enable trace logging (implies debug)").BoolVar(&commands.GlobalFlags.Trace)
	jsonF := kingpin.Flag("json", "Enable JSON output").Bool()

	cmd := kingpin.Parse()

	logrus.SetFormatter(new(logger.TextFormatter)) // with levels and timestamps for debug and trace
	if *jsonF {
		logrus.SetFormatter(new(logrus.JSONFormatter)) // with levels and timestamps always present
	}
	if commands.GlobalFlags.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if commands.GlobalFlags.Trace {
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

	commands.SetupClients(ctx, *serverURLF)

	var command commands.Command
	switch cmd {
	case management.RegisterC.FullCommand():
		command = management.Register

	case management.AddMySQLC.FullCommand():
		command = management.AddMySQL

	case management.AddMongoDBC.FullCommand():
		command = management.AddMongoDB

	case management.AddPostgreSQLC.FullCommand():
		command = management.AddPostgreSQL

	case management.AddProxySQLC.FullCommand():
		command = management.AddProxySQL

	case management.RemoveC.FullCommand():
		command = management.Remove

	case inventory.ListNodesC.FullCommand():
		command = inventory.ListNodes

	case inventory.AddNodeGenericC.FullCommand():
		command = inventory.AddNodeGeneric

	case inventory.AddNodeContainerC.FullCommand():
		command = inventory.AddNodeContainer

	case inventory.AddNodeRemoteC.FullCommand():
		command = inventory.AddNodeRemote

	case inventory.RemoveNodeC.FullCommand():
		command = inventory.RemoveNode

	case inventory.ListServicesC.FullCommand():
		command = inventory.ListServices

	case inventory.AddServiceMySQLC.FullCommand():
		command = inventory.AddServiceMySQL

	case inventory.AddServiceMongoDBC.FullCommand():
		command = inventory.AddServiceMongoDB

	case inventory.AddServicePostgreSQLC.FullCommand():
		command = inventory.AddServicePostgreSQL

	case inventory.AddServiceProxySQLC.FullCommand():
		command = inventory.AddServiceProxySQL

	case inventory.RemoveServiceC.FullCommand():
		command = inventory.RemoveService

	case inventory.ListAgentsC.FullCommand():
		command = inventory.ListAgents

	case inventory.AddAgentPMMAgentC.FullCommand():
		command = inventory.AddAgentPMMAgent

	case inventory.AddAgentNodeExporterC.FullCommand():
		command = inventory.AddAgentNodeExporter

	case inventory.AddAgentMysqldExporterC.FullCommand():
		command = inventory.AddAgentMysqldExporter

	case inventory.AddAgentMongodbExporterC.FullCommand():
		command = inventory.AddAgentMongodbExporter

	case inventory.AddAgentPostgresExporterC.FullCommand():
		command = inventory.AddAgentPostgresExporter

	case inventory.AddAgentProxysqlExporterC.FullCommand():
		command = inventory.AddAgentProxysqlExporter

	case inventory.AddAgentQANMySQLPerfSchemaAgentC.FullCommand():
		command = inventory.AddAgentQANMySQLPerfSchemaAgent

	case inventory.AddAgentQANMySQLSlowlogAgentC.FullCommand():
		command = inventory.AddAgentQANMySQLSlowlogAgent

	case inventory.AddAgentQANMongoDBProfilerAgentC.FullCommand():
		command = inventory.AddAgentQANMongoDBProfilerAgent

	case inventory.AddAgentQANPostgreSQLPgStatementsAgentC.FullCommand():
		command = inventory.AddAgentQANPostgreSQLPgStatementsAgent

	case inventory.RemoveAgentC.FullCommand():
		command = inventory.RemoveAgent

	case commands.ListC.FullCommand():
		command = commands.List

	case commands.StatusC.FullCommand():
		logrus.Warn("`status` command is deprecated. Use `summary` instead.")
		fallthrough
	case commands.SummaryC.FullCommand():
		command = commands.Summary

	case commands.ConfigC.FullCommand():
		command = commands.Config

	default:
		logrus.Panicf("Unhandled command %q. Please report this bug.", cmd)
	}

	res, err := command.Run()
	logrus.Debugf("Result: %#v", res)
	logrus.Debugf("Error: %#v", err)

	switch err := err.(type) {
	case nil:
		if *jsonF {
			b, jErr := json.Marshal(res)
			if jErr != nil {
				logrus.Infof("Result: %#v.", res)
				logrus.Panicf("Failed to marshal result to JSON.\n%s.\nPlease report this bug.", jErr)
			}
			fmt.Printf("%s\n", b)
		} else {
			fmt.Println(res.String())
		}

		os.Exit(0)

	case commands.ErrorResponse:
		e := commands.GetError(err)

		if *jsonF {
			b, jErr := json.Marshal(e)
			if jErr != nil {
				logrus.Infof("Error response: %#v.", e)
				logrus.Panicf("Failed to marshal error response to JSON.\n%s.\nPlease report this bug.", jErr)
			}
			fmt.Printf("%s\n", b)
		} else {
			msg := e.Error
			if e.Code == 401 || e.Code == 403 {
				msg += ". Please check username and password."
			}
			fmt.Println(msg)
		}

		os.Exit(1)

	case *exec.ExitError: // from config command that execs `pmm-agent setup`
		if *jsonF {
			b, jErr := json.Marshal(res)
			if jErr != nil {
				logrus.Infof("Result: %#v.", res)
				logrus.Panicf("Failed to marshal result to JSON.\n%s.\nPlease report this bug.", jErr)
			}
			fmt.Printf("%s\n", b)
		} else {
			fmt.Println(res.String())
		}

		os.Exit(err.ExitCode())

	default:
		if *jsonF {
			b, jErr := json.Marshal(err.Error())
			if jErr != nil {
				logrus.Infof("Error: %#v.", err)
				logrus.Panicf("Failed to marshal error to JSON.\n%s.\nPlease report this bug.", jErr)
			}
			fmt.Printf("%s\n", b)
		} else {
			fmt.Println(err)
		}

		os.Exit(1)
	}
}
