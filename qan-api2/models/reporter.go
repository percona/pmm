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

package models

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/prometheus/prometheus/promql/parser"
	"google.golang.org/grpc/metadata"

	qanpb "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/qan-api2/utils/clickhouse"
	"github.com/percona/pmm/qan-api2/utils/logger"
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
type M map[string]any

const LBACHeaderName = "X-Proxy-Filter"

const queryReportTmpl = `
SELECT
{{ .Group }} AS dimension,
any(database) as database_name,
{{ if eq .Group "queryid" }} any(fingerprint) {{ else }} '' {{ end }} AS fingerprint,
SUM(num_queries) AS num_queries,
{{range $j, $col := .CommonColumns}}
    SUM(m_{{ $col }}_cnt) AS m_{{ $col }}_cnt,
    SUM(m_{{ $col }}_sum) AS m_{{ $col }}_sum,
    MIN(m_{{ $col }}_min) AS m_{{ $col }}_min,
    MAX(m_{{ $col }}_max) AS m_{{ $col }}_max,
	AVG(m_{{ $col }}_p99) AS m_{{ $col }}_p99,
	m_{{ $col }}_sum/num_queries AS m_{{ $col }}_avg,
{{ end }}
{{range $j, $col := .SumColumns}}
    SUM(m_{{ $col }}_cnt) AS m_{{ $col }}_cnt,
	SUM(m_{{ $col }}_sum) AS m_{{ $col }}_sum,
	m_{{ $col }}_sum/num_queries AS m_{{ $col }}_avg,
{{ end }}
{{range $j, $col := .SpecialColumns}}
    {{ if eq $col "load" }}
        {{ if $.IsQueryTimeInSelect }}
            m_query_time_sum / {{ $.PeriodDuration }} AS load,
        {{ else }}
            SUM(m_query_time_sum) / {{ $.PeriodDuration }} AS load,
        {{ end }}
    {{ else }}
        SUM({{ $col }}) AS {{ $col }},
    {{ end }}
{{ end }}
count(DISTINCT dimension) AS total_rows
FROM metrics
WHERE period_start >= :period_start_from AND period_start <= :period_start_to
{{ if .Search }}
	{{ if eq .Group "queryid" }}
		AND ( lowerUTF8(queryid) LIKE :search OR lowerUTF8(metrics.fingerprint) LIKE :search )
	{{ else }}
		AND lowerUTF8({{ .Group }}) LIKE :search
	{{ end }}
{{ end }}
{{ if .Dimensions }}
    {{range $key, $vals := .Dimensions }}
        AND {{ $key }} IN ( '{{ StringsJoin $vals "', '" }}' )
    {{ end }}
{{ end }}
{{ if .Labels }}{{$i := 0}}
    AND ({{range $key, $vals := .Labels }}{{ $i = inc $i}}
        {{ if gt $i 1}} AND {{ end }} has(['{{ StringsJoin $vals "', '" }}'], labels.value[indexOf(labels.key, '{{ $key }}')])
    {{ end }})
{{ end }}
{{ if .Where }}
		AND	{{ .Where }}
{{ end }}
GROUP BY {{ .Group }}
        WITH TOTALS
ORDER BY {{ .Order }}
LIMIT :offset, :limit
`

var tmplQueryReport = template.Must(template.New("queryReportTmpl").Funcs(funcMap).Parse(queryReportTmpl))

// workaround to issues in closed PR https://github.com/jmoiron/sqlx/pull/579
func escapeColons(in string) string {
	return strings.ReplaceAll(in, ":", "::")
}

func escapeColonsInMap(m map[string][]string) map[string][]string {
	escapedMap := make(map[string][]string, len(m))
	for k, v := range m {
		key := escapeColons(k)
		escapedMap[key] = make([]string, len(v))
		for i, value := range v {
			escapedMap[key][i] = escapeColons(value)
		}
	}
	return escapedMap
}

