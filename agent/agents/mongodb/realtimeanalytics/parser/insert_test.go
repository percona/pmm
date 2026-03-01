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

var dataInsert = []byte(`
  {
    "type": "op",
    "host": "c4486b1ebd30:27017",
    "desc": "conn14823",
    "connectionId": 14823,
    "client": "192.168.107.1:54514",
    "clientMetadata": {
      "driver": {
        "name": "mongo-go-driver",
        "version": "2.4.0"
      },
      "os": {
        "type": "darwin",
        "architecture": "arm64"
      },
      "platform": "go1.25.7"
    },
    "active": true,
    "currentOpTime": "2026-02-12T09:44:46.880+00:00",
    "effectiveUsers": [
      {
        "user": "root",
        "db": "admin"
      }
    ],
    "isFromUserConnection": true,
    "threaded": true,
    "opid": 1991397463,
    "lsid": {
      "id": {},
      "uid": "Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg="
    },
    "op": "command",
    "ns": "airline.$cmd",
    "redacted": false,
    "command": {
      "insert": "flights",
      "ordered": true,
      "writeConcern": { 
        "w": 2, 
        "j": true, 
        "wtimeout": 5000
      },
      "lsid": {
        "id": {}
      },
      "$db": "airline"
    },
    "numYields": 0,
    "queues": {
      "ingress": {
        "admissions": 0,
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

func Test_parseCommandInsert(t *testing.T) {
	t.Parallel()

	raw := parseBsonRaw(dataInsert)
	commandRaw, ok := raw.Lookup("command").DocumentOK()
	require.True(t, ok, "Expected to find 'command' field in raw BSON")

	result := parseCommandInsert(commandRaw)
	require.NotEmpty(t, result, "Expected non-empty result from parseCommandInsert")
	require.Contains(t, result, "db.flights.insert(?, {", "Expected fingerprint to contain 'db.flights.insert(?, {'")
}

func Benchmark_ParseCommandInsert(b *testing.B) {
	raw := parseBsonRaw(dataInsert)
	commandRaw, _ := raw.Lookup("command").DocumentOK()

	for b.Loop() {
		_ = parseCommandInsert(commandRaw)
	}
}
