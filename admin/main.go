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
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/cli"
	"github.com/percona/pmm/admin/cli/flags"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/commands/base"
	"github.com/percona/pmm/admin/commands/management"
	"github.com/percona/pmm/admin/logger"
	"github.com/percona/pmm/utils/nodeinfo"
	"github.com/percona/pmm/version"
)

func main() {
	// Detect defaults
	nodeinfo := nodeinfo.Get()
	nodeTypeDefault := "generic"
	if nodeinfo.Container {
		nodeTypeDefault = "container"
	}

	hostname, _ := os.Hostname()

	var defaultMachineID string
	if nodeinfo.MachineID != "" {
		defaultMachineID = "/machine_id/" + nodeinfo.MachineID
	}

	mysqlQuerySources := []string{
		management.MysqlQuerySourceSlowLog,
		management.MysqlQuerySourcePerfSchema,
		management.MysqlQuerySourceNone,
	}

	mongoDBQuerySources := []string{
		management.MongodbQuerySourceProfiler,
		management.MongodbQuerySourceNone,
	}

	// Configure CLI
	var opts cli.Commands
	kongCtx := kong.Parse(&opts,
		kong.Name("pmm-admin"),
		kong.Description(fmt.Sprintf("Version %s", version.Version)),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:             true,
			NoExpandSubcommands: true,
		}),
		kong.Vars{
			"socketPath":                   flags.SocketPath,
			"nodeIp":                       nodeinfo.PublicAddress,
			"nodeTypeDefault":              nodeTypeDefault,
			"hostname":                     hostname,
			"serviceTypesEnum":             strings.Join(management.AllServiceTypesKeys, ","),
			"defaultMachineID":             defaultMachineID,
			"distro":                       nodeinfo.Distro,
			"metricsModesEnum":             strings.Join(management.MetricsModes, ","),
			"mysqlQuerySourcesEnum":        strings.Join(mysqlQuerySources, ","),
			"mysqlQuerySourceDefault":      mysqlQuerySources[0],
			"mongoDbQuerySourcesEnum":      strings.Join(mongoDBQuerySources, ","),
			"mongoDbQuerySourceDefault":    mongoDBQuerySources[0],
			"externalDefaultServiceName":   management.DefaultServiceNameSuffix,
			"externalDefaultGroupExporter": management.DefaultGroupExternalExporter,
		})

	logrus.SetFormatter(&logger.TextFormatter{}) // with levels and timestamps for debug and trace
	if opts.JSON {
		logrus.SetFormatter(&logrus.JSONFormatter{}) // with levels and timestamps always present
	}
	if opts.EnableDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if opts.EnableTrace {
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

	if opts.PMMAgentListenPort == 0 && opts.PMMAgentSocket == "" {
		opts.PMMAgentSocket, opts.PMMAgentListenPort = findSocketOrPort()
	}

	agentlocal.SetTransport(
		ctx,
		opts.EnableDebug || opts.EnableTrace,
		opts.PMMAgentListenPort,
		opts.PMMAgentSocket,
	)

	// pmm-admin status command don't connect to PMM Server.
	if commands.SetupClientsEnabled {
		base.SetupClients(ctx, &opts.GlobalFlags)
	}

	commands.CLICtx = ctx

	err := kongCtx.Run(&opts.GlobalFlags)
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

func findSocketOrPort() (socket string, port uint32) {
	logrus.Debug("Detecting socket or port of local pmm-agent")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// We make the channels buffered with size 1 to avoid goroutine leaks
	socketChan := make(chan string, 1)
	portChan := make(chan uint32, 1)

	// Check socket connection
	go func() {
		defer close(socketChan)

		var socket string

		dialer := net.Dialer{}
		conn, err := dialer.DialContext(ctx, "unix", flags.SocketPath)
		if err == nil {
			err := conn.Close()
			if err != nil {
				logrus.Debugf("Socket close error: %#v", err)
			}

			socket = flags.SocketPath
		}

		socketChan <- socket
	}()

	// Check TCP connection
	go func() {
		defer close(portChan)

		var port uint32

		dialer := net.Dialer{}
		conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(agentlocal.Localhost, strconv.Itoa(int(agentlocal.DefaultPMMAgentListenPort))))
		if err == nil {
			err := conn.Close()
			if err != nil {
				logrus.Debugf("TCP close error: %#v", err)
			}

			port = agentlocal.DefaultPMMAgentListenPort
		}

		portChan <- port
	}()

	if sock := <-socketChan; sock != "" {
		logrus.Debugf("Found socket %s", sock)
		return sock, 0
	}

	if port := <-portChan; port != 0 {
		logrus.Debugf("Found port %d", port)
		return "", port
	}

	logrus.Debug("Could not detect socket or port. Using default.")
	return flags.SocketPath, 0
}
