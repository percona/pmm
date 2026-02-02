package realtimeanalytics

import (
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

var (
	staticQueries_1 []*rtav1.QueryData
	staticQueries_2 []*rtav1.QueryData
)

func init() {
	staticQueries_1 = []*rtav1.QueryData{
		{
			ServiceId:         "683fda11-a9e1-44b4-9da5-d2918e19f8f9",
			ServiceName:       "psmdb-1",
			QueryId:           "static-query-1",
			QueryText:         `{ find: "mycollection", filter: { status: "active" } }`,
			State:             "RUNNING",
			ExecutionDuration: durationpb.New(15),
			RowsExamined:      200,
			RowsSent:          100,
			CollectTime:       timestamppb.Now(),
			RawQueryJson:      `{ find: "mycollection", filter: { status: "active" } }`,
			Payload: &rtav1.QueryData_MongoDbPayload{
				&rtav1.QueryMongoDBData{
					Opid: "1",
					// SecsRunning:    15,
					Client:         "127.0.0.1:5060",
					WaitingForLock: false,
					IndexUtilized:  "COLLSCAN",
				},
			},
		},
		{
			ServiceId:         "683fda11-a9e1-44b4-9da5-d2918e19f8f9",
			ServiceName:       "psmdb-1",
			QueryId:           "static-query-2",
			QueryText:         `{ find: "mycollection", filter: { status: "active" } }`,
			State:             "PROCESSING",
			ExecutionDuration: durationpb.New(25),
			RowsExamined:      200,
			RowsSent:          100,
			CollectTime:       timestamppb.Now(),
			RawQueryJson:      `{ find: "mycollection", filter: { status: "active" } }`,
			Payload: &rtav1.QueryData_MongoDbPayload{
				&rtav1.QueryMongoDBData{
					Opid: "2",
					// SecsRunning:    15,
					Client:         "127.0.0.1:5061",
					WaitingForLock: true,
					IndexUtilized:  "IXSCAN",
				},
			},
		},
		{
			ServiceId:         "683fda11-a9e1-44b4-9da5-d2918e19f8f9",
			ServiceName:       "psmdb-1",
			QueryId:           "static-query-3",
			QueryText:         `{ find: "mycollection", filter: { status: "active" } }`,
			State:             "FINISHED",
			ExecutionDuration: durationpb.New(35),
			RowsExamined:      200,
			RowsSent:          100,
			CollectTime:       timestamppb.Now(),
			RawQueryJson:      `{ find: "mycollection", filter: { status: "active" } }`,
			Payload: &rtav1.QueryData_MongoDbPayload{
				&rtav1.QueryMongoDBData{
					Opid: "1",
					// SecsRunning:    15,
					Client:         "127.0.0.1:5062",
					WaitingForLock: true,
					IndexUtilized:  "COLLSCAN",
				},
			},
		},
	}
}
