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

// Package qan contains integration tests for the Query Analytics API.
//
// These tests need no QAN data: an empty report is a valid answer. What they pin is that
// the server answers at all -- PMM-15160 was a panic reaching the client as a 500 -- and
// that a malformed request is rejected as InvalidArgument rather than as an internal
// error.
//
// Two of them issue raw HTTP requests rather than using the generated Swagger client,
// against the convention in api-tests/AGENTS.md. That is deliberate and the reason is
// worth recording: the report queries emit ClickHouse NaN both as a deliberate "metric
// not collected" sentinel in the sparkline columns and as the result of aggregating over
// zero rows. ProtoJSON renders those as the JSON string "NaN", while the generated client
// declares every metric field as float32, so it cannot decode a successful report at all:
//
//	json: cannot unmarshal string into Go struct field
//	  GetReportOKBodyRowsItems0MetricsAnonStats.rows.metrics.stats.p99 of type float32
//
// Until that is settled -- either the sentinel changes or the generated field types do --
// the success paths can only be asserted over the wire. Each such test says so inline.
package qan

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	qanClient "github.com/percona/pmm/api/qan/v1/json/client"
	qanService "github.com/percona/pmm/api/qan/v1/json/client/qan_service"
)

func reportBody(from, to time.Time) qanService.GetReportBody {
	return qanService.GetReportBody{
		PeriodStartFrom: strfmt.DateTime(from),
		PeriodStartTo:   strfmt.DateTime(to),
		GroupBy:         "queryid",
		Columns:         []string{"load", "num_queries", "query_time"},
		OrderBy:         "-load",
		Limit:           10,
	}
}

// postRaw sends body to a PMM Server path and returns the status code and response body.
// It exists only for the responses the generated client cannot decode; see the package
// comment.
func postRaw(t *testing.T, path, body string) (int, string) {
	t.Helper()

	u := *pmmapitests.BaseURL
	u.Path = path
	password, _ := u.User.Password()
	username := u.User.Username()
	u.User = nil

	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodPost, u.String(), bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(username, password)

	client := &http.Client{Timeout: 30 * time.Second}
	if pmmapitests.ServerInsecureTLS {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	res, err := client.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	payload, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return res.StatusCode, string(payload)
}

func rawReportBody(from, to time.Time) string {
	return fmt.Sprintf(`{"period_start_from":%q,"period_start_to":%q,`+
		`"group_by":"queryid","columns":["load","num_queries","query_time"],`+
		`"order_by":"-load","limit":10}`,
		from.Format(time.RFC3339), to.Format(time.RFC3339))
}

// TestReportDegeneratePeriod covers PMM-15160: when both bounds of a report fall inside
// the same wall-clock minute the sparkline period collapses to zero, which used to divide
// by zero and fail the request with a panic. Such a period is unusual but entirely legal.
//
// Asserted over the wire -- see the package comment on why the generated client cannot be
// used for a successful report.
func TestReportDegeneratePeriod(t *testing.T) {
	t.Parallel()

	base := time.Now().UTC().Truncate(time.Minute).Add(-time.Hour)

	for _, tc := range []struct {
		name     string
		from, to time.Time
	}{
		{"same minute", base.Add(10 * time.Second), base.Add(50 * time.Second)},
		{"identical timestamps", base, base},
		{"one minute", base, base.Add(time.Minute)},
		{"reversed within a minute", base.Add(50 * time.Second), base.Add(10 * time.Second)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			code, body := postRaw(t, "/v1/qan/metrics:getReport", rawReportBody(tc.from, tc.to))
			// A reversed range is a client error; every other shape here must succeed.
			if tc.name == "reversed within a minute" {
				assert.Equal(t, 400, code, "body: %s", body)
				return
			}
			assert.Equal(t, 200, code, "body: %s", body)
			assert.NotContains(t, body, "integer divide by zero")
		})
	}
}

// TestReportNormalPeriods is the companion of the above: ordinary ranges, including the
// two-hour boundary where the sparkline switches from one point per minute to a capped
// point count. Over the wire for the same reason.
func TestReportNormalPeriods(t *testing.T) {
	t.Parallel()

	to := time.Now().UTC().Truncate(time.Minute)

	for _, tc := range []struct {
		name string
		span time.Duration
	}{
		{"one hour", time.Hour},
		{"just under two hours", 2*time.Hour - time.Minute},
		{"exactly two hours", 2 * time.Hour},
		{"twelve hours", 12 * time.Hour},
		{"thirty days", 30 * 24 * time.Hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			code, body := postRaw(t, "/v1/qan/metrics:getReport", rawReportBody(to.Add(-tc.span), to))
			assert.Equal(t, 200, code, "body: %s", body)
		})
	}
}

