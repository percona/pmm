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
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

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

// M is map for interfaces.
type M map[string]interface{}

const queryReportTmpl = `
	SELECT
	{{ index . "group" }} AS dimension,

	{{ if eq (index . "group") "queryid" }} any(fingerprint) {{ else }} '' {{ end }} AS fingerprint,
	SUM(num_queries) AS num_queries,

	SUM(m_query_time_cnt) AS m_query_time_cnt,
	SUM(m_query_time_sum) AS m_query_time_sum,
	MIN(m_query_time_min) AS m_query_time_min,
	MAX(m_query_time_max) AS m_query_time_max,
	AVG(m_query_time_p99) AS m_query_time_p99,

	{{range $j, $col := index . "common_columns"}}
		SUM(m_{{ $col }}_cnt) AS m_{{ $col }}_cnt,
		SUM(m_{{ $col }}_sum) AS m_{{ $col }}_sum,
		MIN(m_{{ $col }}_min) AS m_{{ $col }}_min,
		MAX(m_{{ $col }}_max) AS m_{{ $col }}_max,
		AVG(m_{{ $col }}_p99) AS m_{{ $col }}_p99,
	{{ end }}
	{{range $j, $col := index . "bool_columns"}}
		SUM(m_{{ $col }}_cnt) AS m_{{ $col }}_cnt,
		SUM(m_{{ $col }}_sum) AS m_{{ $col }}_sum,
	{{ end }}

	rowNumberInAllBlocks() AS total_rows

	FROM metrics
	WHERE period_start > :period_start_from AND period_start < :period_start_to
	{{ if index . "queryids" }} AND queryid IN ( :queryids ) {{ end }}
	{{ if index . "servers" }} AND server IN ( :servers ) {{ end }}
	{{ if index . "databases" }} AND database IN ( :databases ) {{ end }}
	{{ if index . "schemas" }} AND schema IN ( :schemas ) {{ end }}
	{{ if index . "users" }} AND username IN ( :users ) {{ end }}
	{{ if index . "hosts" }} AND client_host IN ( :hosts ) {{ end }}
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

//nolint
var tmplQueryReport = template.Must(template.New("queryReportTmpl").Funcs(funcMap).Parse(queryReportTmpl))

// Select select metrics for report.
func (r *Reporter) Select(ctx context.Context, periodStartFromSec, periodStartToSec int64,
	dQueryids, dServers, dDatabases, dSchemas, dUsernames, dClientHosts []string,
	dbLabels map[string][]string, group, order string, offset, limit uint32,
	commonColumns, boolColumns []string) ([]M, error) {

	arg := map[string]interface{}{
		"period_start_from": periodStartFromSec,
		"period_start_to":   periodStartToSec,
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
		"common_columns":    commonColumns,
		"bool_columns":      boolColumns,
	}

	var queryBuffer bytes.Buffer

	if err := tmplQueryReport.Execute(&queryBuffer, arg); err != nil {
		return nil, errors.Wrap(err, "cannot execute tmplQueryReport")
	}

	var results []M
	query, args, err := sqlx.Named(queryBuffer.String(), arg)
	if err != nil {
		return nil, errors.Wrap(err, "prepare named tmplQueryReport")
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "populate agruments in IN clause")
	}
	query = r.db.Rebind(query)

	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryxContext(queryCtx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "QueryxContext error")
	}
	for rows.Next() {
		result := make(M)
		err = rows.MapScan(result)
		if err != nil {
			return nil, errors.Wrap(err, "DimensionReport Scan error")
		}
		results = append(results, result)
	}
	rows.NextResultSet()
	total := make(M)
	for rows.Next() {
		err = rows.MapScan(total)
		if err != nil {
			return nil, errors.Wrap(err, "DimensionReport Scan TOTALS error")
		}
		results = append([]M{total}, results...)
	}
	return results, err
}

const queryReportSparklinesTmpl = `
	SELECT
		intDivOrZero(toUnixTimestamp( :period_start_to ) - toUnixTimestamp(period_start), {{ index . "time_frame" }}) AS point,
		toDateTime(toUnixTimestamp( :period_start_to ) - (point * {{ index . "time_frame" }})) AS timestamp,
		{{ index . "time_frame" }} AS time_frame,
		{{range $j, $col := index . "columns"}}
		if(SUM(m_{{ $col }}_cnt) == 0, NaN, SUM(m_{{ $col }}_sum) / time_frame) AS m_{{ $col }}_sum_per_sec,
		{{ end }}
		SUM(num_queries) / time_frame AS num_queries_per_sec
	FROM metrics
	WHERE period_start >= :period_start_from AND period_start <= :period_start_to
	{{ if index . "dimension_val" }} AND {{ index . "group" }} = '{{ index . "dimension_val" }}' {{ end }}
	{{ if index . "queryids" }} AND queryid IN ( :queryids ) {{ end }}
	{{ if index . "servers" }} AND server IN ( :servers ) {{ end }}
	{{ if index . "databases" }} AND database IN ( :databases ) {{ end }}
	{{ if index . "schemas" }} AND schema IN ( :schemas ) {{ end }}
	{{ if index . "users" }} AND username IN ( :users ) {{ end }}
	{{ if index . "hosts" }} AND client_host IN ( :hosts ) {{ end }}
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

