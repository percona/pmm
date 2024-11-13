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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManagementGlobalFlags(t *testing.T) {
	tests := []struct {
		testName string

		nameArg    string
		addressArg string

		serviceNameFlag string
		hostFlag        string
		portFlag        uint16
		socketFlag      string

		wantServiceName string
		wantHost        string
		wantPort        uint16
		wantSocket      string
	}{
		{
			testName: "Only positional arguments",

			nameArg:    "service-name",
			addressArg: "localhost:27017",

			wantServiceName: "service-name",
			wantHost:        "localhost",
			wantPort:        27017,
		},
		{
			testName: "Override only host",

			nameArg:    "service-name",
			addressArg: "localhost:27017",

			hostFlag: "visitant-host",

			wantServiceName: "service-name",
			wantHost:        "visitant-host",
			wantPort:        27017,
		},
		{
			testName: "Override only port",

			nameArg:    "service-name",
			addressArg: "localhost:27017",

			portFlag: 27018,

			wantServiceName: "service-name",
			wantHost:        "localhost",
			wantPort:        27018,
		},
		{
			testName: "Override only service name",

			nameArg:    "service-name",
			addressArg: "localhost:27017",

			serviceNameFlag: "no-service",

			wantServiceName: "no-service",
			wantHost:        "localhost",
			wantPort:        27017,
		},
		{
			testName: "Override everything",

			nameArg:    "service-name",
			addressArg: "localhost:27017",

			serviceNameFlag: "out-of-service",
			hostFlag:        "new-address",
			portFlag:        27019,

			wantServiceName: "out-of-service",
			wantHost:        "new-address",
			wantPort:        27019,
		},
		{
			testName: "Socket",

			serviceNameFlag: "service-with-socket",
			socketFlag:      "/tmp/mongodb-27017.sock",

			wantServiceName: "service-with-socket",
			wantSocket:      "/tmp/mongodb-27017.sock",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.testName, func(t *testing.T) {
			cmd := &AddMongoDBCommand{
				ServiceName: test.nameArg,
				Address:     test.addressArg,
				Socket:      test.socketFlag,
				AddCommonFlags: AddCommonFlags{
					AddServiceNameFlag: test.serviceNameFlag,
					AddHostFlag:        test.hostFlag,
					AddPortFlag:        test.portFlag,
				},
			}

			serviceName, socket, host, port, err := processGlobalAddFlagsWithSocket(cmd, cmd.AddCommonFlags)

			assert.NoError(t, err)
			assert.Equal(t, serviceName, test.wantServiceName)
			assert.Equal(t, host, test.wantHost)
			assert.Equal(t, int(port), int(test.wantPort))
			assert.Equal(t, socket, test.wantSocket)
		})
	}
}
