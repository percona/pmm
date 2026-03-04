// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var dataCommand = []byte(`
{
    "type": "op",
    "host": "c4486b1ebd30:27017",
    "desc": "conn15031",
    "connectionId": 15031,
    "client": "127.0.0.1:48192",
    "appName": "mongosh 2.5.10",
    "clientMetadata": {
      "application": {
        "name": "mongosh 2.5.10"
      },
      "driver": {
        "name": "nodejs|mongosh",
        "version": "6.19.0|2.5.10"
      },
      "platform": "Node.js v20.19.6, LE",
      "os": {
        "name": "linux",
        "architecture": "arm64",
        "version": "6.1.0-41-arm64",
        "type": "Linux"
      },
      "env": {
        "container": {
          "runtime": "docker"
        }
      }
    },
    "active": true,
    "currentOpTime": "2026-02-13T10:45:24.575+00:00",
    "isFromUserConnection": true,
    "threaded": true,
    "opid": 1488428356,
    "secs_running": {
      "low": 6,
      "high": 0,
      "unsigned": false
    },
    "microsecs_running": {
      "low": 6378282,
      "high": 0,
      "unsigned": false
    },
    "op": "command",
    "ns": "admin.$cmd",
    "redacted": false,
    "command": {
      "hello": 1,
      "maxAwaitTimeMS": 10000,
      "topologyVersion": {
        "processId": {},
        "counter": {
          "low": 0,
          "high": 0,
          "unsigned": false
        }
      },
      "$db": "admin"
    },
    "numYields": 0,
    "queues": {
      "ingress": {
        "admissions": 1,
        "totalTimeQueuedMicros": {
          "low": 0,
          "high": 0,
          "unsigned": false
        }
      },
      "execution": {
        "admissions": 0,
        "totalTimeQueuedMicros": {
          "low": 0,
          "high": 0,
          "unsigned": false
        }
      }
    },
    "currentQueue": null,
    "locks": {},
    "waitingForLock": false,
    "lockStats": {},
    "waitingForFlowControl": false,
    "flowControlStats": {}
  }
`)

func Test_parseCommand(t *testing.T) {
	t.Parallel()

	raw := parseBsonRaw(dataCommand)
	result := parseCommand(raw)
	require.NotEmpty(t, result, "Expected non-empty result from parseCommand")
	require.Contains(t, result, "db.runCommand({", "Expected fingerprint to contain 'db.runCommand({'")
}

func Benchmark(b *testing.B) {
	raw := parseBsonRaw(dataAggregate)
	for b.Loop() {
		parseCommand(raw)
	}
}