//nolint
var tmplQueryReportSparklines = template.Must(template.New("queryReportSparklines").Funcs(funcMap).Parse(queryReportSparklinesTmpl))

// SelectSparklines selects datapoint for sparklines.
func (r *Reporter) SelectSparklines(ctx context.Context, dimensionVal string,
	periodStartFromSec, periodStartToSec int64,
	dQueryids, dServers, dDatabases, dSchemas, dUsernames, dClientHosts []string,
	dbLabels map[string][]string, group string, columns []string) ([]*qanpb.Point, error) {

	// Align to minutes
	periodStartToSec = periodStartToSec / 60 * 60
	periodStartFromSec = periodStartFromSec / 60 * 60

	// If time range is bigger then two hour - amount of sparklines points = 120 to avoid huge data in response.
	// Otherwise amount of sparklines points is equal to minutes in in time range to not mess up calculation.
	amountOfPoints := int64(optimalAmountOfPoint)
	timePeriod := periodStartToSec - periodStartFromSec
	// reduce amount of point if period less then 2h.
	if timePeriod < int64((minFullTimeFrame).Seconds()) {
		// minimum point is 1 minute
		amountOfPoints = timePeriod / 60
	}

	// how many full minutes we can fit into given amount of points.
	minutesInPoint := (periodStartToSec - periodStartFromSec) / 60 / amountOfPoints
	// we need aditional point to show this minutes
	remainder := ((periodStartToSec - periodStartFromSec) / 60) % amountOfPoints
	amountOfPoints += remainder / minutesInPoint
	timeFrame := minutesInPoint * 60

	arg := map[string]interface{}{
		"dimension_val":     dimensionVal,
		"period_start_from": periodStartFromSec,
		"period_start_to":   periodStartToSec,
		"queryids":          dQueryids,
		"servers":           dServers,
		"databases":         dDatabases,
		"schemas":           dSchemas,
		"users":             dUsernames,
		"hosts":             dClientHosts,
		"labels":            dbLabels,
		"group":             group,
		"columns":           columns,
		"time_frame":        timeFrame,
	}

	var results []*qanpb.Point
	var queryBuffer bytes.Buffer

	if err := tmplQueryReportSparklines.Execute(&queryBuffer, arg); err != nil {
		return nil, errors.Wrap(err, "cannot execute tmplQueryReportSparklines")
	}
	query, args, err := sqlx.Named(queryBuffer.String(), arg)
	if err != nil {
		return nil, errors.Wrap(err, "prepare named")
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "populate agruments in IN clause")
	}
	query = r.db.Rebind(query)

	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryxContext(queryCtx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "report query")
	}
	resultsWithGaps := map[uint32]*qanpb.Point{}

	sparklinePointFieldsToQuery := []string{
		"point",
		"timestamp",
		"time_frame",
	}
	for _, v := range columns {
		sparklinePointFieldsToQuery = append(sparklinePointFieldsToQuery, fmt.Sprintf("m_%s_sum_per_sec", v))
	}
	sparklinePointFieldsToQuery = append(sparklinePointFieldsToQuery, "num_queries_per_sec")

	for rows.Next() {
		p := qanpb.Point{}
		res := getPointFieldsList(&p, sparklinePointFieldsToQuery)
		err = rows.Scan(res...)
		if err != nil {
			return nil, errors.Wrap(err, "DimensionReport scan error")
		}
		resultsWithGaps[p.Point] = &p
	}

	// fill in gaps in time series.
	for pointN := uint32(0); int64(pointN) < amountOfPoints; pointN++ {
		p, ok := resultsWithGaps[pointN]
		if !ok {
			p = &qanpb.Point{}
			p.Point = pointN
			p.TimeFrame = uint32(timeFrame)
			timeShift := timeFrame * int64(pointN)
			ts := periodStartToSec - timeShift
			// p.Timestamp = &timestamp.Timestamp{Seconds: ts}
			p.Timestamp = time.Unix(ts, 0).UTC().Format(time.RFC3339)
		}
		results = append(results, p)
	}

	return results, err
}

