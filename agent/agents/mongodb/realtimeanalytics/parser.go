package realtimeanalytics

import (
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

// parseCurrentOp parses raw bson document returned by currentOp command into *QueryData.
func parseCurrentOp(raw bson.Raw) *rtav1.QueryData {
	q := &rtav1.QueryData{
		Payload: &rtav1.QueryData_MongoDbPayload{
			MongoDbPayload: &rtav1.QueryMongoDBData{},
		},
	}

	// TODO: errgroup.Go
	parseGenericFields(raw, q)
	parseMongoFields(raw, q)

	return q
}

// parseGenericFields parses common fields from raw bson document returned by currentOp command into *QueryData.
func parseGenericFields(raw bson.Raw, q *rtav1.QueryData) {
	// TODO: recover from panic
	if opid, ok := raw.Lookup("opid").Int32OK(); ok {
		q.QueryId = fmt.Sprintf("%v", opid)
	}

	if msRunning, ok := raw.Lookup("microsecs_running").Int64OK(); ok {
		q.QueryExecutionDuration = durationpb.New(time.Duration(1000 * msRunning))
	}
	q.ClientAddress, _ = raw.Lookup("client").StringValueOK()
	q.QueryRawJson = raw.String()
}

// parseMongoFields parses MongoDB-specific fields from raw bson document returned by currentOp command into *QueryData.
func parseMongoFields(raw bson.Raw, q *rtav1.QueryData) {
	// TODO: recover from panic
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
	if ns, ok := raw.Lookup("ns").StringValueOK(); ok {
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
}