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
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var dataAggregate = []byte(`
{
    "type": "op",
    "host": "c4486b1ebd30:27017",
    "desc": "conn14811",
    "connectionId": 14811,
    "client": "192.168.107.1:44684",
    "appName": "DataGrip",
    "clientMetadata": {
      "application": {
        "name": "DataGrip"
      },
      "driver": {
        "name": "mongo-java-driver|sync",
        "version": "4.11.1"
      },
      "os": {
        "type": "Darwin",
        "name": "Mac OS X",
        "architecture": "aarch64",
        "version": "26.2"
      },
      "platform": "Java/JetBrains s.r.o./21.0.9+10-b1163.86"
    },
    "active": true,
    "currentOpTime": "2026-02-12T16:30:24.505+00:00",
    "effectiveUsers": [
      {
        "user": "root",
        "db": "admin"
      }
    ],
    "isFromUserConnection": true,
    "threaded": true,
    "opid": 1626132511,
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
      "low": 151,
      "high": 0,
      "unsigned": false
    },
    "op": "command",
    "ns": "admin.$cmd.aggregate",
    "redacted": false,
    "command": {
      "aggregate": 1,
      "pipeline": [
        {
          "$currentOp": {
            "allUsers": true,
            "idleSessions": false,
            "idleCursors": false,
            "idleConnections": false
          }
        },
        {
          "$match": {
            "$and": [
              {
                "appName": {
                  "$not": {
                    "$regex": "^(RTA-mongodb-.*$)"
                  }
                }
              },
              {
                "desc": {
                  "$nin": [
                    "Checkpointer",
                    "JournalFlusher"
                  ]
                }
              },
              {
                "active": true
              }
            ]
          }
        }
      ],
      "cursor": {},
      "$db": "admin",
      "lsid": {
        "id": {}
      }
    },
    "queryFramework": "classic",
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

func parseBsonRaw(data []byte) bson.Raw {
	vr, err := bson.NewExtJSONValueReader(bytes.NewReader(data), true)
	if err != nil {
		panic(err)
	}
	decoder := bson.NewDecoder(vr)

	var raw bson.Raw
	err = decoder.Decode(&raw)
	if err != nil {
		panic(err)
	}
	return raw
}

func Test_parseCommandAggregate(t *testing.T) {
	t.Parallel()

	raw := parseBsonRaw(dataAggregate)
	commandRaw, ok := raw.Lookup("command").DocumentOK()
	require.True(t, ok, "Expected to find 'command' field in raw BSON")

	ns, ok := raw.Lookup("ns").StringValueOK()
	require.True(t, ok, "Expected to find 'ns' field in raw BSON")

	result := parseCommandAggregate(commandRaw, ns)
	require.NotEmpty(t, result, "Expected non-empty result from parseCommandAggregate")
	require.Contains(t, result, "admin.$cmd.aggregate", "Expected fingerprint to contain 'admin.$cmd.aggregate'")
}

func Benchmark_ParseCommandAggregate(b *testing.B) {
	raw := parseBsonRaw(dataAggregate)
	commandRaw, _ := raw.Lookup("command").DocumentOK()
	ns, _ := raw.Lookup("ns").StringValueOK()

	for b.Loop() {
		_ = parseCommandAggregate(commandRaw, ns)
	}
}