// TestReportMissingBound pins the guards on absent timestamps. These cannot be expressed
// through the generated client: its period_start fields are value types, so a request
// always carries a timestamp -- an omitted one arrives as year 1, not as absent.
func TestReportMissingBound(t *testing.T) {
	t.Parallel()

	to := time.Now().UTC().Truncate(time.Minute)

	for _, tc := range []struct {
		name, body, path string
	}{
		{
			"getReport without period_start_from",
			fmt.Sprintf(`{"period_start_to":%q,"group_by":"queryid","columns":["load"],"limit":10}`, to.Format(time.RFC3339)),
			"/v1/qan/metrics:getReport",
		},
		{
			"getMetrics without period_start_from",
			fmt.Sprintf(`{"period_start_to":%q,"group_by":"queryid"}`, to.Format(time.RFC3339)),
			"/v1/qan:getMetrics",
		},
		{
			"getMetrics without period_start_to",
			fmt.Sprintf(`{"period_start_from":%q,"group_by":"queryid"}`, to.Format(time.RFC3339)),
			"/v1/qan:getMetrics",
		},
		{
			"getFilters without period_start_from",
			fmt.Sprintf(`{"period_start_to":%q,"main_metric_name":"query_time"}`, to.Format(time.RFC3339)),
			"/v1/qan/metrics:getFilters",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			code, body := postRaw(t, tc.path, tc.body)
			assert.Equal(t, 400, code, "body: %s", body)

			var payload struct {
				Code int32 `json:"code"`
			}
			require.NoError(t, json.Unmarshal([]byte(body), &payload))
			assert.Equal(t, int32(codes.InvalidArgument), payload.Code, "body: %s", body)
		})
	}
}

// TestReportInvalidRequests pins the validation that the generated client can reach: an
// error response carries no metric fields, so it decodes cleanly.
func TestReportInvalidRequests(t *testing.T) {
	t.Parallel()

	to := time.Now().UTC().Truncate(time.Minute)
	from := to.Add(-time.Hour)

	t.Run("reversed range", func(t *testing.T) {
		t.Parallel()
		params := &qanService.GetReportParams{Body: reportBody(to, from), Context: pmmapitests.Context}
		res, err := qanClient.Default.QANService.GetReport(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "cannot be later then to-date")
		assert.Nil(t, res)
	})

	t.Run("unknown group dimension", func(t *testing.T) {
		t.Parallel()
		body := reportBody(from, to)
		body.GroupBy = "no_such_dimension"
		params := &qanService.GetReportParams{Body: body, Context: pmmapitests.Context}
		res, err := qanClient.Default.QANService.GetReport(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
			"unknown group dimension: 'no_such_dimension'")
		assert.Nil(t, res)
	})

	t.Run("order column not among selected columns", func(t *testing.T) {
		t.Parallel()
		body := reportBody(from, to)
		body.Columns = []string{"load"}
		body.OrderBy = "-lock_time"
		params := &qanService.GetReportParams{Body: body, Context: pmmapitests.Context}
		res, err := qanClient.Default.QANService.GetReport(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
			"order column 'lock_time' not in selected columns: [load]")
		assert.Nil(t, res)
	})
}

// TestGetMetricsReversedRange pins a deliberate contract change made with PMM-15160:
// GetMetrics used to accept a reversed range, answering 200 with an empty body (or, with
// totals set, a point for a window that cannot match anything). It now rejects it the way
// GetReport always has.
func TestGetMetricsReversedRange(t *testing.T) {
	t.Parallel()

	to := time.Now().UTC().Truncate(time.Minute)
	from := to.Add(-time.Hour)

	for name, totals := range map[string]bool{"without totals": false, "with totals": true} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			params := &qanService.GetMetricsParams{
				Body: qanService.GetMetricsBody{
					PeriodStartFrom: strfmt.DateTime(to), // reversed on purpose
					PeriodStartTo:   strfmt.DateTime(from),
					GroupBy:         "queryid",
					Totals:          totals,
				},
				Context: pmmapitests.Context,
			}
			res, err := qanClient.Default.QANService.GetMetrics(params)
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "cannot be later then to-date")
			assert.Nil(t, res)
		})
	}
}

// TestGetFiltersFinitePercentages guards the other half of PMM-15160: when the selected
// main metric is zero across the whole period there is no total to take a percentage of,
// and the percentage used to be computed anyway, yielding NaN. The getFilters response carries no
// sparkline columns, so the generated client can decode it -- which makes a successful
// decode here part of the assertion.
func TestGetFiltersFinitePercentages(t *testing.T) {
	t.Parallel()

	to := time.Now().UTC().Truncate(time.Minute)

	for _, metric := range []string{"num_queries_with_warnings", "num_queries_with_errors", "query_time"} {
		t.Run(metric, func(t *testing.T) {
			t.Parallel()
			params := &qanService.GetFilteredMetricsNamesParams{
				Body: qanService.GetFilteredMetricsNamesBody{
					PeriodStartFrom: strfmt.DateTime(to.Add(-time.Hour)),
					PeriodStartTo:   strfmt.DateTime(to),
					MainMetricName:  metric,
				},
				Context: pmmapitests.Context,
			}
			res, err := qanClient.Default.QANService.GetFilteredMetricsNames(params)
			require.NoError(t, err)
			require.NotNil(t, res.Payload)

			for key, label := range res.Payload.Labels {
				for _, v := range label.Name {
					p := float64(v.MainMetricPercent)
					assert.False(t, math.IsNaN(p), "label %s value %s has a NaN percentage", key, v.Value)
					assert.False(t, math.IsInf(p, 0),
						"label %s value %s has an infinite percentage", key, v.Value)
				}
			}
		})
	}
}
