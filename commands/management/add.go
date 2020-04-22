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

package management

import (
	"net"
	"strconv"

	"gopkg.in/alecthomas/kingpin.v2"
)

// register command
var (
	AddC = kingpin.Command("add", "Add Service to monitoring")

	// Add command global flags
	addServiceNameFlag = AddC.Flag("service-name", "Service name (overrides positional argument)").PlaceHolder("NAME").String()
	addHostFlag        = AddC.Flag("host", "Service hostname or IP address (overrides positional argument)").String()
	addPortFlag        = AddC.Flag("port", "Service port number (overrides positional argument)").Uint16()
)

type getter interface {
	GetServiceName() string
	GetAddress() string
}

// Types implementing the getter interface:
// - addMongoDBCommand
// - addMySQLCommand
// - addPostgreSQLCommand
// - addProxySQLCommand
// Returns service name, host, port, error.
func processGlobalAddFlags(cmd getter) (string, string, uint16, error) {
	serviceName := cmd.GetServiceName()
	if *addServiceNameFlag != "" {
		serviceName = *addServiceNameFlag
	}

	host, portS, err := net.SplitHostPort(cmd.GetAddress())
	if err != nil {
		return "", "", 0, err
	}

	port, err := strconv.Atoi(portS)
	if err != nil {
		return "", "", 0, err
	}

	if *addHostFlag != "" {
		host = *addHostFlag
	}

	if *addPortFlag != 0 {
		port = int(*addPortFlag)
	}

	return serviceName, host, uint16(port), nil
}
