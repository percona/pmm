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
	"go.mongodb.org/mongo-driver/bson"
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
				QueryText:     "admin.$cmd.aggregate([\n    {\n        \"$currentOp\": {\n            \"allUsers\": true,\n            \"idleConnections\": false,\n            \"idleCursors\": false,\n            \"idleSessions\": false\n        }\n    },\n    {\n        \"$match\": {\n            \"$and\": [\n                {\n                    \"appName\": {\n                        \"$not\": {\n                            \"$regex\": \"^(rta-mongodb-.*$)\"\n                        }\n                    }\n                },\n                {\n                    \"desc\": {\n                        \"$nin\": [\n                            \"Checkpointer\",\n                            \"JournalFlusher\"\n                        ]\n                    }\n                },\n                {\n                    \"active\": true\n                }\n            ]\n        }\n    }\n], {\"cursor\":[]})",
				QueryRawJson:  "{\n    \"active\": true,\n    \"appName\": \"DataGrip\",\n    \"client\": \"192.168.107.1:44684\",\n    \"clientMetadata\": {\n        \"application\": {\n            \"name\": \"DataGrip\"\n        },\n        \"driver\": {\n            \"name\": \"mongo-java-driver|sync\",\n            \"version\": \"4.11.1\"\n        },\n        \"os\": {\n            \"architecture\": \"aarch64\",\n            \"name\": \"Mac OS X\",\n            \"type\": \"Darwin\",\n            \"version\": \"26.2\"\n        },\n        \"platform\": \"Java/JetBrains s.r.o./21.0.9+10-b1163.86\"\n    },\n    \"command\": {\n        \"$db\": \"admin\",\n        \"aggregate\": 1,\n        \"cursor\": {},\n        \"lsid\": {\n            \"id\": {}\n        },\n        \"pipeline\": [\n            {\n                \"$currentOp\": {\n                    \"allUsers\": true,\n                    \"idleConnections\": false,\n                    \"idleCursors\": false,\n                    \"idleSessions\": false\n                }\n            },\n            {\n                \"$match\": {\n                    \"$and\": [\n                        {\n                            \"appName\": {\n                                \"$not\": {\n                                    \"$regex\": \"^(rta-mongodb-.*$)\"\n                                }\n                            }\n                        },\n                        {\n                            \"desc\": {\n                                \"$nin\": [\n                                    \"Checkpointer\",\n                                    \"JournalFlusher\"\n                                ]\n                            }\n                        },\n                        {\n                            \"active\": true\n                        }\n                    ]\n                }\n            }\n        ]\n    },\n    \"connectionId\": 14811,\n    \"currentOpTime\": \"2026-02-12T16:30:24.505+00:00\",\n    \"currentQueue\": null,\n    \"desc\": \"conn14811\",\n    \"effectiveUsers\": [\n        {\n            \"db\": \"admin\",\n            \"user\": \"root\"\n        }\n    ],\n    \"flowControlStats\": {},\n    \"host\": \"c4486b1ebd30:27017\",\n    \"isFromUserConnection\": true,\n    \"lockStats\": {},\n    \"locks\": {},\n    \"lsid\": {\n        \"id\": {},\n        \"uid\": \"Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg=\"\n    },\n    \"microsecs_running\": {\n        \"high\": 0,\n        \"low\": 151,\n        \"unsigned\": false\n    },\n    \"ns\": \"admin.$cmd.aggregate\",\n    \"numYields\": 0,\n    \"op\": \"command\",\n    \"opid\": 1626132511,\n    \"queryFramework\": \"classic\",\n    \"queues\": {\n        \"execution\": {\n            \"admissions\": 0,\n            \"totalTimeQueuedMicros\": {\n                \"high\": 0,\n                \"low\": 0,\n                \"unsigned\": false\n            }\n        },\n        \"ingress\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"high\": 0,\n                \"low\": 0,\n                \"unsigned\": false\n            }\n        }\n    },\n    \"redacted\": false,\n    \"secs_running\": {\n        \"high\": 0,\n        \"low\": 0,\n        \"unsigned\": false\n    },\n    \"threaded\": true,\n    \"type\": \"op\",\n    \"waitingForFlowControl\": false,\n    \"waitingForLock\": false\n}",
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
				QueryText:     "db.runCommand({\n    \"$db\": \"admin\",\n    \"hello\": 1,\n    \"maxAwaitTimeMS\": 10000,\n    \"topologyVersion\": {\n        \"counter\": {\n            \"high\": 0,\n            \"low\": 0,\n            \"unsigned\": false\n        },\n        \"processId\": {}\n    }\n})",
				QueryRawJson:  "{\n    \"active\": true,\n    \"appName\": \"mongosh 2.5.10\",\n    \"client\": \"127.0.0.1:48192\",\n    \"clientMetadata\": {\n        \"application\": {\n            \"name\": \"mongosh 2.5.10\"\n        },\n        \"driver\": {\n            \"name\": \"nodejs|mongosh\",\n            \"version\": \"6.19.0|2.5.10\"\n        },\n        \"env\": {\n            \"container\": {\n                \"runtime\": \"docker\"\n            }\n        },\n        \"os\": {\n            \"architecture\": \"arm64\",\n            \"name\": \"linux\",\n            \"type\": \"Linux\",\n            \"version\": \"6.1.0-41-arm64\"\n        },\n        \"platform\": \"Node.js v20.19.6, LE\"\n    },\n    \"command\": {\n        \"$db\": \"admin\",\n        \"hello\": 1,\n        \"maxAwaitTimeMS\": 10000,\n        \"topologyVersion\": {\n            \"counter\": {\n                \"high\": 0,\n                \"low\": 0,\n                \"unsigned\": false\n            },\n            \"processId\": {}\n        }\n    },\n    \"connectionId\": 15031,\n    \"currentOpTime\": \"2026-02-13T10:45:24.575+00:00\",\n    \"currentQueue\": null,\n    \"desc\": \"conn15031\",\n    \"flowControlStats\": {},\n    \"host\": \"c4486b1ebd30:27017\",\n    \"isFromUserConnection\": true,\n    \"lockStats\": {},\n    \"locks\": {},\n    \"microsecs_running\": {\n        \"high\": 0,\n        \"low\": 6378282,\n        \"unsigned\": false\n    },\n    \"ns\": \"admin.$cmd\",\n    \"numYields\": 0,\n    \"op\": \"command\",\n    \"opid\": 1488428356,\n    \"queues\": {\n        \"execution\": {\n            \"admissions\": 0,\n            \"totalTimeQueuedMicros\": {\n                \"high\": 0,\n                \"low\": 0,\n                \"unsigned\": false\n            }\n        },\n        \"ingress\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"high\": 0,\n                \"low\": 0,\n                \"unsigned\": false\n            }\n        }\n    },\n    \"redacted\": false,\n    \"secs_running\": {\n        \"high\": 0,\n        \"low\": 6,\n        \"unsigned\": false\n    },\n    \"threaded\": true,\n    \"type\": \"op\",\n    \"waitingForFlowControl\": false,\n    \"waitingForLock\": false\n}",
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
				QueryText:     "db.flights.deleteOne({\n    \"_id\": \"val-918\"\n}, {\"writeConcern\":[{\"Key\":\"w\",\"Value\":2},{\"Key\":\"j\",\"Value\":true},{\"Key\":\"wtimeout\",\"Value\":5000}]})",
				QueryRawJson:  "{\n    \"active\": true,\n    \"client\": \"192.168.107.1:59634\",\n    \"clientMetadata\": {\n        \"driver\": {\n            \"name\": \"mongo-go-driver\",\n            \"version\": \"2.4.0\"\n        },\n        \"os\": {\n            \"architecture\": \"arm64\",\n            \"type\": \"darwin\"\n        },\n        \"platform\": \"go1.25.7\"\n    },\n    \"command\": {\n        \"limit\": 1,\n        \"q\": {\n            \"_id\": \"val-918\"\n        },\n        \"writeConcern\": {\n            \"j\": true,\n            \"w\": 2,\n            \"wtimeout\": 5000\n        }\n    },\n    \"connectionId\": 14886,\n    \"currentOpTime\": \"2026-02-12T15:32:07.666+00:00\",\n    \"currentQueue\": null,\n    \"desc\": \"conn14886\",\n    \"effectiveUsers\": [\n        {\n            \"db\": \"admin\",\n            \"user\": \"root\"\n        }\n    ],\n    \"flowControlStats\": {\n        \"acquireCount\": {\n            \"high\": 0,\n            \"low\": 1,\n            \"unsigned\": false\n        }\n    },\n    \"host\": \"c4486b1ebd30:27017\",\n    \"isFromUserConnection\": true,\n    \"lockStats\": {\n        \"Collection\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"high\": 0,\n                    \"low\": 1,\n                    \"unsigned\": false\n                }\n            }\n        },\n        \"Database\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"high\": 0,\n                    \"low\": 1,\n                    \"unsigned\": false\n                }\n            }\n        },\n        \"Global\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"high\": 0,\n                    \"low\": 1,\n                    \"unsigned\": false\n                }\n            }\n        },\n        \"ReplicationStateTransition\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"high\": 0,\n                    \"low\": 1,\n                    \"unsigned\": false\n                }\n            }\n        }\n    },\n    \"locks\": {\n        \"Collection\": \"w\",\n        \"Database\": \"w\",\n        \"Global\": \"w\",\n        \"ReplicationStateTransition\": \"w\"\n    },\n    \"lsid\": {\n        \"id\": {},\n        \"uid\": \"Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg=\"\n    },\n    \"microsecs_running\": {\n        \"high\": 0,\n        \"low\": 25,\n        \"unsigned\": false\n    },\n    \"ns\": \"airline.flights\",\n    \"numYields\": 0,\n    \"op\": \"remove\",\n    \"opid\": 606573938,\n    \"planSummary\": \"EXPRESS_IXSCAN { _id: 1 },EXPRESS_DELETE\",\n    \"queues\": {\n        \"execution\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"high\": 0,\n                \"low\": 0,\n                \"unsigned\": false\n            }\n        },\n        \"ingress\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"high\": 0,\n                \"low\": 0,\n                \"unsigned\": false\n            }\n        }\n    },\n    \"redacted\": false,\n    \"secs_running\": {\n        \"high\": 0,\n        \"low\": 0,\n        \"unsigned\": false\n    },\n    \"threaded\": true,\n    \"type\": \"op\",\n    \"waitingForFlowControl\": false,\n    \"waitingForLock\": false\n}",
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
				QueryText:     "db.flights.find({\n    \"flight_id\": 880\n}, {\n    \"_id\": 0,\n    \"destination\": 1,\n    \"flight_id\": 1,\n    \"gate\": 1,\n    \"origin\": 1\n}, {\"limit\":[{\"Key\":\"low\",\"Value\":5},{\"Key\":\"high\",\"Value\":0},{\"Key\":\"unsigned\",\"Value\":false}]}).limit([\n    {\n        \"Key\": \"low\",\n        \"Value\": 5\n    },\n    {\n        \"Key\": \"high\",\n        \"Value\": 0\n    },\n    {\n        \"Key\": \"unsigned\",\n        \"Value\": false\n    }\n]).batchSize(1)",
				QueryRawJson:  "{\n    \"active\": true,\n    \"client\": \"192.168.107.1:33122\",\n    \"clientMetadata\": {\n        \"driver\": {\n            \"name\": \"mongo-go-driver\",\n            \"version\": \"2.4.0\"\n        },\n        \"os\": {\n            \"architecture\": \"arm64\",\n            \"type\": \"darwin\"\n        },\n        \"platform\": \"go1.25.7\"\n    },\n    \"command\": {\n        \"$db\": \"airline\",\n        \"batchSize\": 1,\n        \"filter\": {\n            \"flight_id\": 880\n        },\n        \"find\": \"flights\",\n        \"limit\": {\n            \"high\": 0,\n            \"low\": 5,\n            \"unsigned\": false\n        },\n        \"lsid\": {\n            \"id\": {}\n        },\n        \"projection\": {\n            \"_id\": 0,\n            \"destination\": 1,\n            \"flight_id\": 1,\n            \"gate\": 1,\n            \"origin\": 1\n        }\n    },\n    \"connectionId\": 14544,\n    \"currentOpTime\": \"2026-02-11T19:34:56.677+00:00\",\n    \"currentQueue\": null,\n    \"desc\": \"conn14544\",\n    \"effectiveUsers\": [\n        {\n            \"db\": \"admin\",\n            \"user\": \"root\"\n        }\n    ],\n    \"flowControlStats\": {},\n    \"host\": \"c4486b1ebd30:27017\",\n    \"isFromUserConnection\": true,\n    \"lockStats\": {\n        \"Global\": {\n            \"acquireCount\": {\n                \"r\": {\n                    \"high\": 0,\n                    \"low\": 1,\n                    \"unsigned\": false\n                }\n            }\n        }\n    },\n    \"locks\": {\n        \"Global\": \"r\"\n    },\n    \"lsid\": {\n        \"id\": {},\n        \"uid\": \"Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg=\"\n    },\n    \"microsecs_running\": {\n        \"high\": 0,\n        \"low\": 60,\n        \"unsigned\": false\n    },\n    \"ns\": \"airline.flights\",\n    \"numYields\": 0,\n    \"op\": \"query\",\n    \"opid\": -2024364589,\n    \"queues\": {\n        \"execution\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"high\": 0,\n                \"low\": 0,\n                \"unsigned\": false\n            }\n        },\n        \"ingress\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"high\": 0,\n                \"low\": 0,\n                \"unsigned\": false\n            }\n        }\n    },\n    \"redacted\": false,\n    \"secs_running\": {\n        \"high\": 0,\n        \"low\": 0,\n        \"unsigned\": false\n    },\n    \"threaded\": true,\n    \"type\": \"op\",\n    \"waitingForFlowControl\": false,\n    \"waitingForLock\": false\n}",
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
				QueryText:     "db.runCommand({\n    \"$db\": \"airline\",\n    \"insert\": \"flights\",\n    \"lsid\": {\n        \"id\": {}\n    },\n    \"ordered\": true,\n    \"writeConcern\": {\n        \"j\": true,\n        \"w\": 2,\n        \"wtimeout\": 5000\n    }\n})",
				QueryRawJson:  "{\n    \"active\": true,\n    \"client\": \"192.168.107.1:54514\",\n    \"clientMetadata\": {\n        \"driver\": {\n            \"name\": \"mongo-go-driver\",\n            \"version\": \"2.4.0\"\n        },\n        \"os\": {\n            \"architecture\": \"arm64\",\n            \"type\": \"darwin\"\n        },\n        \"platform\": \"go1.25.7\"\n    },\n    \"command\": {\n        \"$db\": \"airline\",\n        \"insert\": \"flights\",\n        \"lsid\": {\n            \"id\": {}\n        },\n        \"ordered\": true,\n        \"writeConcern\": {\n            \"j\": true,\n            \"w\": 2,\n            \"wtimeout\": 5000\n        }\n    },\n    \"connectionId\": 14823,\n    \"currentOpTime\": \"2026-02-12T09:44:46.880+00:00\",\n    \"currentQueue\": null,\n    \"desc\": \"conn14823\",\n    \"effectiveUsers\": [\n        {\n            \"db\": \"admin\",\n            \"user\": \"root\"\n        }\n    ],\n    \"flowControlStats\": {},\n    \"host\": \"c4486b1ebd30:27017\",\n    \"isFromUserConnection\": true,\n    \"lockStats\": {},\n    \"locks\": {},\n    \"lsid\": {\n        \"id\": {},\n        \"uid\": \"Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg=\"\n    },\n    \"ns\": \"airline.$cmd\",\n    \"numYields\": 0,\n    \"op\": \"command\",\n    \"opid\": 1991397463,\n    \"queues\": {\n        \"execution\": {\n            \"admissions\": 0,\n            \"totalTimeQueuedMicros\": {\n                \"high\": 0,\n                \"low\": 0,\n                \"unsigned\": false\n            }\n        },\n        \"ingress\": {\n            \"admissions\": 0,\n            \"totalTimeQueuedMicros\": {\n                \"high\": 0,\n                \"low\": 0,\n                \"unsigned\": false\n            }\n        }\n    },\n    \"redacted\": false,\n    \"threaded\": true,\n    \"type\": \"op\",\n    \"waitingForFlowControl\": false,\n    \"waitingForLock\": false\n}",
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
				QueryRawJson:  "{\n    \"active\": true,\n    \"client\": \"192.168.107.1:46310\",\n    \"clientMetadata\": {\n        \"driver\": {\n            \"name\": \"mongo-go-driver\",\n            \"version\": \"2.4.0\"\n        },\n        \"os\": {\n            \"architecture\": \"arm64\",\n            \"type\": \"darwin\"\n        },\n        \"platform\": \"go1.25.7\"\n    },\n    \"command\": {\n        \"multi\": false,\n        \"q\": {\n            \"_id\": \"val-669\"\n        },\n        \"u\": {\n            \"$set\": {\n                \"duration_minutes\": 216\n            }\n        },\n        \"upsert\": false\n    },\n    \"connectionId\": 14858,\n    \"currentOpTime\": \"2026-02-12T10:45:40.456+00:00\",\n    \"currentQueue\": null,\n    \"desc\": \"conn14858\",\n    \"effectiveUsers\": [\n        {\n            \"db\": \"admin\",\n            \"user\": \"root\"\n        }\n    ],\n    \"flowControlStats\": {\n        \"acquireCount\": {\n            \"high\": 0,\n            \"low\": 1,\n            \"unsigned\": false\n        }\n    },\n    \"host\": \"c4486b1ebd30:27017\",\n    \"isFromUserConnection\": true,\n    \"lockStats\": {\n        \"Collection\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"high\": 0,\n                    \"low\": 1,\n                    \"unsigned\": false\n                }\n            }\n        },\n        \"Database\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"high\": 0,\n                    \"low\": 1,\n                    \"unsigned\": false\n                }\n            }\n        },\n        \"Global\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"high\": 0,\n                    \"low\": 1,\n                    \"unsigned\": false\n                }\n            }\n        },\n        \"ReplicationStateTransition\": {\n            \"acquireCount\": {\n                \"w\": {\n                    \"high\": 0,\n                    \"low\": 1,\n                    \"unsigned\": false\n                }\n            }\n        }\n    },\n    \"locks\": {},\n    \"lsid\": {\n        \"id\": {},\n        \"uid\": \"Y5mrDaxi8gv8RmdTsQ+1j7fmkr7JUsabhNmXAheU0fg=\"\n    },\n    \"microsecs_running\": {\n        \"high\": 0,\n        \"low\": 119,\n        \"unsigned\": false\n    },\n    \"ns\": \"airline.flights\",\n    \"numYields\": 0,\n    \"op\": \"update\",\n    \"opid\": -941663415,\n    \"planSummary\": \"EXPRESS_IXSCAN { _id: 1 },EXPRESS_UPDATE\",\n    \"queues\": {\n        \"execution\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"high\": 0,\n                \"low\": 0,\n                \"unsigned\": false\n            }\n        },\n        \"ingress\": {\n            \"admissions\": 1,\n            \"totalTimeQueuedMicros\": {\n                \"high\": 0,\n                \"low\": 0,\n                \"unsigned\": false\n            }\n        }\n    },\n    \"redacted\": false,\n    \"secs_running\": {\n        \"high\": 0,\n        \"low\": 0,\n        \"unsigned\": false\n    },\n    \"threaded\": true,\n    \"type\": \"op\",\n    \"waitingForFlowControl\": false,\n    \"waitingForLock\": false\n}",
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
