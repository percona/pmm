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

	allCommands := map[string]commands.Command{
		management.RegisterC.FullCommand(): management.Register,

		management.AddMySQLC.FullCommand():      management.AddMySQL,
		management.AddMongoDBC.FullCommand():    management.AddMongoDB,
		management.AddPostgreSQLC.FullCommand(): management.AddPostgreSQL,
		management.AddProxySQLC.FullCommand():   management.AddProxySQL,

		management.RemoveC.FullCommand(): management.Remove,

		inventory.ListNodesC.FullCommand(): inventory.ListNodes,

		inventory.AddNodeGenericC.FullCommand():   inventory.AddNodeGeneric,
		inventory.AddNodeContainerC.FullCommand(): inventory.AddNodeContainer,
		inventory.AddNodeRemoteC.FullCommand():    inventory.AddNodeRemote,
		inventory.AddNodeRemoteRDSC.FullCommand(): inventory.AddNodeRemoteRDS,

		inventory.RemoveNodeC.FullCommand(): inventory.RemoveNode,

		inventory.ListServicesC.FullCommand(): inventory.ListServices,

		inventory.AddServiceMySQLC.FullCommand():      inventory.AddServiceMySQL,
		inventory.AddServiceMongoDBC.FullCommand():    inventory.AddServiceMongoDB,
		inventory.AddServicePostgreSQLC.FullCommand(): inventory.AddServicePostgreSQL,
		inventory.AddServiceProxySQLC.FullCommand():   inventory.AddServiceProxySQL,
		inventory.AddExternalServiceC.FullCommand():   inventory.AddExternalService,

		inventory.RemoveServiceC.FullCommand(): inventory.RemoveService,

		inventory.ListAgentsC.FullCommand(): inventory.ListAgents,

		inventory.AddAgentPMMAgentC.FullCommand():                       inventory.AddAgentPMMAgent,
		inventory.AddAgentNodeExporterC.FullCommand():                   inventory.AddAgentNodeExporter,
		inventory.AddAgentMysqldExporterC.FullCommand():                 inventory.AddAgentMysqldExporter,
		inventory.AddAgentMongodbExporterC.FullCommand():                inventory.AddAgentMongodbExporter,
		inventory.AddAgentPostgresExporterC.FullCommand():               inventory.AddAgentPostgresExporter,
		inventory.AddAgentProxysqlExporterC.FullCommand():               inventory.AddAgentProxysqlExporter,
		inventory.AddAgentQANMySQLPerfSchemaAgentC.FullCommand():        inventory.AddAgentQANMySQLPerfSchemaAgent,
		inventory.AddAgentQANMySQLSlowlogAgentC.FullCommand():           inventory.AddAgentQANMySQLSlowlogAgent,
		inventory.AddAgentQANMongoDBProfilerAgentC.FullCommand():        inventory.AddAgentQANMongoDBProfilerAgent,
		inventory.AddAgentQANPostgreSQLPgStatementsAgentC.FullCommand(): inventory.AddAgentQANPostgreSQLPgStatementsAgent,
		inventory.AddAgentRDSExporterC.FullCommand():                    inventory.AddAgentRDSExporter,
		inventory.AddAgentExternalExporterC.FullCommand():               inventory.AddAgentExternalExporter,

		inventory.RemoveAgentC.FullCommand(): inventory.RemoveAgent,

		commands.ListC.FullCommand():       commands.List,
		commands.AnnotationC.FullCommand(): commands.Annotation,
		commands.StatusC.FullCommand():     commands.Status,
		commands.SummaryC.FullCommand():    commands.Summary,
		commands.ConfigC.FullCommand():     commands.Config,
	}
	command := allCommands[cmd]

	if command == nil {
		logrus.Panicf("Unhandled command %q. Please report this bug.", cmd)
	}

	var res commands.Result
	var err error
	if cc, ok := command.(commands.CommandWithContext); ok {
		res, err = cc.RunWithContext(ctx)
	} else {
		res, err = command.Run()
	}
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

		if err.Stderr != nil {
			logrus.Debugf("%s, stderr:\n%s", err.String(), err.Stderr)
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
