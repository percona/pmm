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

package management

import (
	"net"
	"strconv"
)

// AddCommand is used by Kong for CLI flags and commands.
type AddCommand struct {
	External           AddExternalCommand           `cmd:"" help:"Add External source of data (like a custom exporter running on a port) to monitoring"`
	ExternalServerless AddExternalServerlessCommand `cmd:"" help:"Add External Service on Remote node to monitoring"`
	HAProxy            AddHAProxyCommand            `cmd:"" name:"haproxy" help:"Add HAProxy to monitoring"`
	MongoDB            AddMongoDBCommand            `cmd:"" name:"mongodb" help:"Add MongoDB to monitoring"`
	MySQL              AddMySQLCommand              `cmd:"" name:"mysql" help:"Add MySQL to monitoring"`
	PostgreSQL         AddPostgreSQLCommand         `cmd:"" name:"postgresql" help:"Add PostgreSQL to monitoring"`
	ProxySQL           AddProxySQLCommand           `cmd:"" name:"proxysql" help:"Add ProxySQL to monitoring"`
}

// AddCommonFlags is used by Kong for CLI flags and commands.
type AddCommonFlags struct {
	AddServiceNameFlag string `name:"service-name" placeholder:"NAME" help:"Service name (overrides positional argument)"`
	AddHostFlag        string `name:"host" placeholder:"HOST" help:"Service hostname or IP address (overrides positional argument)"`
	AddPortFlag        uint16 `name:"port" placeholder:"PORT" help:"Service port number (overrides positional argument)"`
}

// AddLogLevelFatalFlags contains log level flag with "fatal" option.
type AddLogLevelFatalFlags struct {
	AddLogLevel string `name:"log-level" enum:"debug,info,warn,error,fatal" default:"warn" help:"Service logging level. One of: [debug, info, warn, error, fatal]"`
}

// AddLogLevelNoFatalFlags contains log level flag without "fatal" option.
type AddLogLevelNoFatalFlags struct {
	AddLogLevel string `name:"log-level" enum:"debug,info,warn,error" default:"warn" help:"Service logging level. One of: [debug, info, warn, error]"`
}

type connectionGetter interface {
	GetServiceName() string
	GetAddress() string
	GetDefaultAddress() string
	GetSocket() string
}

// Types implementing the getter interface:
// - addMySQLCommand
// - addProxySQLCommand
// - addPostgreSQLCommand
// - addMongoDBCommand
// Returns service name, socket, host, port, error.
func processGlobalAddFlagsWithSocket(cmd connectionGetter, opts AddCommonFlags) (string, string, string, uint16, error) {
	serviceName := cmd.GetServiceName()
	if opts.AddServiceNameFlag != "" {
		serviceName = opts.AddServiceNameFlag
	}

	socket := cmd.GetSocket()
	address := cmd.GetAddress()
	if socket == "" && address == "" {
		address = cmd.GetDefaultAddress()
	}

	var portI int
	var host string
	var err error

	if address != "" {
		var portS string
		host, portS, err = net.SplitHostPort(address)
		if err != nil {
			return "", "", "", 0, err
		}

		portI, err = strconv.Atoi(portS)
		if err != nil {
			return "", "", "", 0, err
		}
	}

	if opts.AddHostFlag != "" {
		host = opts.AddHostFlag
	}

	if opts.AddPortFlag != 0 {
		portI = int(opts.AddPortFlag)
	}

	return serviceName, socket, host, uint16(portI), nil
}
