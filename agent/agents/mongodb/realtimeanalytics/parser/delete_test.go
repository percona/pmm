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

var dataDelete = []byte(`
{
    "type": "op",
    "host": "c4486b1ebd30:27017",
    "desc": "conn14886",
    "connectionId": 14886,
    "client": "192.168.107.1:59634",
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
    "currentOpTime": "2026-02-12T15:32:07.666+00:00",
    "effectiveUsers": [
      {
        "user": "root",
        "db": "admin"
      }
    ],
    "isFromUserConnection": true,
    "threaded": true,
    "opid": 606573938,
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
      "low": 25,
      "high": 0,
      "unsigned": false
    },
    "op": "remove",
    "ns": "airline.flights",
    "redacted": false,
    "command": {
      "q": {
        "_id": "val-918"
      },
"writeConcern": { 
        "w": 2, 
        "j": true, 
        "wtimeout": 5000
      },
      "limit": 1
    },
    "planSummary": "EXPRESS_IXSCAN { _id: 1 },EXPRESS_DELETE",
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
      "ReplicationStateTransition": "w",
      "Global": "w",
      "Database": "w",
      "Collection": "w"
    },
    "waitingForLock": false,
    "lockStats": {
      "ReplicationStateTransition": {
        "acquireCount": {
          "w": {
            "low": 1,
            "high": 0,
            "unsigned": false
          }
        }
      },
      "Global": {
        "acquireCount": {
          "w": {
            "low": 1,
            "high": 0,
            "unsigned": false
          }
        }
      },
      "Database": {
        "acquireCount": {
          "w": {
            "low": 1,
            "high": 0,
            "unsigned": false
          }
        }
      },
      "Collection": {
        "acquireCount": {
          "w": {
            "low": 1,
            "high": 0,
            "unsigned": false
          }
        }
      }
    },
    "waitingForFlowControl": false,
    "flowControlStats": {
      "acquireCount": {
        "low": 1,
        "high": 0,
        "unsigned": false
      }
    }
  }
`)

func Test_parseCommandDelete(t *testing.T) {
	t.Parallel()

	raw := parseBsonRaw(dataDelete)
	commandRaw, ok := raw.Lookup("command").DocumentOK()
	require.True(t, ok, "Expected to find 'command' field in raw BSON")

	ns, ok := raw.Lookup("ns").StringValueOK()
	require.True(t, ok, "Expected to find 'ns' field in raw BSON")

	result := parseCommandDelete(commandRaw, ns)
	println(result)
	require.NotEmpty(t, result, "Expected non-empty result from parseCommandDelete")
	require.Contains(t, result, "db.airline.flights.deleteOne({", "Expected fingerprint to contain 'db.airline.flights.deleteOne({'")
}

func Benchmark_ParseCommandDelete(b *testing.B) {
	raw := parseBsonRaw(dataDelete)
	commandRaw, _ := raw.Lookup("command").DocumentOK()
	ns, _ := raw.Lookup("ns").StringValueOK()

	for b.Loop() {
		_ = parseCommandDelete(commandRaw, ns)
	}
}
