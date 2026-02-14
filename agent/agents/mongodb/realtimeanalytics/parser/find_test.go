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

var dataFind = []byte(`
  {
    "type": "op",
    "host": "c4486b1ebd30:27017",
    "desc": "conn14544",
    "connectionId": 14544,
    "client": "192.168.107.1:33122",
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
    "currentOpTime": "2026-02-11T19:34:56.677+00:00",
    "effectiveUsers": [
      {
        "user": "root",
        "db": "admin"
      }
    ],
    "isFromUserConnection": true,
    "threaded": true,
    "opid": -2024364589,
    "lsid": {
      "id": {},
      "uid": "Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg="
    },
    "secs_running": {
      "low": 0,
      "high": 0,
      "unsigned": false
    },
    "microsecs_running": {
      "low": 60,
      "high": 0,
      "unsigned": false
    },
    "op": "query",
    "ns": "airline.flights",
    "redacted": false,
    "command": {
      "find": "flights",
      "batchSize": 1,
      "filter": {
        "flight_id": 880
      },
      "limit": {
        "low": 5,
        "high": 0,
        "unsigned": false
      },
      "projection": {
        "origin": 1,
        "destination": 1,
        "gate": 1,
        "_id": 0,
        "flight_id": 1
      },
      "lsid": {
        "id": {}
      },
      "$db": "airline"
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
        "admissions": 1,
        "totalTimeQueuedMicros": {
          "low": 0,
          "high": 0,
          "unsigned": false
        }
      }
    },
    "currentQueue": null,
    "locks": {
      "Global": "r"
    },
    "waitingForLock": false,
    "lockStats": {
      "Global": {
        "acquireCount": {
          "r": {
            "low": 1,
            "high": 0,
            "unsigned": false
          }
        }
      }
    },
    "waitingForFlowControl": false,
    "flowControlStats": {}
  },
  {
    "type": "op",
    "host": "c4486b1ebd30:27017",
    "desc": "conn14550",
    "connectionId": 14550,
    "client": "192.168.107.1:33188",
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
    "currentOpTime": "2026-02-11T19:34:56.677+00:00",
    "effectiveUsers": [
      {
        "user": "root",
        "db": "admin"
      }
    ],
    "isFromUserConnection": true,
    "threaded": true,
    "opid": -2024404519,
    "lsid": {
      "id": {},
      "uid": "Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg="
    },
    "secs_running": {
      "low": 0,
      "high": 0,
      "unsigned": false
    },
    "microsecs_running": {
      "low": 79,
      "high": 0,
      "unsigned": false
    },
    "op": "query",
    "ns": "airline.flights",
    "redacted": false,
    "command": {
      "find": "flights",
      "batchSize": 1,
      "filter": {
        "flight_id": 347
      },
      "limit": {
        "low": 5,
        "high": 0,
        "unsigned": false
      },
      "projection": {
        "origin": 1,
        "destination": 1,
        "gate": 1,
        "_id": 0,
        "flight_id": 1
      },
      "lsid": {
        "id": {}
      },
      "$db": "airline"
    },
    "queryFramework": "classic",
    "planSummary": "IXSCAN { flight_id: 1, equipment.plane_type: 1 }",
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
        "admissions": 1,
        "totalTimeQueuedMicros": {
          "low": 0,
          "high": 0,
          "unsigned": false
        }
      }
    },
    "currentQueue": null,
    "queryShapeHash": "C0228CF7207E860A0A83C3786D305F60DD43415739437F97C021D2CA37918A2D",
    "locks": {},
    "waitingForLock": false,
    "lockStats": {
      "Global": {
        "acquireCount": {
          "r": {
            "low": 1,
            "high": 0,
            "unsigned": false
          }
        }
      }
    },
    "waitingForFlowControl": false,
    "flowControlStats": {}
  }
`)

func Test_parseCommandFind(t *testing.T) {
	t.Parallel()

	raw := parseBsonRaw(dataFind)
	commandRaw, ok := raw.Lookup("command").DocumentOK()
	require.True(t, ok, "Expected to find 'command' field in raw BSON")

	result := parseCommandFind(commandRaw)
	println("Parsed command fingerprint:", result)
	require.NotEmpty(t, result, "Expected non-empty result from parseCommandFind")
	require.Contains(t, result, "db.flights.find({", "Expected fingerprint to contain 'db.flights.find({'")
}

func Benchmark_ParseCommandFind(b *testing.B) {
	raw := parseBsonRaw(dataFind)
	commandRaw, _ := raw.Lookup("command").DocumentOK()

	for b.Loop() {
		_ = parseCommandFind(commandRaw)
	}
}
