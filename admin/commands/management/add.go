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

package management

import (
	"net"
	"strconv"

	"gopkg.in/alecthomas/kingpin.v2"
)

// register command
var (
	AddC = kingpin.Command("add", "Add Service to monitoring")

	addServiceNameFlag string
	addHostFlag        string
	addPortFlag        uint16
	addLogLevel        string
)

func addGlobalFlags(cmd *kingpin.CmdClause, supportLogLevelFatal bool) {
	// Add command global flags
	cmd.Flag("service-name", "Service name (overrides positional argument)").PlaceHolder("NAME").StringVar(&addServiceNameFlag)
	cmd.Flag("host", "Service hostname or IP address (overrides positional argument)").StringVar(&addHostFlag)
	cmd.Flag("port", "Service port number (overrides positional argument)").Uint16Var(&addPortFlag)

	if supportLogLevelFatal {
		cmd.Flag("log-level", "Service logging level. One of: [debug, info, warn, error, fatal]").Default("warn").EnumVar(&addLogLevel, "debug", "info", "warn", "error", "fatal")
	} else {
		cmd.Flag("log-level", "Service logging level. One of: [debug, info, warn, error]").Default("warn").EnumVar(&addLogLevel, "debug", "info", "warn", "error")
	}
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
func processGlobalAddFlagsWithSocket(cmd connectionGetter) (serviceName string, socket string, host string, port uint16, err error) {
	serviceName = cmd.GetServiceName()
	if addServiceNameFlag != "" {
		serviceName = addServiceNameFlag
	}

	socket = cmd.GetSocket()
	address := cmd.GetAddress()
	if socket == "" && address == "" {
		address = cmd.GetDefaultAddress()
	}

	var portI int

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

	if addHostFlag != "" {
		host = addHostFlag
	}

	if addPortFlag != 0 {
		portI = int(addPortFlag)
	}

	return serviceName, socket, host, uint16(portI), nil
}
