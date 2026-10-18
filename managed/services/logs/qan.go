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
	"database/sql"
	"regexp"
	"strings"
	"time"

	logsv1 "github.com/percona/pmm/api/logs/v1"
)

// queryWindow is how far from a line's time QAN is searched for its query.
const queryWindow = 30 * time.Minute

var (
	// A PostgreSQL statement or execute line, after an optional duration.
	statementRe = regexp.MustCompile(`^(?:duration: [0-9.]+ ms\s+)?(?:statement|execute [^:\n]*):\s*`)
	// Lines PostgreSQL appends to an entry, such as the parameters of an execute.
	appendedRe = regexp.MustCompile(`\n(?:DETAIL|HINT|CONTEXT|QUERY|LOCATION|STATEMENT):`)

	sqlCommentRe = regexp.MustCompile(`(?s)/\*.*?\*/|--[^\n]*`)
	sqlStringRe  = regexp.MustCompile(`'(?:[^']|'')*'`)
	sqlParamRe   = regexp.MustCompile(`\$[0-9]+`)
	sqlNumberRe  = regexp.MustCompile(`\b[0-9]+(?:\.[0-9]+)?\b`)
	sqlSpaceRe   = regexp.MustCompile(`\s+`)
	sqlPunctRe   = regexp.MustCompile(`\s*([(),;=<>+\-*/])\s*`)
	sqlListRe    = regexp.MustCompile(`\?(?:,\?)+`)
)

// statementOf returns the SQL a PostgreSQL log line logged, or "" when it logged none.
func statementOf(body string) string {
	loc := statementRe.FindStringIndex(body)
	if loc == nil {
		return ""
	}
	stmt := body[loc[1]:]
	if end := appendedRe.FindStringIndex(stmt); end != nil {
		stmt = stmt[:end[0]]
	}
	stmt = strings.TrimSpace(stmt)
	if strings.Trim(stmt, "; ") == "" {
		return ""
	}
	return stmt
}

// normalizeSQL folds a logged statement and its QAN fingerprint to the same text: literals and placeholders become ?,
// lists of them one ?, and comments, case and spacing go.
func normalizeSQL(s string) string {
	s = sqlCommentRe.ReplaceAllString(s, " ")
	s = sqlStringRe.ReplaceAllString(s, "?")
	s = sqlParamRe.ReplaceAllString(s, "?")
	s = sqlNumberRe.ReplaceAllString(s, "?")
	s = sqlSpaceRe.ReplaceAllString(strings.ToLower(s), " ")
	s = sqlPunctRe.ReplaceAllString(s, "$1")
	s = sqlListRe.ReplaceAllString(s, "?")
	return strings.TrimRight(strings.TrimSpace(s), ";")
}

// findQueryQuery returns the PostgreSQL QAN queries of a node around a time, most executed first.
func findQueryQuery(nodeName string, at time.Time) (string, []any) {
	return "SELECT queryid, service_name, service_id, any(fingerprint), sum(num_queries) AS num " +
			"FROM pmm.metrics WHERE node_name = ? AND service_type = 'postgresql' AND period_start BETWEEN toDateTime(?) AND toDateTime(?) " +
			"GROUP BY queryid, service_name, service_id ORDER BY num DESC LIMIT 5000" + querySettings,
		[]any{nodeName, at.Add(-queryWindow).Unix(), at.Add(queryWindow).Unix()}
}

// FindLogQuery finds the QAN query a PostgreSQL log line ran.
func (s *Service) FindLogQuery(ctx context.Context, req *logsv1.FindLogQueryRequest) (*logsv1.FindLogQueryResponse, error) {
	err := checkLBAC(ctx)
	if err != nil {
		return nil, err
	}
	res := &logsv1.FindLogQueryResponse{}
	stmt := statementOf(req.Body)
	if stmt == "" {
		return res, nil
	}
	want := normalizeSQL(stmt)
	q, args := findQueryQuery(req.NodeName, req.Time.AsTime())
	err = s.query(ctx, q, args, func(rows *sql.Rows) error {
		var (
			queryID, serviceName, serviceID, fingerprint string
			num                                          float64
		)
		err := rows.Scan(&queryID, &serviceName, &serviceID, &fingerprint, &num)
		if err != nil {
			return err
		}
		// Rows come most executed first, so the first match is the busiest service of the busiest query.
		if res.QueryId == "" && normalizeSQL(fingerprint) == want {
			res.QueryId, res.ServiceName, res.ServiceId, res.Fingerprint = queryID, serviceName, serviceID, fingerprint
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}