const queryServers = `
	SELECT 'server' AS key, server AS value, SUM(%s) AS main_metric_sum
	  FROM metrics
	 WHERE period_start >= ?
	   AND period_start <= ?
  GROUP BY server
  	  WITH TOTALS
  ORDER BY main_metric_sum DESC, value;
`
const queryDatabases = `
	SELECT 'database' AS key, database AS value, SUM(%s) AS main_metric_sum
	  FROM metrics
	 WHERE period_start >= ?
	   AND period_start <= ?
  GROUP BY database
      WITH TOTALS
  ORDER BY main_metric_sum DESC, value;
`
const querySchemas = `
	SELECT 'schema' AS key, schema AS value, SUM(%s) AS main_metric_sum
	  FROM metrics
	 WHERE period_start >= ?
	   AND period_start <= ?
  GROUP BY schema
      WITH TOTALS
  ORDER BY main_metric_sum DESC, value;
`
const queryUsernames = `
	SELECT 'username' AS key, username AS value, SUM(%s) AS main_metric_sum
	  FROM metrics
	 WHERE period_start >= ?
	   AND period_start <= ?
  GROUP BY username
      WITH TOTALS
  ORDER BY main_metric_sum DESC, value;
`
const queryClientHosts = `
	SELECT 'client_host' AS key, client_host AS value, SUM(%s) AS main_metric_sum
	  FROM metrics
	 WHERE period_start >= ?
	   AND period_start <= ?
  GROUP BY client_host
      WITH TOTALS
  ORDER BY main_metric_sum DESC, value;
`
const queryReplicationSet = `
	SELECT 'replication_set' AS key, replication_set AS value, SUM(%s) AS main_metric_sum
	  FROM metrics
	 WHERE period_start >= ?
	   AND period_start <= ?
  GROUP BY replication_set
      WITH TOTALS
  ORDER BY main_metric_sum DESC, value;
`
const queryCluster = `
	SELECT 'cluster' AS key, cluster AS value, SUM(%s) AS main_metric_sum
	  FROM metrics
	 WHERE period_start >= ?
	   AND period_start <= ?
  GROUP BY cluster
      WITH TOTALS
  ORDER BY main_metric_sum DESC, value;
`
const queryServiceType = `
	SELECT 'service_type' AS key, service_type AS value, SUM(%s) AS main_metric_sum
	  FROM metrics
	 WHERE period_start >= ?
	   AND period_start <= ?
  GROUP BY service_type
      WITH TOTALS
  ORDER BY main_metric_sum DESC, value;
`
const queryEnvironment = `
	SELECT 'environment' AS key, environment AS value, SUM(%s) AS main_metric_sum
	  FROM metrics
	 WHERE period_start >= ?
	   AND period_start <= ?
  GROUP BY environment
      WITH TOTALS
  ORDER BY main_metric_sum DESC, value;
`
const queryAZ = `
	SELECT 'az' AS key, az AS value, SUM(%s) AS main_metric_sum
	  FROM metrics
	 WHERE period_start >= ?
	   AND period_start <= ?
  GROUP BY az
      WITH TOTALS
  ORDER BY main_metric_sum DESC, value;
`
const queryRegion = `
	SELECT 'region' AS key, region AS value, SUM(%s) AS main_metric_sum
	  FROM metrics
	 WHERE period_start >= ?
	   AND period_start <= ?
  GROUP BY region
      WITH TOTALS
  ORDER BY main_metric_sum DESC, value;
`
const queryNodeModel = `
	SELECT 'node_model' AS key, node_model AS value, SUM(%s) AS main_metric_sum
	  FROM metrics
	 WHERE period_start >= ?
	   AND period_start <= ?
  GROUP BY node_model
      WITH TOTALS
  ORDER BY main_metric_sum DESC, value;
`
const queryContainerName = `
	SELECT 'container_name' AS key, container_name AS value, SUM(%s) AS main_metric_sum
	  FROM metrics
	 WHERE period_start >= ?
	   AND period_start <= ?
  GROUP BY container_name
      WITH TOTALS
  ORDER BY main_metric_sum DESC, value;
`
const queryLabels = `
	SELECT labels.key AS key, labels.value AS value, SUM(%s) AS main_metric_sum
	  FROM metrics
ARRAY JOIN labels
	 WHERE period_start >= ?
	   AND period_start <= ?
  GROUP BY labels.key, labels.value
  ORDER BY main_metric_sum DESC, labels.key, labels.value;
`

