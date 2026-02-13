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
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/types/known/timestamppb"

	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

func TestParseCurrentOp(t *testing.T) {
	t.Parallel()

	aggTime, _ := time.Parse(time.RFC3339, "2026-02-12T16:30:24.505Z")
	cmdTime, _ := time.Parse(time.RFC3339, "2026-02-13T10:45:24.575Z")
	deleteTime, _ := time.Parse(time.RFC3339, "2026-02-12T15:32:07.666Z")
	findTime, _ := time.Parse(time.RFC3339, "2026-02-11T19:34:56.677Z")
	insertTime, _ := time.Parse(time.RFC3339, "2026-02-12T09:44:46.880Z")
	updateTime, _ := time.Parse(time.RFC3339, "2026-02-12T10:45:40.456Z")
	tests := []struct {
		name      string
		raw       bson.Raw
		wantQData *rtav1.QueryData
	}{
		{
			name: "valid commandAggregate",
			raw:  parseBsonRaw(dataAggregate),
			wantQData: &rtav1.QueryData{
				QueryId:       "1626132511",
				QueryText:     "admin.$cmd.aggregate([\n    {\n        \"$currentOp\": {\n            \"allUsers\": true,\n            \"idleSessions\": false,\n            \"idleCursors\": false,\n            \"idleConnections\": false\n        }\n    },\n    {\n        \"$match\": {\n            \"$and\": [\n                {\n                    \"appName\": {\n                        \"$not\": {\n                            \"$regex\": \"^(RTA-mongodb-.*$)\"\n                        }\n                    }\n                },\n                {\n                    \"desc\": {\n                        \"$nin\": [\n                            \"Checkpointer\",\n                            \"JournalFlusher\"\n                        ]\n                    }\n                },\n                {\n                    \"active\": true\n                }\n            ]\n        }\n    }\n], {\"cursor\":{}})",
				QueryRawJson:  "{\n    \"type\": \"op\",\n    \"host\": \"c4486b1ebd30:27017\",\n    \"desc\": \"conn14811\",\n    \"connectionId\": 14811,\n    \"client\": \"192.168.107.1:44684\",\n    \"appName\": \"DataGrip\",\n    \"clientMetadata\": {\n        \"application\": {\n            \"name\": \"DataGrip\"\n        },\n        \"driver\": {\n            \"name\": \"mongo-java-driver|sync\",\n            \"version\": \"4.11.1\"\n        },\n        \"os\": {\n            \"type\": \"Darwin\",\n            \"name\": \"Mac OS X\",\n            \"architecture\": \"aarch64\",\n            \"version\": \"26.2\"\n        },\n        \"platform\": \"Java/JetBrains s.r.o./21.0.9+10-b1163.86\"\n    },\n    \"active\": true,\n    \"currentOpTime\": \"2026-02-12T16:30:24.505+00:00\",\n    \"effectiveUsers\": [\n        {\n            \"user\": \"root\",\n            \"db\": \"admin\"\n        }\n    ],\n    \"isFromUserConnection\": true,\n    \"threaded\": true,\n    \"opid\": 1626132511,\n    \"lsid\": {\n        \"id\": {},\n        \"uid\": \"Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg=\"\n    },\n    \"secs_running\": {\n        \"low\": 0,\n        \"high\": 0,\n        \"unsigned\": false\n    },\n    \"microsecs_running\": {\n        \"low\": 151,\n        \"high\": 0,\n        \"unsigned\": false\n    },\n    \"op\": \"command\",\n    \"ns\": \"admin.$cmd.aggregate\",\n    \"redacted\": false,\n    \"command\": {\n        \"aggregate\": 1,\n        \"pipeline\": [\n            {\n                \"$currentOp\": {\n                    \"allUsers\": true,\n                    \"idleSessions\": false,\n                    \"idleCursors\": false,\n                    \"idleConnections\": false\n                }\n            },\n            {\n                \"$match\": {\n                    \"$and\": [\n                        {\n                            \"appName\": {\n                                \"$not\": {\n                                    \"$regex\": \"^(RTA-mongodb-.*$)\"\n                                }\n                            }\n                        },\n                        {\n                            \"desc\": {\n                                \"$nin\": [\n                                    \"Checkpointer\",\n                                    \"JournalFlusher\"\n                                ]\n                            }\n                        },\n                        {\n                            \"active\": true\n                        }\n                    ]\n                }\n            }\n        ],\n        \"cursor\": {},\n        \"$db\": \"admin\",\n        \"lsid\": {\n            \"id\": {}\n        }\n    },\n    \"queryFramework\": \"classic\",\n    \"numYields\": 0,\n    \"queues\": {\n        \"ingress\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"low\": 0,\n                \"high\": 0,\n                \"unsigned\": false\n            }\n        },\n        \"execution\": {\n            \"admissions\": 0,\n            \"totalTimeQueuedMicros\": {\n                \"low\": 0,\n                \"high\": 0,\n                \"unsigned\": false\n            }\n        }\n    },\n    \"currentQueue\": null,\n    \"locks\": {},\n    \"waitingForLock\": false,\n    \"lockStats\": {},\n    \"waitingForFlowControl\": false,\n    \"flowControlStats\": {}\n}",
				ClientAddress: "192.168.107.1:44684",
				Payload: &rtav1.QueryData_MongoDbPayload{
					MongoDbPayload: &rtav1.QueryMongoDBData{
						DbInstanceAddress:  "c4486b1ebd30:27017",
						ClientAppName:      "DataGrip",
						DatabaseName:       "admin",
						Collection:         "$cmd",
						Operation:          "command",
						OperationStartTime: timestamppb.New(aggTime),
						Username:           "root",
					},
				},
			},
		},
		{
			name: "valid command",
			raw:  parseBsonRaw(dataCommand),
			wantQData: &rtav1.QueryData{
				QueryId:       "1488428356",
				QueryText:     "db.runCommand({\n    \"hello\": 1,\n    \"maxAwaitTimeMS\": 10000,\n    \"topologyVersion\": {\n        \"processId\": {},\n        \"counter\": {\n            \"low\": 0,\n            \"high\": 0,\n            \"unsigned\": false\n        }\n    },\n    \"$db\": \"admin\"\n})",
				QueryRawJson:  "{\n    \"type\": \"op\",\n    \"host\": \"c4486b1ebd30:27017\",\n    \"desc\": \"conn15031\",\n    \"connectionId\": 15031,\n    \"client\": \"127.0.0.1:48192\",\n    \"appName\": \"mongosh 2.5.10\",\n    \"clientMetadata\": {\n        \"application\": {\n            \"name\": \"mongosh 2.5.10\"\n        },\n        \"driver\": {\n            \"name\": \"nodejs|mongosh\",\n            \"version\": \"6.19.0|2.5.10\"\n        },\n        \"platform\": \"Node.js v20.19.6, LE\",\n        \"os\": {\n            \"name\": \"linux\",\n            \"architecture\": \"arm64\",\n            \"version\": \"6.1.0-41-arm64\",\n            \"type\": \"Linux\"\n        },\n        \"env\": {\n            \"container\": {\n                \"runtime\": \"docker\"\n            }\n        }\n    },\n    \"active\": true,\n    \"currentOpTime\": \"2026-02-13T10:45:24.575+00:00\",\n    \"isFromUserConnection\": true,\n    \"threaded\": true,\n    \"opid\": 1488428356,\n    \"secs_running\": {\n        \"low\": 6,\n        \"high\": 0,\n        \"unsigned\": false\n    },\n    \"microsecs_running\": {\n        \"low\": 6378282,\n        \"high\": 0,\n        \"unsigned\": false\n    },\n    \"op\": \"command\",\n    \"ns\": \"admin.$cmd\",\n    \"redacted\": false,\n    \"command\": {\n        \"hello\": 1,\n        \"maxAwaitTimeMS\": 10000,\n        \"topologyVersion\": {\n            \"processId\": {},\n            \"counter\": {\n                \"low\": 0,\n                \"high\": 0,\n                \"unsigned\": false\n            }\n        },\n        \"$db\": \"admin\"\n    },\n    \"numYields\": 0,\n    \"queues\": {\n        \"ingress\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"low\": 0,\n                \"high\": 0,\n                \"unsigned\": false\n            }\n        },\n        \"execution\": {\n            \"admissions\": 0,\n            \"totalTimeQueuedMicros\": {\n                \"low\": 0,\n                \"high\": 0,\n                \"unsigned\": false\n            }\n        }\n    },\n    \"currentQueue\": null,\n    \"locks\": {},\n    \"waitingForLock\": false,\n    \"lockStats\": {},\n    \"waitingForFlowControl\": false,\n    \"flowControlStats\": {}\n}",
				ClientAddress: "127.0.0.1:48192",
				Payload: &rtav1.QueryData_MongoDbPayload{
					MongoDbPayload: &rtav1.QueryMongoDBData{
						DbInstanceAddress:  "c4486b1ebd30:27017",
						ClientAppName:      "mongosh 2.5.10",
						DatabaseName:       "admin",
						Operation:          "command",
						OperationStartTime: timestamppb.New(cmdTime),
					},
				},
			},
		},
		{
			name: "valid commandDelete",
			raw:  parseBsonRaw(dataDelete),
			wantQData: &rtav1.QueryData{
				QueryId:       "606573938",
				QueryText:     "db.flights.deleteOne({\n    \"_id\": \"val-918\"\n}, {\"writeConcern\":{\"w\":2,\"j\":true,\"wtimeout\":5000}})",
				QueryRawJson:  "{\n    \"type\": \"op\",\n    \"host\": \"c4486b1ebd30:27017\",\n    \"desc\": \"conn14886\",\n    \"connectionId\": 14886,\n    \"client\": \"192.168.107.1:59634\",\n    \"clientMetadata\": {\n        \"driver\": {\n            \"name\": \"mongo-go-driver\",\n            \"version\": \"2.4.0\"\n        },\n        \"os\": {\n            \"type\": \"darwin\",\n            \"architecture\": \"arm64\"\n        },\n        \"platform\": \"go1.25.7\"\n    },\n    \"active\": true,\n    \"currentOpTime\": \"2026-02-12T15:32:07.666+00:00\",\n    \"effectiveUsers\": [\n        {\n            \"user\": \"root\",\n            \"db\": \"admin\"\n        }\n    ],\n    \"isFromUserConnection\": true,\n    \"threaded\": true,\n    \"opid\": 606573938,\n    \"lsid\": {\n        \"id\": {},\n        \"uid\": \"Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg=\"\n    },\n    \"secs_running\": {\n        \"low\": 0,\n        \"high\": 0,\n        \"unsigned\": false\n    },\n    \"microsecs_running\": {\n        \"low\": 25,\n        \"high\": 0,\n        \"unsigned\": false\n    },\n    \"op\": \"remove\",\n    \"ns\": \"airline.flights\",\n    \"redacted\": false,\n    \"command\": {\n        \"q\": {\n            \"_id\": \"val-918\"\n        },\n        \"writeConcern\": {\n            \"w\": 2,\n            \"j\": true,\n            \"wtimeout\": 5000\n        },\n        \"limit\": 1\n    },\n    \"planSummary\": \"EXPRESS_IXSCAN { _id: 1 },EXPRESS_DELETE\",\n    \"numYields\": 0,\n    \"queues\": {\n        \"ingress\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"low\": 0,\n                \"high\": 0,\n                \"unsigned\": false\n            }\n        },\n        \"execution\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"low\": 0,\n                \"high\": 0,\n                \"unsigned\": false\n            }\n        }\n    },\n    \"currentQueue\": null,\n    \"locks\": {\n        \"ReplicationStateTransition\": \"w\",\n        \"Global\": \"w\",\n        \"Database\": \"w\",\n        \"Collection\": \"w\"\n    },\n    \"waitingForLock\": false,\n    \"lockStats\": {\n        \"ReplicationStateTransition\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"low\": 1,\n                    \"high\": 0,\n                    \"unsigned\": false\n                }\n            }\n        },\n        \"Global\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"low\": 1,\n                    \"high\": 0,\n                    \"unsigned\": false\n                }\n            }\n        },\n        \"Database\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"low\": 1,\n                    \"high\": 0,\n                    \"unsigned\": false\n                }\n            }\n        },\n        \"Collection\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"low\": 1,\n                    \"high\": 0,\n                    \"unsigned\": false\n                }\n            }\n        }\n    },\n    \"waitingForFlowControl\": false,\n    \"flowControlStats\": {\n        \"acquireCount\": {\n            \"low\": 1,\n            \"high\": 0,\n            \"unsigned\": false\n        }\n    }\n}",
				ClientAddress: "192.168.107.1:59634",
				Payload: &rtav1.QueryData_MongoDbPayload{
					MongoDbPayload: &rtav1.QueryMongoDBData{
						DbInstanceAddress:  "c4486b1ebd30:27017",
						ClientAppName:      "",
						DatabaseName:       "airline",
						Collection:         "flights",
						Operation:          "remove",
						OperationStartTime: timestamppb.New(deleteTime),
						Username:           "root",
						PlanSummary:        "EXPRESS_IXSCAN { _id: 1 },EXPRESS_DELETE",
					},
				},
			},
		},
		{
			name: "valid commandFind",
			raw:  parseBsonRaw(dataFind),
			wantQData: &rtav1.QueryData{
				QueryId:       "-2024364589",
				QueryText:     "db.flights.find({\n    \"flight_id\": 880\n}, {\n    \"origin\": 1,\n    \"destination\": 1,\n    \"gate\": 1,\n    \"_id\": 0,\n    \"flight_id\": 1\n}, {\"limit\":{\"low\":5,\"high\":0,\"unsigned\":false}}).limit({\n    \"low\": 5,\n    \"high\": 0,\n    \"unsigned\": false\n}).batchSize(1)",
				QueryRawJson:  "{\n    \"type\": \"op\",\n    \"host\": \"c4486b1ebd30:27017\",\n    \"desc\": \"conn14544\",\n    \"connectionId\": 14544,\n    \"client\": \"192.168.107.1:33122\",\n    \"clientMetadata\": {\n        \"driver\": {\n            \"name\": \"mongo-go-driver\",\n            \"version\": \"2.4.0\"\n        },\n        \"os\": {\n            \"type\": \"darwin\",\n            \"architecture\": \"arm64\"\n        },\n        \"platform\": \"go1.25.7\"\n    },\n    \"active\": true,\n    \"currentOpTime\": \"2026-02-11T19:34:56.677+00:00\",\n    \"effectiveUsers\": [\n        {\n            \"user\": \"root\",\n            \"db\": \"admin\"\n        }\n    ],\n    \"isFromUserConnection\": true,\n    \"threaded\": true,\n    \"opid\": -2024364589,\n    \"lsid\": {\n        \"id\": {},\n        \"uid\": \"Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg=\"\n    },\n    \"secs_running\": {\n        \"low\": 0,\n        \"high\": 0,\n        \"unsigned\": false\n    },\n    \"microsecs_running\": {\n        \"low\": 60,\n        \"high\": 0,\n        \"unsigned\": false\n    },\n    \"op\": \"query\",\n    \"ns\": \"airline.flights\",\n    \"redacted\": false,\n    \"command\": {\n        \"find\": \"flights\",\n        \"batchSize\": 1,\n        \"filter\": {\n            \"flight_id\": 880\n        },\n        \"limit\": {\n            \"low\": 5,\n            \"high\": 0,\n            \"unsigned\": false\n        },\n        \"projection\": {\n            \"origin\": 1,\n            \"destination\": 1,\n            \"gate\": 1,\n            \"_id\": 0,\n            \"flight_id\": 1\n        },\n        \"lsid\": {\n            \"id\": {}\n        },\n        \"$db\": \"airline\"\n    },\n    \"numYields\": 0,\n    \"queues\": {\n        \"ingress\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"low\": 0,\n                \"high\": 0,\n                \"unsigned\": false\n            }\n        },\n        \"execution\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"low\": 0,\n                \"high\": 0,\n                \"unsigned\": false\n            }\n        }\n    },\n    \"currentQueue\": null,\n    \"locks\": {\n        \"Global\": \"r\"\n    },\n    \"waitingForLock\": false,\n    \"lockStats\": {\n        \"Global\": {\n            \"acquireCount\": {\n                \"r\": {\n                    \"low\": 1,\n                    \"high\": 0,\n                    \"unsigned\": false\n                }\n            }\n        }\n    },\n    \"waitingForFlowControl\": false,\n    \"flowControlStats\": {}\n}",
				ClientAddress: "192.168.107.1:33122",
				Payload: &rtav1.QueryData_MongoDbPayload{
					MongoDbPayload: &rtav1.QueryMongoDBData{
						DbInstanceAddress:  "c4486b1ebd30:27017",
						DatabaseName:       "airline",
						Collection:         "flights",
						Operation:          "query",
						OperationStartTime: timestamppb.New(findTime),
						Username:           "root",
					},
				},
			},
		},
		{
			name: "valid commandInsert",
			raw:  parseBsonRaw(dataInsert),
			wantQData: &rtav1.QueryData{
				QueryId:       "1991397463",
				QueryText:     "db.runCommand({\n    \"insert\": \"flights\",\n    \"ordered\": true,\n    \"writeConcern\": {\n        \"w\": 2,\n        \"j\": true,\n        \"wtimeout\": 5000\n    },\n    \"lsid\": {\n        \"id\": {}\n    },\n    \"$db\": \"airline\"\n})",
				QueryRawJson:  "{\n    \"type\": \"op\",\n    \"host\": \"c4486b1ebd30:27017\",\n    \"desc\": \"conn14823\",\n    \"connectionId\": 14823,\n    \"client\": \"192.168.107.1:54514\",\n    \"clientMetadata\": {\n        \"driver\": {\n            \"name\": \"mongo-go-driver\",\n            \"version\": \"2.4.0\"\n        },\n        \"os\": {\n            \"type\": \"darwin\",\n            \"architecture\": \"arm64\"\n        },\n        \"platform\": \"go1.25.7\"\n    },\n    \"active\": true,\n    \"currentOpTime\": \"2026-02-12T09:44:46.880+00:00\",\n    \"effectiveUsers\": [\n        {\n            \"user\": \"root\",\n            \"db\": \"admin\"\n        }\n    ],\n    \"isFromUserConnection\": true,\n    \"threaded\": true,\n    \"opid\": 1991397463,\n    \"lsid\": {\n        \"id\": {},\n        \"uid\": \"Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg=\"\n    },\n    \"op\": \"command\",\n    \"ns\": \"airline.$cmd\",\n    \"redacted\": false,\n    \"command\": {\n        \"insert\": \"flights\",\n        \"ordered\": true,\n        \"writeConcern\": {\n            \"w\": 2,\n            \"j\": true,\n            \"wtimeout\": 5000\n        },\n        \"lsid\": {\n            \"id\": {}\n        },\n        \"$db\": \"airline\"\n    },\n    \"numYields\": 0,\n    \"queues\": {\n        \"ingress\": {\n            \"admissions\": 0,\n            \"totalTimeQueuedMicros\": {\n                \"low\": 0,\n                \"high\": 0,\n                \"unsigned\": false\n            }\n        },\n        \"execution\": {\n            \"admissions\": 0,\n            \"totalTimeQueuedMicros\": {\n                \"low\": 0,\n                \"high\": 0,\n                \"unsigned\": false\n            }\n        }\n    },\n    \"currentQueue\": null,\n    \"locks\": {},\n    \"waitingForLock\": false,\n    \"lockStats\": {},\n    \"waitingForFlowControl\": false,\n    \"flowControlStats\": {}\n}",
				ClientAddress: "192.168.107.1:54514",
				Payload: &rtav1.QueryData_MongoDbPayload{
					MongoDbPayload: &rtav1.QueryMongoDBData{
						DbInstanceAddress:  "c4486b1ebd30:27017",
						DatabaseName:       "airline",
						Operation:          "command",
						OperationStartTime: timestamppb.New(insertTime),
						Username:           "root",
					},
				},
			},
		},
		{
			name: "valid commandUpdate",
			raw:  parseBsonRaw(dataUpdate),
			wantQData: &rtav1.QueryData{
				QueryId:       "-941663415",
				QueryText:     "db.flights.update({\n    \"_id\": \"val-669\"\n}, {\n    \"$set\": {\n        \"duration_minutes\": 216\n    }\n}, {\"multi\":false,\"upsert\":false})",
				QueryRawJson:  "{\n    \"type\": \"op\",\n    \"host\": \"c4486b1ebd30:27017\",\n    \"desc\": \"conn14858\",\n    \"connectionId\": 14858,\n    \"client\": \"192.168.107.1:46310\",\n    \"clientMetadata\": {\n        \"driver\": {\n            \"name\": \"mongo-go-driver\",\n            \"version\": \"2.4.0\"\n        },\n        \"os\": {\n            \"type\": \"darwin\",\n            \"architecture\": \"arm64\"\n        },\n        \"platform\": \"go1.25.7\"\n    },\n    \"active\": true,\n    \"currentOpTime\": \"2026-02-12T10:45:40.456+00:00\",\n    \"effectiveUsers\": [\n        {\n            \"user\": \"root\",\n            \"db\": \"admin\"\n        }\n    ],\n    \"isFromUserConnection\": true,\n    \"threaded\": true,\n    \"opid\": -941663415,\n    \"lsid\": {\n        \"id\": {},\n        \"uid\": \"Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg=\"\n    },\n    \"secs_running\": {\n        \"low\": 0,\n        \"high\": 0,\n        \"unsigned\": false\n    },\n    \"microsecs_running\": {\n        \"low\": 119,\n        \"high\": 0,\n        \"unsigned\": false\n    },\n    \"op\": \"update\",\n    \"ns\": \"airline.flights\",\n    \"redacted\": false,\n    \"command\": {\n        \"q\": {\n            \"_id\": \"val-669\"\n        },\n        \"u\": {\n            \"$set\": {\n                \"duration_minutes\": 216\n            }\n        },\n        \"multi\": false,\n        \"upsert\": false\n    },\n    \"planSummary\": \"EXPRESS_IXSCAN { _id: 1 },EXPRESS_UPDATE\",\n    \"numYields\": 0,\n    \"queues\": {\n        \"ingress\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"low\": 0,\n                \"high\": 0,\n                \"unsigned\": false\n            }\n        },\n        \"execution\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"low\": 0,\n                \"high\": 0,\n                \"unsigned\": false\n            }\n        }\n    },\n    \"currentQueue\": null,\n    \"locks\": {},\n    \"waitingForLock\": false,\n    \"lockStats\": {\n        \"ReplicationStateTransition\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"low\": 1,\n                    \"high\": 0,\n                    \"unsigned\": false\n                }\n            }\n        },\n        \"Global\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"low\": 1,\n                    \"high\": 0,\n                    \"unsigned\": false\n                }\n            }\n        },\n        \"Database\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"low\": 1,\n                    \"high\": 0,\n                    \"unsigned\": false\n                }\n            }\n        },\n        \"Collection\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"low\": 1,\n                    \"high\": 0,\n                    \"unsigned\": false\n                }\n            }\n        }\n    },\n    \"waitingForFlowControl\": false,\n    \"flowControlStats\": {\n        \"acquireCount\": {\n            \"low\": 1,\n            \"high\": 0,\n            \"unsigned\": false\n        }\n    }\n}",
				ClientAddress: "192.168.107.1:46310",
				Payload: &rtav1.QueryData_MongoDbPayload{
					MongoDbPayload: &rtav1.QueryMongoDBData{
						DbInstanceAddress:  "c4486b1ebd30:27017",
						DatabaseName:       "airline",
						Collection:         "flights",
						Operation:          "update",
						OperationStartTime: timestamppb.New(updateTime),
						Username:           "root",
						PlanSummary:        "EXPRESS_IXSCAN { _id: 1 },EXPRESS_UPDATE",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotQData := ParseCurrentOp(tt.raw)
			require.True(t, reflect.DeepEqual(tt.wantQData, gotQData), "got: %v, want: %v", gotQData, tt.wantQData)
		})
	}
}

func Benchmark_ParseCurrentOp_Aggregate(b *testing.B) {
	raw := parseBsonRaw(dataAggregate)

	for b.Loop() {
		_ = ParseCurrentOp(raw)
	}
}

func Benchmark_ParseCurrentOp_Command(b *testing.B) {
	raw := parseBsonRaw(dataCommand)

	for b.Loop() {
		_ = ParseCurrentOp(raw)
	}
}

func Benchmark_ParseCurrentOp_Delete(b *testing.B) {
	raw := parseBsonRaw(dataDelete)

	for b.Loop() {
		_ = ParseCurrentOp(raw)
	}
}

func Benchmark_ParseCurrentOp_Find(b *testing.B) {
	raw := parseBsonRaw(dataFind)

	for b.Loop() {
		_ = ParseCurrentOp(raw)
	}
}

func Benchmark_ParseCurrentOp_Insert(b *testing.B) {
	raw := parseBsonRaw(dataInsert)

	for b.Loop() {
		_ = ParseCurrentOp(raw)
	}
}

func Benchmark_ParseCurrentOp_Update(b *testing.B) {
	raw := parseBsonRaw(dataUpdate)

	for b.Loop() {
		_ = ParseCurrentOp(raw)
	}
}