// Select selects metrics for report.
func (r *Reporter) Select(ctx context.Context, periodStartFromSec, periodStartToSec int64,
	dimensions map[string][]string, labels map[string][]string,
	group, order, search string, offset, limit uint32,
	specialColumns, commonColumns, sumColumns []string,
) ([]M, error) {
	search = strings.TrimSpace(search)

	arg := map[string]any{
		"period_start_from": periodStartFromSec,
		"period_start_to":   periodStartToSec,
		"group":             group,
		"order":             order,
		"search":            "%" + strings.ToLower(search) + "%",
		"offset":            offset,
		"limit":             limit,
	}

	where, err := filtersToWhere(ctx)
	if err != nil {
		return nil, err
	}

	tmplArgs := struct {
		PeriodStartFrom     int64
		PeriodStartTo       int64
		PeriodDuration      int64
		Dimensions          map[string][]string
		Labels              map[string][]string
		Group               string
		Order               string
		Search              string
		Offset              uint32
		Limit               uint32
		SpecialColumns      []string
		CommonColumns       []string
		SumColumns          []string
		IsQueryTimeInSelect bool
		Where               string
	}{
		PeriodStartFrom:     periodStartFromSec,
		PeriodStartTo:       periodStartToSec,
		PeriodDuration:      periodStartToSec - periodStartFromSec,
		Dimensions:          escapeColonsInMap(dimensions),
		Labels:              escapeColonsInMap(labels),
		Group:               group,
		Order:               order,
		Search:              search,
		Offset:              offset,
		Limit:               limit,
		SpecialColumns:      specialColumns,
		CommonColumns:       commonColumns,
		SumColumns:          sumColumns,
		IsQueryTimeInSelect: slices.Contains(commonColumns, "query_time"),
		Where:               where,
	}

	var queryBuffer bytes.Buffer

	if err := tmplQueryReport.Execute(&queryBuffer, tmplArgs); err != nil {
		return nil, fmt.Errorf("cannot execute tmplQueryReport: %w", err)
	}

	var results []M
	query, args, err := sqlx.Named(queryBuffer.String(), arg)
	if err != nil {
		return nil, fmt.Errorf("prepare named tmplQueryReport: %w", err)
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		return nil, fmt.Errorf("populate arguments in IN clause: %w", err)
	}
	query = r.db.Rebind(query)

	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryxContext(queryCtx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("QueryxContext error: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		result := make(M)
		err = rows.MapScan(result)
		if err != nil {
			return nil, fmt.Errorf("DimensionReport Scan error: %w", err)
		}
		results = append(results, result)
	}
	rows.NextResultSet()
	total := make(M)
	for rows.Next() {
		err = rows.MapScan(total)
		if err != nil {
			return nil, fmt.Errorf("DimensionReport Scan TOTALS error: %w", err)
		}
		results = append([]M{total}, results...)
	}
	return results, err
}

const queryReportSparklinesTmpl = `
SELECT
    intDivOrZero(toUnixTimestamp( :period_start_to ) - toUnixTimestamp(period_start), {{ .TimeFrame }}) AS point,
    toDateTime(toUnixTimestamp( :period_start_to ) - (point * {{ .TimeFrame }})) AS timestamp,
    {{ .TimeFrame }} AS time_frame,
    {{ if .IsCommon }}
        if(SUM(m_{{ .Column }}_cnt) == 0, NaN, SUM(m_{{ .Column }}_sum) / time_frame) AS m_{{ .Column }}_sum_per_sec
		{{ else }}
			{{ if eq .Column "num_queries" }}
				SUM(num_queries) / time_frame AS num_queries_per_sec
			{{ end }}
			{{ if eq .Column "num_queries_with_errors" }}
				SUM(num_queries_with_errors) / time_frame AS num_queries_with_errors_per_sec
			{{ end }}
			{{ if eq .Column "num_queries_with_warnings" }}
				SUM(num_queries_with_warnings) / time_frame AS num_queries_with_warnings_per_sec
			{{ end }}
			{{ if eq .Column "load" }}
				SUM(m_query_time_sum) / time_frame AS load
			{{ end }}
		{{ end }}
FROM metrics
WHERE period_start >= :period_start_from AND period_start <= :period_start_to
{{ if not .IsTotal }} AND {{ .Group }} = '{{ .DimensionVal }}' {{ end }}
    {{range $key, $vals := .Dimensions }} AND {{ $key }} IN ( '{{ StringsJoin $vals "', '" }}' ){{ end }}
{{ if .Labels }}{{$i := 0}}
    AND ({{range $key, $val := .Labels }} {{ $i = inc $i}}
        {{ if gt $i 1}} OR {{ end }} has(['{{ StringsJoin $val "', '" }}'], labels.value[indexOf(labels.key, '{{ $key }}')])
    {{ end }})
{{ end }}
{{ if .Where }}
		AND	{{ .Where }}
{{ end }}
GROUP BY point
ORDER BY point ASC;
`

var tmplQueryReportSparklines = template.Must(template.New("queryReportSparklines").Funcs(funcMap).Parse(queryReportSparklinesTmpl))

// SelectSparklines selects datapoint for sparklines.
func (r *Reporter) SelectSparklines(ctx context.Context, dimensionVal string,
	periodStartFromSec, periodStartToSec int64,
	dimensions map[string][]string, labels map[string][]string,
	group string, column string, isTotal bool,
) ([]*qanpb.Point, error) {
	// Align to minutes
	periodStartToSec = periodStartToSec / 60 * 60
	periodStartFromSec = periodStartFromSec / 60 * 60

	// If time range is bigger then two hour - amount of sparklines points = 120 to avoid huge data in response.
	// Otherwise amount of sparklines points is equal to minutes in time range to not mess up calculation.
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

	arg := map[string]any{
		"dimension_val":     dimensionVal,
		"period_start_from": periodStartFromSec,
		"period_start_to":   periodStartToSec,
		"group":             group,
		"column":            column,
		"time_frame":        timeFrame,
	}

	where, err := filtersToWhere(ctx)
	if err != nil {
		return nil, err
	}

	tmplArgs := struct {
		DimensionVal    string
		PeriodStartFrom int64
		PeriodStartTo   int64
		PeriodDuration  int64
		Dimensions      map[string][]string
		Labels          map[string][]string
		Group           string
		Column          string
		IsCommon        bool
		TimeFrame       int64
		IsTotal         bool
		Where           string
	}{
		DimensionVal:    escapeColons(dimensionVal),
		PeriodStartFrom: periodStartFromSec,
		PeriodStartTo:   periodStartToSec,
		PeriodDuration:  periodStartToSec - periodStartFromSec,
		Dimensions:      escapeColonsInMap(dimensions),
		Labels:          escapeColonsInMap(labels),
		Group:           group,
		Column:          column,
		IsCommon:        !slices.Contains([]string{"load", "num_queries", "num_queries_with_errors", "num_queries_with_warnings"}, column),
		TimeFrame:       timeFrame,
		IsTotal:         isTotal,
		Where:           where,
	}

	var results []*qanpb.Point
	var queryBuffer bytes.Buffer

	if err := tmplQueryReportSparklines.Execute(&queryBuffer, tmplArgs); err != nil {
		return nil, fmt.Errorf("cannot execute tmplQueryReportSparklines: %w", err)
	}
	query, args, err := sqlx.Named(queryBuffer.String(), arg)
	if err != nil {
		return nil, fmt.Errorf("cannot prepare named: %w", err)
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		return nil, fmt.Errorf("populate arguments in IN clause: %w", err)
	}
	query = r.db.Rebind(query)

	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryxContext(queryCtx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("report query: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	resultsWithGaps := make(map[uint32]*qanpb.Point)

	var mainMetricColumnName string
	switch column {
	case "":
		mainMetricColumnName = "m_query_time_sum"
	case "load":
		mainMetricColumnName = "load"
	case "num_queries":
		mainMetricColumnName = "num_queries_per_sec"
	case "num_queries_with_errors":
		mainMetricColumnName = "num_queries_with_errors_per_sec"
	case "num_queries_with_warnings":
		mainMetricColumnName = "num_queries_with_warnings_per_sec"
	default:
		mainMetricColumnName = fmt.Sprintf("m_%s_sum_per_sec", column)
	}

	sparklinePointFieldsToQuery := []string{
		"point",
		"timestamp",
		"time_frame",
		mainMetricColumnName,
	}

	for rows.Next() {
		p := qanpb.Point{}
		res := getPointFieldsList(&p, sparklinePointFieldsToQuery)
		err = rows.Scan(res...)
		if err != nil {
			return nil, fmt.Errorf("SelectSparklines scan errors: %w", err)
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
			p.Timestamp = time.Unix(ts, 0).UTC().Format(time.RFC3339)
		}
		results = append(results, p)
	}

	return results, err
}

const queryDimension = `
SELECT
    key,
    value,
    sum(main_metric_sum) AS main_metric_sum
FROM
(
    SELECT
        '{{ .DimensionName }}' AS key,
        {{ .DimensionName }} AS value,
        SUM({{ .MainMetric }}) AS main_metric_sum
    FROM metrics
    WHERE (period_start >= ?) AND (period_start <= ?)
    {{range $key, $vals := .Dimensions }} AND ({{ $key }} IN ('{{ StringsJoin $vals "', '" }}')){{ end }}
		{{ if .Where }}
			AND	{{ .Where }}
		{{ end }}
    GROUP BY {{ .DimensionName }}
    UNION ALL
    SELECT
        '{{ .DimensionName }}' AS key,
        {{ .DimensionName }} AS value,
        0 AS main_metric_sum
    FROM metrics
    WHERE (period_start >= ?) AND (period_start <= ?)
		{{ if .Where }}
			AND	{{ .Where }}
		{{ end }}
    GROUP BY {{ .DimensionName }}
)
GROUP BY
    key,
    value
    WITH TOTALS
ORDER BY
    main_metric_sum DESC,
    value ASC
`

type customLabel struct {
	key              string
	value            string
	mainMetricPerSec float32
}

var (
	queryDimensionTmpl = template.Must(template.New("queryDimension").Funcs(funcMap).Parse(queryDimension))
	dimensionQueries   = map[string]*template.Template{
		"service_name":     queryDimensionTmpl,
		"database":         queryDimensionTmpl,
		"schema":           queryDimensionTmpl,
		"username":         queryDimensionTmpl,
		"client_host":      queryDimensionTmpl,
		"replication_set":  queryDimensionTmpl,
		"cluster":          queryDimensionTmpl,
		"service_type":     queryDimensionTmpl,
		"service_id":       queryDimensionTmpl,
		"environment":      queryDimensionTmpl,
		"az":               queryDimensionTmpl,
		"region":           queryDimensionTmpl,
		"node_model":       queryDimensionTmpl,
		"node_id":          queryDimensionTmpl,
		"node_name":        queryDimensionTmpl,
		"node_type":        queryDimensionTmpl,
		"machine_id":       queryDimensionTmpl,
		"container_name":   queryDimensionTmpl,
		"container_id":     queryDimensionTmpl,
		"cmd_type":         queryDimensionTmpl,
		"top_queryid":      queryDimensionTmpl,
		"application_name": queryDimensionTmpl,
		"planid":           queryDimensionTmpl,
		"plan_summary":     queryDimensionTmpl,
	}
)

// SelectFilters selects dimension and their values, and also keys and values of labels.
func (r *Reporter) SelectFilters(ctx context.Context, periodStartFromSec, periodStartToSec int64, mainMetricName string, dimensions, labels map[string][]string) (*qanpb.GetFilteredMetricsNamesResponse, error) { //nolint:lll
	if !isValidMetricColumn(mainMetricName) {
		return nil, fmt.Errorf("invalid main metric name %s", mainMetricName)
	}

	result := qanpb.GetFilteredMetricsNamesResponse{
		Labels: r.commentsIntoGroupLabels(ctx, periodStartFromSec, periodStartToSec),
	}

	for dimensionName, dimensionQuery := range dimensionQueries {
		subDimensions := make(map[string][]string)
		for k, v := range dimensions {
			if k == dimensionName {
				continue
			}
			subDimensions[k] = v
		}
		values, mainMetricPerSec, err := r.queryFilters(ctx, periodStartFromSec, periodStartToSec, dimensionName, mainMetricName, dimensionQuery, subDimensions, labels)
		if err != nil {
			return nil, fmt.Errorf("cannot select %s dimension: %w", dimensionName, err)
		}

		totals := make(map[string]float32)
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

func (r *Reporter) queryFilters(ctx context.Context, periodStartFromSec,
	periodStartToSec int64, dimensionName, mainMetricName string, tmplQueryFilter *template.Template, queryDimensions, queryLabels map[string][]string,
) ([]*customLabel, float32, error) {
	durationSec := periodStartToSec - periodStartFromSec
	var labels []*customLabel
	// l := logger.Get(ctx)

	where, err := filtersToWhere(ctx)
	if err != nil {
		return nil, 0, err
	}

	tmplArgs := struct {
		MainMetric    string
		DimensionName string
		Dimensions    map[string][]string
		Labels        map[string][]string
		Where         string
	}{
		mainMetricName,
		dimensionName,
		queryDimensions,
		queryLabels,
		where,
	}

	var queryBuffer bytes.Buffer

	if err := tmplQueryFilter.Execute(&queryBuffer, tmplArgs); err != nil {
		return nil, 0, fmt.Errorf("cannot execute tmplQueryFilter %s: %w", queryBuffer.String(), err)
	}

	rows, err := r.db.QueryContext(ctx, queryBuffer.String(), periodStartFromSec, periodStartToSec, periodStartFromSec, periodStartToSec)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to select for QueryFilter %s: %w", queryBuffer.String(), err)
	}
	defer rows.Close() //nolint:errcheck
	// if strings.Contains(queryBuffer.String(), "service_type AS value") {
	// 	l.Infof("QueryFilter: %s, startFrom: %d, startTo: %d", queryBuffer.String(), periodStartFromSec, periodStartToSec)
	// }

	for rows.Next() {
		var label customLabel
		err = rows.Scan(&label.key, &label.value, &label.mainMetricPerSec)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan for QueryFilter %s: %w", queryBuffer.String(), err)
		}
		label.mainMetricPerSec /= float32(durationSec)
		labels = append(labels, &label)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to select for QueryFilter %s: %w", queryBuffer.String(), err)
	}

	totalMainMetricPerSec := float32(0)

	if rows.NextResultSet() {
		var labelTotal customLabel
		for rows.Next() {
			err = rows.Scan(&labelTotal.key, &labelTotal.value, &labelTotal.mainMetricPerSec)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to scan total for QueryFilter %s: %w", queryBuffer.String(), err)
			}
			totalMainMetricPerSec = labelTotal.mainMetricPerSec / float32(durationSec)
		}
		if err = rows.Err(); err != nil {
			return nil, 0, fmt.Errorf("failed to select total for QueryFilter %s: %w", queryBuffer.String(), err)
		}
	}

	return labels, totalMainMetricPerSec, nil
}

const queryLabels = `
SELECT DISTINCT
	labels.key,
	labels.value
FROM metrics
WHERE (period_start >= ?) AND (period_start <= ?)
{{ if .Where }}
	AND	{{ .Where }}
{{ end }}
ORDER BY
	labels.value ASC
`

type customLabelArray struct {
	keys   []string
	values []string
}

var queryLabelsTmpl = template.Must(template.New("queryLabels").Parse(queryLabels))

// queryLabels retrieves all labels and their values for the given time range.
func (r *Reporter) queryLabels(ctx context.Context, periodStartFromSec, periodStartToSec int64) ([]*customLabelArray, error) {
	where, err := filtersToWhere(ctx)
	if err != nil {
		return nil, err
	}

	var queryBuffer bytes.Buffer
	tmplArgs := struct {
		Where string
	}{
		Where: where,
	}

	if err := queryLabelsTmpl.Execute(&queryBuffer, tmplArgs); err != nil {
		return nil, fmt.Errorf("cannot execute queryLabelsTmpl: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, queryBuffer.String(), periodStartFromSec, periodStartToSec)
	if err != nil {
		return nil, fmt.Errorf("failed to select for QueryFilter %s: %w", queryLabels, err)
	}
	defer rows.Close() //nolint:errcheck

	var labels []*customLabelArray
	for rows.Next() {
		var label customLabelArray
		err = rows.Scan(&label.keys, &label.values)
		if err != nil {
			return nil, fmt.Errorf("failed to scan for QueryFilter %s: %w", queryLabels, err)
		}
		labels = append(labels, &label)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to select for QueryFilter %s: %w", queryLabels, err)
	}

	return labels, nil
}

// commentsIntoGroupLabels parse labels and comment labels into filter groups and values.
func (r *Reporter) commentsIntoGroupLabels(ctx context.Context, periodStartFromSec, periodStartToSec int64) map[string]*qanpb.ListLabels {
	groupLabels := make(map[string]*qanpb.ListLabels)

	labelKeysValues, err := r.queryLabels(ctx, periodStartFromSec, periodStartToSec)
	if err != nil {
		// logger.Get(ctx).Errorf("Failed to query labels: %s", err)
		return groupLabels
	}

	count := len(labelKeysValues)
	res := make(map[string]map[string]float32)
	for _, label := range labelKeysValues {
		for index, key := range label.keys {
			if _, ok := res[key]; !ok {
				res[key] = make(map[string]float32)
			}

			res[key][label.values[index]]++
		}
	}

	for key, values := range res {
		if _, ok := groupLabels[key]; !ok {
			groupLabels[key] = &qanpb.ListLabels{
				Name: []*qanpb.Values{},
			}
		}

		for k, v := range values {
			val := qanpb.Values{
				Value:             k,
				MainMetricPercent: v / float32(count),
			}
			groupLabels[key].Name = append(groupLabels[key].Name, &val)
		}
	}

	return groupLabels
}

// parseFilters decodes and unmarshals the filters.
func parseFilters(filters []string) (string, error) {
	switch len(filters) {
	case 0:
		return "", nil
	case 2: //nolint:mnd
		// There should be only one filter, otherwise it's an error.
	default:
		return "", errors.New("multiple filters are not supported")
	}

	decoded, err := base64.StdEncoding.DecodeString(filters[0])
	if err != nil {
		return "", fmt.Errorf("failed to decode filters %s: %w", filters[0], err)
	}

	var parsed []string
	if err := json.Unmarshal(decoded, &parsed); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	return parsed[0], nil
}

// filtersToWhere extracts filters from the context and converts them to an SQL WHERE clause.
func filtersToWhere(ctx context.Context) (string, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("failed to extract metadata from context")
	}

	l := logger.Get(ctx)
	var selector, where string
	var err error
	if filters := headers.Get(LBACHeaderName); len(filters) > 0 {
		selector, err = parseFilters(filters)
		if err != nil {
			return "", fmt.Errorf("failed to parse filters: %v", err)
		}
		l.Infof("Parsed: %s", selector)
	}

	if selector != "" {
		// Example with four selector types: `{service_type=~"mysql|mongodb", environment!~"prod", environment="dev", az!="us-east-1"}`)
		matchers, err := parser.ParseMetricSelector(selector)
		if err != nil {
			l.Errorf("Failed to parse metric selector: %v", err)
		}

		where, err = clickhouse.MatchersToClickHouse(matchers)
		if err != nil {
			l.Errorf("Failed to convert label matchers to WHERE condition: %s", err)
		}
		l.Infof("WHERE: %s", where)
	}

	return where, nil
}
