// pmm-admin
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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	inventorypb "github.com/percona/pmm/api/inventorypb/json/client"
	managementpb "github.com/percona/pmm/api/managementpb/json/client"
	serverpb "github.com/percona/pmm/api/serverpb/json/client"
	"github.com/percona/pmm/version"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-admin/agentlocal"
	"github.com/percona/pmm-admin/commands"
	"github.com/percona/pmm-admin/commands/inventory"
	"github.com/percona/pmm-admin/commands/management"
	"github.com/percona/pmm-admin/logger"
)

type errFromNginx string

func (e errFromNginx) Error() string {
	return "response from nginx: " + string(e)
}

func (e errFromNginx) GoString() string {
	return fmt.Sprintf("errFromNginx(%q)", string(e))
}

func main() {
	kingpin.CommandLine.Name = "pmm-admin"
	kingpin.CommandLine.Help = fmt.Sprintf("Version %s", version.Version)
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.CommandLine.Version(version.FullInfo())

	serverURLF := kingpin.Flag("server-url", "PMM Server URL").String()
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

	agentlocal.SetTransport(ctx, commands.GlobalFlags.Debug || commands.GlobalFlags.Trace)

	if *serverURLF == "" {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			if err == agentlocal.ErrNotSetUp {
				logrus.Fatalf("Failed to get PMM Server parameters from local pmm-agent: %s.\n"+
					"Please run `pmm-admin config` with --server-url flag.", err)
			}
			logrus.Fatalf("Failed to get PMM Server parameters from local pmm-agent: %s.\n"+
				"Please use --server-url flag to specify PMM Server URL.", err)
		}
		commands.GlobalFlags.ServerURL = status.ServerURL
		commands.GlobalFlags.ServerInsecureTLS = status.ServerInsecureTLS
	} else {
		var err error
		commands.GlobalFlags.ServerURL, err = url.Parse(*serverURLF)
		if err != nil {
			logrus.Fatalf("Failed to parse PMM Server URL %q: %s.", *serverURLF, err)
		}
		if commands.GlobalFlags.ServerURL.Path == "" {
			commands.GlobalFlags.ServerURL.Path = "/"
		}
		if commands.GlobalFlags.ServerURL.Host == "" {
			logrus.Fatalf("Invalid PMM Server URL %q: host is missing.", *serverURLF)
		}
		if commands.GlobalFlags.ServerURL.Scheme == "" {
			logrus.Fatalf("Invalid PMM Server URL %q: scheme is missing.", *serverURLF)
		}
	}

	// use JSON APIs over HTTP/1.1
	transport := httptransport.New(commands.GlobalFlags.ServerURL.Host, commands.GlobalFlags.ServerURL.Path, []string{commands.GlobalFlags.ServerURL.Scheme})
	// FIXME https://jira.percona.com/browse/PMM-3886
	if commands.GlobalFlags.ServerURL.User != nil {
		logrus.Panic("PMM Server authentication is not implemented yet.")
	}
	transport.SetLogger(logrus.WithField("component", "server-transport"))
	transport.SetDebug(commands.GlobalFlags.Debug || commands.GlobalFlags.Trace)
	transport.Context = ctx

	// set error handlers for nginx responses if pmm-managed is down
	errorConsumer := runtime.ConsumerFunc(func(reader io.Reader, data interface{}) error {
		b, _ := ioutil.ReadAll(reader)
		return errFromNginx(string(b))
	})
	transport.Consumers = map[string]runtime.Consumer{
		runtime.JSONMime:    runtime.JSONConsumer(),
		runtime.HTMLMime:    errorConsumer,
		runtime.TextMime:    errorConsumer,
		runtime.DefaultMime: errorConsumer,
	}

	// disable HTTP/2, set TLS config
	httpTransport := transport.Transport.(*http.Transport)
	httpTransport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	if commands.GlobalFlags.ServerURL.Scheme == "https" {
		if httpTransport.TLSClientConfig == nil {
			httpTransport.TLSClientConfig = new(tls.Config)
		}
		if commands.GlobalFlags.ServerInsecureTLS {
			httpTransport.TLSClientConfig.InsecureSkipVerify = true
		} else {
			httpTransport.TLSClientConfig.ServerName = commands.GlobalFlags.ServerURL.Hostname()
		}
	}

	inventorypb.Default.SetTransport(transport)
	managementpb.Default.SetTransport(transport)
	serverpb.Default.SetTransport(transport)

	var command commands.Command
	switch cmd {
	case management.RegisterC.FullCommand():
		command = management.Register

	case management.AddMongoDBC.FullCommand():
		command = management.AddMongoDB

	case management.AddMySQLC.FullCommand():
		command = management.AddMySQL

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

	case inventory.AddAgentMongodbExporterC.FullCommand():
		command = inventory.AddAgentMongodbExporter

	case inventory.AddAgentMysqldExporterC.FullCommand():
		command = inventory.AddAgentMysqldExporter

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
		command = commands.Status

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
			fmt.Println(e.Error)
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

// check interfaces
var (
	_ error          = errFromNginx("")
	_ fmt.GoStringer = errFromNginx("")
)
