// qan-api2
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
	"github.com/percona/pmm/api/qanpb"
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

{{ if eq (index . "group") "queryid" }} any(fingerprint) AS fingerprint, {{ end }}
SUM(num_queries) AS num_queries,

SUM(m_query_time_cnt) AS m_query_time_cnt,
SUM(m_query_time_sum) AS m_query_time_sum,
MIN(m_query_time_min) AS m_query_time_min,
MAX(m_query_time_max) AS m_query_time_max,
AVG(m_query_time_p99) AS m_query_time_p99,

{{range $j, $col := index . "columns"}}
	SUM(m_{{ $col }}_cnt) AS m_{{ $col }}_cnt,
	SUM(m_{{ $col }}_sum) AS m_{{ $col }}_sum,
	MIN(m_{{ $col }}_min) AS m_{{ $col }}_min,
	MAX(m_{{ $col }}_max) AS m_{{ $col }}_max,
	AVG(m_{{ $col }}_p99) AS m_{{ $col }}_p99,
{{ end }}

rowNumberInAllBlocks() AS total_rows

FROM metrics
WHERE period_start > :period_start_from AND period_start < :period_start_to
{{ if index . "first_seen" }} AND first_seen >= :period_start_from {{ end }}
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

// M is map for interfaces.
type M map[string]interface{}

// Select select metrics for report.
func (r *Reporter) Select(periodStartFrom, periodStartTo, keyword string,
	firstSeen bool, dQueryids, dServers, dDatabases, dSchemas, dUsernames,
	dClientHosts []string, dbLabels map[string][]string, group, order string,
	offset uint32, limit uint32, columns []string) ([]M, error) {

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
		"period_start_from": periodStartFrom,
		"period_start_to":   periodStartTo,
		"keyword":           keyword,
		"start_keyword":     "%" + keyword,
		"first_seen":        firstSeen,
		"queryids":          dQueryids,
		"servers":           dServers,
		"databases":         dDatabases,
		"schemas":           dSchemas,
		"users":             dUsernames,
		"hosts":             dClientHosts,
		"labels":            dbLabels,
		"group":             group,
		"order":             order,
		"offset":            offset,
		"limit":             limit,
		"columns":           columns,
	}
	var queryBuffer bytes.Buffer
	if tmpl, err := template.New("queryReport").Funcs(funcMap).Parse(queryReportTmpl); err != nil {
		log.Fatalln(err)
	} else if err = tmpl.Execute(&queryBuffer, arg); err != nil {
		log.Fatalln(err)
	}
	var results []M
	query, args, err := sqlx.Named(queryBuffer.String(), arg)
	if err != nil {
		return results, fmt.Errorf("prepare named:%v", err)
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		return results, fmt.Errorf("populate agruments in IN clause:%v", err)
	}
	query = r.db.Rebind(query)

	rows, err := r.db.Queryx(query, args...)
	fmt.Printf("queryx error: %v", err)
	for rows.Next() {
		result := make(M)
		err = rows.MapScan(result)
		if err != nil {
			fmt.Printf("DimensionReport Scan error: %v", err)
		}
		results = append(results, result)
	}
	rows.NextResultSet()
	total := make(M)
	for rows.Next() {
		err = rows.MapScan(total)
		if err != nil {
			fmt.Printf("DimensionReport Scan TOTALS error: %v", err)
		}
		results = append([]M{total}, results...)
	}
	return results, err
}

const queryReportSparklinesTmpl = `
SELECT
(toUnixTimestamp( :period_start_to ) - toUnixTimestamp( :period_start_from )) / 60 AS time_frame,
intDivOrZero(toUnixTimestamp( :period_start_to ) - toRelativeSecondNum(period_start), time_frame) AS point,
toUnixTimestamp( :period_start_to ) - (point * time_frame) AS timestamp,
SUM(num_queries) AS num_queries_sum,
SUM(m_query_time_sum) AS m_query_time_sum,
m_query_time_sum / time_frame AS m_query_load,
{{range $j, $col := index . "columns"}}
	SUM(m_{{ $col }}_sum) AS m_{{ $col }}_sum,
{{ end }}
m_query_time_sum / num_queries_sum AS m_query_time_avg
FROM metrics
WHERE period_start > :period_start_from AND period_start < :period_start_to
{{ if index . "dimension_val" }} AND {{ index . "group" }} = '{{ index . "dimension_val" }}' {{ end }}
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
GROUP BY point
ORDER BY point ASC;
`

// SelectSparklines selects datapoint for sparklines.
func (r *Reporter) SelectSparklines(dimensionVal, periodStartFrom, periodStartTo,
	keyword string, firstSeen bool, dQueryids, dServers, dDatabases, dSchemas,
	dUsernames, dClientHosts []string, dbLabels map[string][]string, group string,
	columns []string) ([]*qanpb.Point, error) {
	if group == "" {
		group = "queryid"
	}

	arg := map[string]interface{}{
		"dimension_val":     dimensionVal,
		"period_start_from": periodStartFrom,
		"period_start_to":   periodStartTo,
		"keyword":           keyword,
		"start_keyword":     "%" + keyword,
		"first_seen":        firstSeen,
		"queryids":          dQueryids,
		"servers":           dServers,
		"databases":         dDatabases,
		"schemas":           dSchemas,
		"users":             dUsernames,
		"hosts":             dClientHosts,
		"labels":            dbLabels,
		"group":             group,
		"columns":           columns,
	}
	var results []*qanpb.Point
	var queryBuffer bytes.Buffer
	if tmpl, err := template.New("queryReportSparklines").Funcs(funcMap).Parse(queryReportSparklinesTmpl); err != nil {
		log.Fatalln(err)
	} else if err = tmpl.Execute(&queryBuffer, arg); err != nil {
		log.Fatalln(err)
	}
	query, args, err := sqlx.Named(queryBuffer.String(), arg)
	if err != nil {
		return results, fmt.Errorf("prepare named:%v", err)
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		return results, fmt.Errorf("populate agruments in IN clause:%v", err)
	}
	query = r.db.Rebind(query)

	rows, err := r.db.Queryx(query, args...)
	if err != nil {
		return results, fmt.Errorf("report query:%v", err)
	}
	for rows.Next() {
		res := make(map[string]interface{})
		err = rows.MapScan(res)
		if err != nil {
			fmt.Printf("DimensionReport Scan error: %v", err)
		}
		points := qanpb.Point{
			Values: make(map[string]float32),
		}
		for k, v := range res {
			points.Values[k] = interfaceToFloat32(v)
		}
		results = append(results, &points)
	}
	return results, err
}

func interfaceToFloat32(unk interface{}) float32 {
	switch i := unk.(type) {
	case float64:
		return float32(i)
	case float32:
		return i
	case int64:
		return float32(i)
	default:
		return float32(0)
	}
}
