// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package logs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/types/known/timestamppb"

	logsv1 "github.com/percona/pmm/api/logs/v1"
)

// Slow query attrs from adre-pmm7, cut to the fields the fingerprint reads, with the query IDs QAN stored for them.
var mongoSamples = []struct {
	name, attr, queryID string
}{{
	name:    "find",
	attr:    `{"type":"command","ns":"shop.orders","command":{"filter":{"order_no":54400},"find":"orders","limit":1}}`,
	queryID: "ceb614f661657dceef2a13b1b06fd745",
}, {
	name: "insert",
	attr: `{"type":"command","ns":"shop.audit_log","command":{"documents":[{"_id":{"$oid":"6ad5ebc04fceb9f92005cd6e"},` +
		`"at":{"$date":"2026-10-19T10:06:56.013Z"},"event":"viewed","order_no":46917}],"insert":"audit_log"}}`,
	queryID: "746ef8e8052622822899b50d46f54c78",
}, {
	name: "update",
	attr: `{"type":"update","ns":"shop.orders","command":{"multi":false,"q":{"order_no":912483},` +
		`"u":{"$set":{"status":"shipped"}},"upsert":false}}`,
	queryID: "ce427c305ab5d22184ff23fcb84e332a",
}, {
	// The update command logs its statements; the profiler records each statement as an update.
	name: "update command",
	attr: `{"type":"command","ns":"shop.$cmd","command":{"update":"orders","updates":[{"multi":false,` +
		`"q":{"order_no":47968},"u":{"$set":{"status":"shipped"}},"upsert":false}]}}`,
	queryID: "ce427c305ab5d22184ff23fcb84e332a",
}}

func TestMongoQueryID(t *testing.T) {
	for _, s := range mongoSamples {
		t.Run(s.name, func(t *testing.T) {
			id, err := mongoQueryID(s.attr)
			require.NoError(t, err)
			assert.Equal(t, s.queryID, id)
		})
	}

	id, err := mongoQueryID(`{"type":"command","ns":"admin.$cmd"}`)
	require.NoError(t, err)
	assert.Empty(t, id, "no command")
	_, err = mongoQueryID(`{"type":`)
	require.Error(t, err)
	// The fingerprinter panics on a pipeline stage that is not a document.
	_, err = mongoQueryID(`{"type":"command","ns":"shop.orders","command":{"aggregate":"orders","pipeline":[1]}}`)
	require.Error(t, err)
}

func TestMongoProfile(t *testing.T) {
	for attr, want := range map[string]struct{ op, ns string }{
		`{"type":"command","ns":"config.$cmd","command":{"delete":"image_collection","deletes":[{"limit":1,"q":{"_id":1}}]}}`: {
			"remove", "config.image_collection",
		},
		`{"type":"command","ns":"shop.system.profile","command":{"collection":"system.profile","getMore":5}}`: {
			"getmore", "shop.system.profile",
		},
		`{"type":"command","ns":"shop.orders","command":{"aggregate":"orders","pipeline":[]}}`: {"command", "shop.orders"},
		`{"type":"remove","ns":"shop.orders","command":{"limit":0,"q":{"status":"x"}}}`:        {"remove", "shop.orders"},
	} {
		doc, err := mongoProfile(attr)
		require.NoError(t, err, attr)
		assert.Equal(t, want.op, doc.Op, attr)
		assert.Equal(t, want.ns, doc.Ns, attr)
	}

	doc, err := mongoProfile(`{"type":"command","ns":"config.$cmd","command":{"delete":"c","deletes":[{"limit":1,"q":{"_id":1}}]}}`)
	require.NoError(t, err)
	assert.Equal(t, bson.D{{Key: "limit", Value: int32(1)}, {Key: "q", Value: bson.D{{Key: "_id", Value: int32(1)}}}}, doc.Command,
		"the statement, as the profiler records it")
}

func TestFindLogQuery_Mongo(t *testing.T) {
	s, mock := newTestService(t)
	at := testStart.Add(10 * time.Minute)
	sample := mongoSamples[0]
	q, args := mongoQueryQuery(sample.queryID, "rs102", at)
	assert.Equal(t, []any{sample.queryID, "rs102", at.Add(-queryWindow).Unix(), at.Add(queryWindow).Unix()}, args)
	mock.ExpectQuery(q).WillReturnRows(mock.NewRows([]string{"service_name", "service_id", "fingerprint", "num"}).
		AddRow("rs102_9419", "s2", "db.orders.find({\"order_no\":\"?\"}).limit(?)", float64(621)))
	req := &logsv1.FindLogQueryRequest{
		Time:       timestamppb.New(at),
		NodeName:   "rs102",
		Body:       "Slow query",
		Attributes: map[string]string{"attr": sample.attr},
	}
	res, err := s.FindLogQuery(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, &logsv1.FindLogQueryResponse{
		QueryId:     sample.queryID,
		ServiceName: "rs102_9419",
		ServiceId:   "s2",
		Fingerprint: "db.orders.find({\"order_no\":\"?\"}).limit(?)",
	}, res)

	// QAN has no such query on the node.
	mock.ExpectQuery(q).WillReturnRows(mock.NewRows([]string{"service_name", "service_id", "fingerprint", "num"}))
	res, err = s.FindLogQuery(t.Context(), req)
	require.NoError(t, err)
	assert.Empty(t, res.QueryId)

	// An attr that does not parse finds nothing and asks QAN nothing.
	req.Attributes["attr"] = `{"type":`
	res, err = s.FindLogQuery(t.Context(), req)
	require.NoError(t, err)
	assert.Empty(t, res.QueryId)
}
