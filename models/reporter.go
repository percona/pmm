// qan-api
// Copyright (C) 2019 Percona LLC
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

package models

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/jmoiron/sqlx"
)

// Reporter implements models to select metrics bucket by params.
type Reporter struct {
	db *sqlx.DB
}

// NewReporter initialize Reporter with db instance.
func NewReporter(db *sqlx.DB) Reporter {
	return Reporter{db: db}
}

var funcMap = template.FuncMap{
	"inc":         func(i int) int { return i + 1 },
	"StringsJoin": strings.Join,
}

const queryReportTmpl = `
SELECT
{{ index . "group" }} AS dimension,
rowNumberInAllBlocks() AS row_number,
{{ if eq (index . "group") "queryid" }} any(fingerprint) AS fingerprint, {{ end }}
SUM(num_queries) AS num_queries,

SUM(m_query_time_cnt) AS m_query_time_cnt,
SUM(m_query_time_sum) AS m_query_time_sum,
MIN(m_query_time_min) AS m_query_time_min,
MAX(m_query_time_max) AS m_query_time_max,
AVG(m_query_time_p99) AS m_query_time_p99

FROM queries
WHERE period_start > :from AND period_start < :to
{{ if index . "first_seen" }} AND first_seen >= :from {{ end }}
{{ if index . "keyword" }} AND (queryid = :keyword OR fingerprint LIKE :start_keyword ) {{ end }}
{{ if index . "queryids" }} AND queryid IN ( :queryids ) {{ end }}
{{ if index . "servers" }} AND d_server IN ( :servers ) {{ end }}
{{ if index . "databases" }} AND d_database IN ( :databases ) {{ end }}
{{ if index . "schemas" }} AND d_schema IN ( :schemas ) {{ end }}
{{ if index . "users" }} AND d_username IN ( :users ) {{ end }}
{{ if index . "hosts" }} AND d_client_host IN ( :hosts ) {{ end }}
{{ if index . "labels" }}
	AND (
		{{$i := 0}}
		{{range $key, $val := index . "labels"}}
			{{ $i = inc $i}} {{ if gt $i 1}} OR {{ end }}
			has(['{{ StringsJoin $val "','" }}'], labels.value[indexOf(labels.key, '{{ $key }}')])
		{{ end }}
	)
{{ end }}
GROUP BY {{ index . "group" }}
	WITH TOTALS
ORDER BY {{ index . "order" }}
LIMIT :offset, :limit
`

type DimensionReport struct {
	Dimension     string  `db:"dimension"`
	RowNumber     float32 `db:"row_number"`
	Fingerprint   string  `db:"fingerprint"`
	NumQueries    float32 `db:"num_queries"`
	MQueryTimeCtn float32 `db:"m_query_time_cnt"`
	MQueryTimeSum float32 `db:"m_query_time_sum"`
	MQueryTimeMin float32 `db:"m_query_time_min"`
	MQueryTimeMax float32 `db:"m_query_time_max"`
	MQueryTimeP99 float32 `db:"m_query_time_p99"`
}

func (r *Reporter) Select(period_start_from, period_start_to, keyword string, firstSeen bool, dQueryids, dServers, dDatabases, dSchemas, dUsernames, dClientHosts []string, dbLabels map[string][]string, group, order string, offset uint32, limit uint32) ([]DimensionReport, error) {

	if group == "" {
		group = "queryid"
	}
	if order == "" {
		order = "m_query_time_sum"
	}

	if limit == 0 {
		limit = 10
	}
	arg := map[string]interface{}{
		"from":          period_start_from,
		"to":            period_start_to,
		"keyword":       keyword,
		"start_keyword": "%" + keyword,
		"first_seen":    firstSeen,
		"queryids":      dQueryids,
		"servers":       dServers,
		"databases":     dDatabases,
		"schemas":       dSchemas,
		"users":         dUsernames,
		"hosts":         dClientHosts,
		"labels":        dbLabels,
		"group":         group,
		"order":         order,
		"offset":        offset,
		"limit":         limit,
	}
	var queryBuffer bytes.Buffer
	if tmpl, err := template.New("queryReport").Funcs(funcMap).Parse(queryReportTmpl); err != nil {
		log.Fatalln(err)
	} else if err = tmpl.Execute(&queryBuffer, arg); err != nil {
		log.Fatalln(err)
	}
	query, args, err := sqlx.Named(queryBuffer.String(), arg)
	query, args, err = sqlx.In(query, args...)
	query = r.db.Rebind(query)

	results := []DimensionReport{}
	rows, err := r.db.Queryx(query, args...)
	for rows.Next() {
		result := DimensionReport{}
		err := rows.StructScan(&result)
		if err != nil {
			fmt.Printf("DimensionReport Scan error: %v", err)
		}
		results = append(results, result)
	}
	//  Get totals.
	rows.NextResultSet()
	for rows.Next() {
		result := DimensionReport{}
		err := rows.StructScan(&result)
		if err != nil {
			fmt.Printf("DimensionReport Scan TOTALS error: %v", err)
		}
		result.Dimension = "TOTALS"
		results = append([]DimensionReport{result}, results...)
	}
	return results, err
}