type customLabel struct {
	key              string
	value            string
	mainMetricPerSec float32
}

// SelectFilters selects dimension and their values, and also keys and values of labels.
func (r *Reporter) SelectFilters(ctx context.Context, periodStartFromSec, periodStartToSec int64, mainMetricName string) (*qanpb.FiltersReply, error) {
	result := qanpb.FiltersReply{
		Labels: make(map[string]*qanpb.ListLabels),
	}

	if !isValidMetricColumn(mainMetricName) {
		return nil, fmt.Errorf("invalid main metric name %s", mainMetricName)
	}

	dimentionQueries := map[string]string{
		"server":          queryServers,
		"database":        queryDatabases,
		"schema":          querySchemas,
		"username":        queryUsernames,
		"client_host":     queryClientHosts,
		"replication_set": queryReplicationSet,
		"cluster":         queryCluster,
		"service_type":    queryServiceType,
		"environment":     queryEnvironment,
		"az":              queryAZ,
		"region":          queryRegion,
		"node_model":      queryNodeModel,
		"container_name":  queryContainerName,
		"labels":          queryLabels,
	}
	for dimentionName, dimentionQuery := range dimentionQueries {
		values, mainMetricPerSec, err := r.queryFilters(ctx, periodStartFromSec, periodStartToSec, mainMetricName, dimentionQuery)
		if err != nil {
			return nil, errors.Wrap(err, "cannot select "+dimentionName+" dimension")
		}

		totals := map[string]float32{}
		if mainMetricPerSec == 0 {
			for _, label := range values {
				totals[label.key] += label.mainMetricPerSec
			}
		}

		for _, label := range values {
			if _, ok := result.Labels[label.key]; !ok {
				result.Labels[label.key] = &qanpb.ListLabels{
					Name: []*qanpb.Values{},
				}
			}
			total := mainMetricPerSec
			if mainMetricPerSec == 0 {
				total = totals[label.key]
			}
			val := qanpb.Values{
				Value:             label.value,
				MainMetricPerSec:  label.mainMetricPerSec,
				MainMetricPercent: label.mainMetricPerSec / total,
			}
			result.Labels[label.key].Name = append(result.Labels[label.key].Name, &val)
		}
	}

	return &result, nil
}

func (r *Reporter) queryFilters(ctx context.Context, periodStartFromSec, periodStartToSec int64, mainMetricName, query string) ([]*customLabel, float32, error) {
	durationSec := periodStartToSec - periodStartFromSec
	var labels []*customLabel
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(query, mainMetricName), periodStartFromSec, periodStartToSec)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to select for query: "+query)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var label customLabel
		err = rows.Scan(&label.key, &label.value, &label.mainMetricPerSec)
		if err != nil {
			return nil, 0, errors.Wrap(err, "failed to scan for query: "+query)
		}
		label.mainMetricPerSec /= float32(durationSec)
		labels = append(labels, &label)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, errors.Wrap(err, "failed to select for query: "+query)
	}

	totalMainMetricPerSec := float32(0)

	if rows.NextResultSet() {
		var labelTotal customLabel
		for rows.Next() {
			err = rows.Scan(&labelTotal.key, &labelTotal.value, &labelTotal.mainMetricPerSec)
			if err != nil {
				return nil, 0, errors.Wrap(err, "failed to scan total for query: "+query)
			}
			totalMainMetricPerSec = labelTotal.mainMetricPerSec / float32(durationSec)
		}
		if err = rows.Err(); err != nil {
			return nil, 0, errors.Wrap(err, "failed to select total for query: "+query)
		}
	}

	return labels, totalMainMetricPerSec, nil
}
