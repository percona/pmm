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

package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

// ParseCurrentOp parses raw bson document returned by currentOp command into *QueryData.
func ParseCurrentOp(raw bson.Raw) (qData *rtav1.QueryData) {
	qData = &rtav1.QueryData{
		Payload: &rtav1.QueryData_MongoDbPayload{
			MongoDbPayload: &rtav1.QueryMongoDBData{},
		},
	}

	// MongoDB driver parser may panic.
	// We need to recover from panic and return empty QueryData in this case
	// to avoid crashing the whole agent.
	defer func() {
		if r := recover(); r != nil {
			qData = nil
		}
	}()

	parseGenericFields(raw, qData)
	parseMongoFields(raw, qData)

	return qData
}

// parseGenericFields parses common fields from raw bson document returned by currentOp command into *QueryData.
func parseGenericFields(raw bson.Raw, q *rtav1.QueryData) {
	if opid, ok := raw.Lookup("opid").Int32OK(); ok {
		q.QueryId = fmt.Sprintf("%v", opid)
	}

	if msRunning, ok := raw.Lookup("microsecs_running").Int64OK(); ok {
		q.QueryExecutionDuration = durationpb.New(time.Duration(1000 * msRunning))
	}
	q.ClientAddress, _ = raw.Lookup("client").StringValueOK()
	var m any
	if err := bson.Unmarshal(raw, &m); err == nil {
		if jsonValue, err := json.MarshalIndent(m, "", "    "); err == nil {
			q.QueryRawJson = string(jsonValue)
		}
	}
}

// parseMongoFields parses MongoDB-specific fields from raw bson document returned by currentOp command into *QueryData.
func parseMongoFields(raw bson.Raw, q *rtav1.QueryData) {
	var p *rtav1.QueryData_MongoDbPayload
	var ok bool
	if q.Payload == nil {
		p = &rtav1.QueryData_MongoDbPayload{
			MongoDbPayload: &rtav1.QueryMongoDBData{},
		}
		q.Payload = p
	} else if p, ok = q.Payload.(*rtav1.QueryData_MongoDbPayload); !ok {
		// If Payload is already set but not of type MongoDbPayload, we should not overwrite it.
		return
	}

	// MongoDB specific fields
	p.MongoDbPayload.DbInstanceAddress, _ = raw.Lookup("host").StringValueOK()
	p.MongoDbPayload.ClientAppName, _ = raw.Lookup("appName").StringValueOK()
	p.MongoDbPayload.PlanSummary, _ = raw.Lookup("planSummary").StringValueOK()
	p.MongoDbPayload.Operation, _ = raw.Lookup("op").StringValueOK()
	if opTimeStr, ok := raw.Lookup("currentOpTime").StringValueOK(); ok {
		if opTime, err := time.Parse(time.RFC3339, opTimeStr); err == nil {
			p.MongoDbPayload.OperationStartTime = timestamppb.New(opTime)
		}
	}
	// parse username from effectiveUsers array
	if effectiveUsers, ok := raw.Lookup("effectiveUsers").ArrayOK(); ok {
		if eud, ok := effectiveUsers.Index(0).DocumentOK(); ok {
			p.MongoDbPayload.Username, _ = eud.Lookup("user").StringValueOK()
		}
	}

	// parse database name and collection name from ns field
	var ns string
	if ns, ok = raw.Lookup("ns").StringValueOK(); ok {
		// ns has in format "database.collection", we need to split it to get database name
		parts := strings.SplitN(ns, ".", -1)
		p.MongoDbPayload.DatabaseName = parts[0]
		if len(parts) > 1 {
			col := parts[1]
			if len(parts) == 2 && col == "$cmd" {
				// reset collection name because $cmd is not a real collection, it's just a namespace for commands
				col = ""
			}
			p.MongoDbPayload.Collection = col
		}
	}

	// parse command field to get query text
	commandRaw, _ := raw.Lookup("command").DocumentOK()
	switch p.MongoDbPayload.Operation {
	case "query":
		q.QueryText = parseCommandFind(commandRaw)
	case "insert":
		q.QueryText = parseCommandInsert(commandRaw)
	case "update":
		q.QueryText = parseCommandUpdate(commandRaw, p.MongoDbPayload.Collection)
	case "remove", "delete":
		q.QueryText = parseCommandDelete(commandRaw, p.MongoDbPayload.Collection)
	default:
		q.QueryText = parseCommand(raw)
	}
}
