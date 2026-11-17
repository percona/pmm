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
	"context"
	"crypto/md5" //nolint:gosec // the query ID pmm-agent computes
	"database/sql"
	"fmt"
	"strings"
	"time"

	pfingerprinter "github.com/percona/percona-toolkit/src/go/mongolib/fingerprinter"
	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/percona/pmm/agent/agents/mongodb/shared/fingerprinter"
	logsv1 "github.com/percona/pmm/api/logs/v1"
)

// mongoSlowQuery is what the profiler's fingerprint needs from the attr of a mongod "Slow query" entry.
type mongoSlowQuery struct {
	Type    string `bson:"type"`
	Ns      string `bson:"ns"`
	Command bson.D `bson:"command"`
}

// field returns the value of key in d, or nil.
func field(d bson.D, key string) any {
	for _, e := range d {
		if e.Key == key {
			return e.Value
		}
	}
	return nil
}

// mongoProfile returns the system.profile document the profiler records for the operation of a "Slow query" entry's
// attr. The log names a find or getMore a command, and logs update and delete commands with their statements, which
// the profiler records one by one; the first statement stands for them.
func mongoProfile(attr string) (proto.SystemProfile, error) {
	var q mongoSlowQuery
	err := bson.UnmarshalExtJSON([]byte(attr), false, &q)
	if err != nil {
		return proto.SystemProfile{}, fmt.Errorf("failed to parse attr: %w", err)
	}
	doc := proto.SystemProfile{Op: q.Type, Ns: q.Ns, Command: q.Command}
	if q.Type != "command" {
		return doc, nil
	}
	db, _, _ := strings.Cut(q.Ns, ".")
	statement := func(op, collection, list string) {
		statements, _ := field(q.Command, list).(bson.A)
		if len(statements) == 0 {
			return
		}
		first, ok := statements[0].(bson.D)
		if !ok {
			return
		}
		doc.Op, doc.Ns, doc.Command = op, db+"."+fmt.Sprint(field(q.Command, collection)), first
	}
	// The collector sorts attr's keys, so the command is told by its fields, not by its first key.
	switch {
	case field(q.Command, "find") != nil:
		doc.Op = "query"
	case field(q.Command, "getMore") != nil:
		doc.Op = "getmore"
	case field(q.Command, "insert") != nil:
		doc.Op = "insert"
	case field(q.Command, "update") != nil && field(q.Command, "updates") != nil:
		statement("update", "update", "updates")
	case field(q.Command, "delete") != nil && field(q.Command, "deletes") != nil:
		statement("remove", "delete", "deletes")
	}
	return doc, nil
}

// mongoQueryID returns the QAN query ID of a "Slow query" entry's attr as pmm-agent's profiler computes it: the MD5 of
// the op, namespace and fingerprint (percona-toolkit mongolib/stats). It is "" when the entry logged no command.
func mongoQueryID(attr string) (string, error) {
	doc, err := mongoProfile(attr)
	if err != nil || len(doc.Command) == 0 {
		return "", err
	}
	var fp pfingerprinter.Fingerprint
	func() {
		// The fingerprinter asserts the types of what it reads, which comes from a log line here.
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("failed to fingerprint: %v", r)
			}
		}()
		fp, err = fingerprinter.NewFingerprinter(fingerprinter.DefaultKeyFilters()).Fingerprint(doc)
	}()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(doc.Op+doc.Ns+fp.Fingerprint))), nil //nolint:gosec // see the import
}

// mongoQueryQuery returns the busiest MongoDB service of a node that ran a QAN query around a time.
func mongoQueryQuery(queryID, nodeName string, at time.Time) (string, []any) {
	return "SELECT service_name, service_id, any(fingerprint), sum(num_queries) AS num FROM pmm.metrics " +
			"WHERE queryid = ? AND node_name = ? AND service_type = 'mongodb' AND period_start BETWEEN toDateTime(?) AND toDateTime(?) " +
			"GROUP BY service_name, service_id ORDER BY num DESC LIMIT 1" + querySettings,
		[]any{queryID, nodeName, at.Add(-queryWindow).Unix(), at.Add(queryWindow).Unix()}
}

// findMongoQuery finds the QAN query of a MongoDB "Slow query" line from its attr.
func (s *Service) findMongoQuery(ctx context.Context, req *logsv1.FindLogQueryRequest, attr string) (*logsv1.FindLogQueryResponse, error) {
	res := &logsv1.FindLogQueryResponse{}
	id, err := mongoQueryID(attr)
	if err != nil {
		s.l.WithError(err).Debug("No query in a MongoDB slow query line.")
		return res, nil
	}
	if id == "" {
		return res, nil
	}
	q, args := mongoQueryQuery(id, req.NodeName, req.Time.AsTime())
	err = s.query(ctx, q, args, func(rows *sql.Rows) error {
		var num float64
		res.QueryId = id
		return rows.Scan(&res.ServiceName, &res.ServiceId, &res.Fingerprint, &num)
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}
