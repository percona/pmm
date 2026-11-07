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

//go:build load

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	"github.com/percona/pmm/api/inventory/v1/json/client/services_service"
)

const (
	envLoadTokens   = "PMM_MCP_LOAD_TOKENS"
	envLoadOut      = "PMM_MCP_LOAD_OUT"
	envLoadDuration = "PMM_MCP_LOAD_DURATION"

	defaultLoadDuration = 10 * time.Minute
	callers             = 5
	loopsPerCaller      = 2
	monitoredCount      = 10
	callTimeout         = 90 * time.Second
	rateLimitedPause    = 500 * time.Millisecond
	probeInterval       = 10 * time.Second
	probeRange          = 12 * time.Hour
	probeStep           = 60 * time.Second
	probeTimeout        = 60 * time.Second
	metricsStep         = 15 * time.Second
	maxDashboardRatio   = 2.0
	datasourceName      = "Metrics"
	vmJob               = "victoriametrics"
	vmCPUQuery          = `rate(process_cpu_seconds_total{job="victoriametrics"}[1m])`
	vmMemoryQuery       = `process_resident_memory_bytes{job="victoriametrics"}`
)

// ruleFailureMetrics are the rule-evaluation failure counters that PMM scrapes from Grafana alerting and from VMAlert, by job.
var ruleFailureMetrics = map[string][]string{
	"grafana": {"grafana_alerting_rule_evaluation_failures_total"},
	"vmalert": {"vmalert_execution_errors_total"},
}

const (
	rangeQuery   = `rate(mysql_global_status_questions[5m])`
	instantQuery = `rate(mysql_global_status_questions[6d])`
	maxSeries    = 2000
	queryLabels  = "range_rate_5m,instant_rate_6d,range_selector"
)

// selectorQuery is the third query shape over the first n of the db services.
func selectorQuery(n int) string {
	names := make([]string, n)
	for i := range names {
		names[i] = "db" + strconv.Itoa(i+1)
	}
	return `{__name__=~"mysql_global_status_.*", service_name=~"` + strings.Join(names, "|") + `"}`
}

// mysqlSignalQueries are the MySQL signals of healthSignals in managed/services/mcp/tools_health.go, with %[1]s for the service selector.
var mysqlSignalQueries = []string{
	`avg by (service_name) (rate(mysql_global_status_queries{%[1]s}[5m]) or irate(mysql_global_status_queries{%[1]s}[5m]))`,
	`avg by (service_name) (max(max_over_time(mysql_global_status_threads_connected{%[1]s}[5m]) or max_over_time(mysql_global_status_threads_connected{%[1]s}[5m])))`,
	`avg by (service_name) (max_over_time(mysql_global_variables_max_connections{%[1]s}[5m]) or max_over_time(mysql_global_variables_max_connections{%[1]s}[5m]))`,
	`avg by (service_name) (max_over_time(mysql_global_status_threads_running{%[1]s}[5m]) or max_over_time(mysql_global_status_threads_running{%[1]s}[5m]))`,
	`avg by (service_name) (rate(mysql_global_status_aborted_connects{%[1]s}[5m]) or irate(mysql_global_status_aborted_connects{%[1]s}[5m]))`,
	`avg by (service_name) (rate(mysql_global_status_slow_queries{%[1]s}[5m]) or irate(mysql_global_status_slow_queries{%[1]s}[5m]))`,
	`avg by (service_name) (rate(mysql_global_status_created_tmp_disk_tables{%[1]s}[5m]) or irate(mysql_global_status_created_tmp_disk_tables{%[1]s}[5m]))`,
	`clamp_min(avg by (service_name) ( (max_over_time(mysql_slave_status_seconds_behind_master{%[1]s}[5m]) or max_over_time(mysql_slave_status_seconds_behind_master{%[1]s}[5m])) - (max_over_time(mysql_slave_status_sql_delay{%[1]s}[5m]) or max_over_time(mysql_slave_status_sql_delay{%[1]s}[5m])) ),0) or clamp_min(avg by (service_name) ( (max_over_time(mysql_slave_status_seconds_behind_source{%[1]s}[5m]) or max_over_time(mysql_slave_status_seconds_behind_source{%[1]s}[5m])) - (max_over_time(mysql_slave_status_sql_delay{%[1]s}[5m]) or max_over_time(mysql_slave_status_sql_delay{%[1]s}[5m])) ),0)`,
	`avg by (service_name) ((rate(mysql_global_status_innodb_buffer_pool_reads{%[1]s}[5m]) or irate(mysql_global_status_innodb_buffer_pool_reads{%[1]s}[5m])) / (rate(mysql_global_status_innodb_buffer_pool_read_requests{%[1]s}[5m]) or irate(mysql_global_status_innodb_buffer_pool_read_requests{%[1]s}[5m])))`,
}

// stat is the maximum and the 95th percentile of a series of values.
type stat struct {
	Max float64 `json:"max"`
	P95 float64 `json:"p95"`
}

type probeSummary struct {
	Requests int     `json:"requests"`
	Errors   int     `json:"errors"`
	P95Ms    float64 `json:"p95_ms"`
}

// callStats counts the calls of one tool or one query by result code.
type callStats struct {
	Calls       int            `json:"calls"`
	Ok          int            `json:"ok"`
	RateLimited int            `json:"rate_limited"`
	Other       map[string]int `json:"other_codes"`
	P95Ms       float64        `json:"p95_ms"`
	OkP95Ms     float64        `json:"ok_p95_ms"`
}

type mcpSummary struct {
	callStats

	ByTool  map[string]callStats `json:"by_tool"`
	ByQuery map[string]callStats `json:"by_query"`
}

// runSummary is what one run, the baseline or the loaded one, measured.
type runSummary struct {
	Name            string             `json:"name"`
	Start           time.Time          `json:"start"`
	End             time.Time          `json:"end"`
	CPUCores        stat               `json:"vm_cpu_cores"`
	MemoryBytes     stat               `json:"vm_memory_bytes"`
	Dashboard       probeSummary       `json:"dashboard_probe"`
	RuleFailures    map[string]float64 `json:"rule_failure_increase"`
	RuleFailureSum  float64            `json:"rule_failure_increase_total"`
	MCP             *mcpSummary        `json:"mcp,omitempty"`
	MetricsReadFail []string           `json:"metrics_read_failures,omitempty"`
}

type loadReport struct {
	Duration           string     `json:"duration"`
	Baseline           runSummary `json:"baseline"`
	Loaded             runSummary `json:"loaded"`
	DashboardP95Ratio  float64    `json:"dashboard_p95_ratio"`
	ExtraRuleFailures  float64    `json:"extra_rule_failures"`
	NoExtraRuleFailure bool       `json:"no_extra_rule_failures"`
	DashboardWithinCap bool       `json:"dashboard_p95_within_2x"`
	ThirdQueryServices []string   `json:"third_query_services"`
	Failures           []string   `json:"failures,omitempty"`
	Pass               bool       `json:"pass"`
}

// bearerTransport sends one service-account token with every request.
type bearerTransport struct {
	next  http.RoundTripper
	token string
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.next.RoundTrip(req)
}

func connectWithToken(ctx context.Context, token string) (*mcp.ClientSession, error) {
	base := httpClient(false)
	client := mcp.NewClient(&mcp.Implementation{Name: "pmm-mcp-load", Version: "1"}, nil)
	return client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   endpoint(),
		HTTPClient: &http.Client{Transport: &bearerTransport{next: base.Transport, token: token}},
	}, nil)
}

func readTokens(t *testing.T) []string {
	t.Helper()

	path := os.Getenv(envLoadTokens)
	if path == "" {
		t.Skipf("set %s to a file of %d Viewer service-account tokens, one per line", envLoadTokens, callers)
	}
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	var tokens []string
	for line := range strings.SplitSeq(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			tokens = append(tokens, line)
		}
	}
	require.GreaterOrEqual(t, len(tokens), callers, "%s needs %d tokens", envLoadTokens, callers)
	return tokens[:callers]
}

func loadDuration(t *testing.T) time.Duration {
	t.Helper()

	v := os.Getenv(envLoadDuration)
	if v == "" {
		return defaultLoadDuration
	}
	d, err := time.ParseDuration(v)
	require.NoError(t, err)
	require.Positive(t, d)
	return d
}

// serviceIDs maps db1 to db10 to their service ids.
func serviceIDs(t *testing.T) []string {
	t.Helper()

	res, err := inventoryClient.Default.ServicesService.ListServices(&services_service.ListServicesParams{Context: pmmapitests.Context})
	require.NoError(t, err)
	byName := map[string]string{}
	for _, svc := range res.Payload.Mysql {
		byName[svc.ServiceName] = svc.ServiceID
	}
	ids := make([]string, 0, monitoredCount)
	for i := 1; i <= monitoredCount; i++ {
		id, ok := byName["db"+strconv.Itoa(i)]
		require.True(t, ok, "MySQL service db%d is not registered", i)
		ids = append(ids, id)
	}
	return ids
}

// admin reads VictoriaMetrics as Admin through the Grafana datasource proxy.
type admin struct {
	client *http.Client
	uid    string
}

func newAdmin(ctx context.Context) (*admin, error) {
	a := &admin{client: httpClient(true)}
	body, err := a.get(ctx, "graph/api/datasources/name/"+datasourceName, nil)
	if err != nil {
		return nil, err
	}
	var ds struct {
		UID string `json:"uid"`
	}
	err = json.Unmarshal(body, &ds)
	if err != nil {
		return nil, fmt.Errorf("decoding the datasource: %w", err)
	}
	if ds.UID == "" {
		return nil, fmt.Errorf("the %s datasource has no uid", datasourceName)
	}
	a.uid = ds.UID
	return a, nil
}

func (a *admin) get(ctx context.Context, path string, query url.Values) ([]byte, error) {
	u := pmmapitests.BaseURL.ResolveReference(&url.URL{Path: path, RawQuery: query.Encode()})
	u.User = nil
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s answered %d: %s", path, resp.StatusCode, b)
	}
	return b, nil
}

type promSeries struct {
	Values [][]any `json:"values"`
	Value  []any   `json:"value"`
}

func (a *admin) query(ctx context.Context, endpointName string, params url.Values) ([]promSeries, error) {
	b, err := a.get(ctx, "graph/api/datasources/proxy/uid/"+a.uid+"/api/v1/"+endpointName, params)
	if err != nil {
		return nil, err
	}
	var out struct {
		Status string `json:"status"`
		Error  string `json:"error"`
		Data   struct {
			Result []promSeries `json:"result"`
		} `json:"data"`
	}
	err = json.Unmarshal(b, &out)
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", endpointName, err)
	}
	if out.Status != "success" {
		return nil, fmt.Errorf("%s failed: %s", endpointName, out.Error)
	}
	return out.Data.Result, nil
}

func (a *admin) rangeQuery(ctx context.Context, expr string, start, end time.Time, step time.Duration) ([]promSeries, error) {
	return a.query(ctx, "query_range", url.Values{
		"query": {expr},
		"start": {strconv.FormatInt(start.Unix(), 10)},
		"end":   {strconv.FormatInt(end.Unix(), 10)},
		"step":  {strconv.Itoa(int(step.Seconds()))},
	})
}

// instant returns the sum of the values of an instant query and whether it returned any series.
func (a *admin) instant(ctx context.Context, expr string, at time.Time) (float64, bool, error) {
	series, err := a.query(ctx, "query", url.Values{"query": {expr}, "time": {strconv.FormatInt(at.Unix(), 10)}})
	if err != nil {
		return 0, false, err
	}
	var sum float64
	for _, s := range series {
		if len(s.Value) == 2 {
			sum += sampleValue(s.Value[1])
		}
	}
	return sum, len(series) > 0, nil
}

// seriesCount counts the series a selector matches over [start, end], up to maxSeries+1.
func (a *admin) seriesCount(ctx context.Context, selector string, start, end time.Time) (int, error) {
	b, err := a.get(ctx, "graph/api/datasources/proxy/uid/"+a.uid+"/api/v1/series", url.Values{
		"match[]": {selector},
		"start":   {strconv.FormatInt(start.Unix(), 10)},
		"end":     {strconv.FormatInt(end.Unix(), 10)},
		"limit":   {strconv.Itoa(maxSeries + 1)},
	})
	if err != nil {
		return 0, err
	}
	var out struct {
		Status string           `json:"status"`
		Data   []map[string]any `json:"data"`
	}
	err = json.Unmarshal(b, &out)
	if err != nil {
		return 0, fmt.Errorf("decoding series: %w", err)
	}
	if out.Status != "success" {
		return 0, fmt.Errorf("series failed: %s", b)
	}
	return len(out.Data), nil
}

// thirdQueryServices returns how many of db1 to db10, as a prefix, the third query shape selects within the series cap.
func thirdQueryServices(ctx context.Context, t *testing.T, a *admin) int {
	t.Helper()

	end := time.Now()
	start := end.Add(-6*24*time.Hour - 10*time.Minute)
	chosen := 0
	for n := 1; n <= monitoredCount; n++ {
		count, err := a.seriesCount(ctx, selectorQuery(n), start, end)
		require.NoError(t, err)
		t.Logf("third query over db1..db%d matches %d series (cap %d)", n, count, maxSeries)
		if count > maxSeries {
			break
		}
		chosen = n
	}
	require.Positive(t, chosen, "the third query over db1 alone matches more than %d series, so pmm_query_metrics refuses it", maxSeries)
	t.Logf("third query runs over db1..db%d", chosen)
	return chosen
}

func sampleValue(v any) float64 {
	s, ok := v.(string)
	if !ok {
		return math.NaN()
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return math.NaN()
	}
	return f
}

// p95 is the nearest-rank 95th percentile of values; 0 when there are none.
func p95(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	rank := int(math.Ceil(0.95 * float64(len(sorted))))
	return sorted[max(rank, 1)-1]
}

func millis(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

func durationsMs(ds []time.Duration) []float64 {
	out := make([]float64, len(ds))
	for i, d := range ds {
		out[i] = millis(d)
	}
	return out
}

// probe runs the nine MySQL signal queries of db1 in sequence, then waits out the rest of probeInterval, so slow queries stretch the cadence.
type probe struct {
	admin *admin

	mu        sync.Mutex
	latencies []time.Duration
	errors    int
}

func (p *probe) run(ctx context.Context) {
	ticker := time.NewTicker(probeInterval)
	defer ticker.Stop()
	for {
		for _, q := range mysqlSignalQueries {
			if ctx.Err() != nil {
				return
			}
			p.once(ctx, fmt.Sprintf(q, `service_name="db1"`))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *probe) once(ctx context.Context, expr string) {
	callCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	end := time.Now()
	started := time.Now()
	_, err := p.admin.rangeQuery(callCtx, expr, end.Add(-probeRange), end, probeStep)
	elapsed := time.Since(started)
	if ctx.Err() != nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.latencies = append(p.latencies, elapsed)
	if err != nil {
		p.errors++
	}
}

func (p *probe) summary() probeSummary {
	p.mu.Lock()
	defer p.mu.Unlock()
	return probeSummary{Requests: len(p.latencies), Errors: p.errors, P95Ms: p95(durationsMs(p.latencies))}
}

// call is one MCP call: its tool, query label, latency and result code.
type call struct {
	tool, query string
	latency     time.Duration
	code        string
}

type recorder struct {
	mu    sync.Mutex
	calls []call
}

func (r *recorder) add(c call) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, c)
}

func statsOf(calls []call) callStats {
	s := callStats{Calls: len(calls), Other: map[string]int{}}
	all := make([]float64, 0, len(calls))
	var ok []float64
	for _, c := range calls {
		ms := millis(c.latency)
		all = append(all, ms)
		switch c.code {
		case "ok":
			s.Ok++
			ok = append(ok, ms)
		case "rate_limited":
			s.RateLimited++
		default:
			s.Other[c.code]++
		}
	}
	s.P95Ms, s.OkP95Ms = p95(all), p95(ok)
	return s
}

func (r *recorder) summary() *mcpSummary {
	r.mu.Lock()
	defer r.mu.Unlock()
	byTool := map[string][]call{}
	byQuery := map[string][]call{}
	for _, c := range r.calls {
		byTool[c.tool] = append(byTool[c.tool], c)
		byQuery[c.query] = append(byQuery[c.query], c)
	}
	s := &mcpSummary{callStats: statsOf(r.calls), ByTool: map[string]callStats{}, ByQuery: map[string]callStats{}}
	for k, v := range byTool {
		s.ByTool[k] = statsOf(v)
	}
	for k, v := range byQuery {
		s.ByQuery[k] = statsOf(v)
	}
	return s
}

// resultCode is ok, the code of an error result ("error: <code>" on the first line), or transport_error.
func resultCode(res *mcp.CallToolResult, err error) string {
	if err != nil {
		return "transport_error"
	}
	if !res.IsError {
		return "ok"
	}
	if len(res.Content) > 0 {
		text, ok := res.Content[0].(*mcp.TextContent)
		if ok {
			first, _, _ := strings.Cut(text.Text, "\n")
			code, found := strings.CutPrefix(first, "error: ")
			if found {
				return code
			}
		}
	}
	return "error"
}

// request is what a loop sends on its i-th call, with the label its counts are kept under.
type request struct {
	args  map[string]any
	query string
}

// callLoop calls a tool in turn with the requests next returns, back to back until ctx ends.
func callLoop(ctx context.Context, session *mcp.ClientSession, rec *recorder, name string, next func(i int) request, first int) {
	for i := first; ctx.Err() == nil; i++ {
		req := next(i)
		callCtx, cancel := context.WithTimeout(ctx, callTimeout)
		started := time.Now()
		res, err := session.CallTool(callCtx, &mcp.CallToolParams{Name: name, Arguments: req.args})
		elapsed := time.Since(started)
		cancel()
		if ctx.Err() != nil {
			return
		}

		code := resultCode(res, err)
		rec.add(call{tool: name, query: req.query, latency: elapsed, code: code})
		if code == "rate_limited" {
			select {
			case <-ctx.Done():
				return
			case <-time.After(rateLimitedPause):
			}
		}
	}
}

// queryRequest is the i-th call of a pmm_query_metrics loop: the three query shapes in turn.
func queryRequest(selector string) func(i int) request {
	labels := strings.Split(queryLabels, ",")
	return func(i int) request {
		switch i % len(labels) {
		case 0:
			return request{args: map[string]any{"query": rangeQuery, "period_from": "now-6d"}, query: labels[0]}
		case 1:
			return request{args: map[string]any{"query": instantQuery}, query: labels[1]}
		default:
			return request{args: map[string]any{"query": selector, "period_from": "now-6d"}, query: labels[2]}
		}
	}
}

// runCallers starts the loops of every caller and returns when ctx ends.
func runCallers(ctx context.Context, sessions []*mcp.ClientSession, serviceIDs []string, selector string, rec *recorder) {
	var wg sync.WaitGroup
	for _, session := range sessions {
		for loop := range loopsPerCaller {
			wg.Go(func() {
				callLoop(ctx, session, rec, "pmm_query_metrics", queryRequest(selector), loop)
			})
		}
		wg.Go(func() {
			callLoop(ctx, session, rec, "pmm_service_health", func(i int) request {
				return request{args: map[string]any{"service_id": serviceIDs[i%len(serviceIDs)], "period_from": "now-6h"}, query: "service_health"}
			}, 0)
		})
	}
	wg.Wait()
}

// runOnce measures one run of d: the probe alone, or with the callers when sessions is not empty.
func runOnce(ctx context.Context, t *testing.T, a *admin, name string, d time.Duration, sessions []*mcp.ClientSession, ids []string, selector string) runSummary {
	t.Helper()

	t.Logf("run %s: %s", name, d)
	runCtx, cancel := context.WithTimeout(ctx, d)
	defer cancel()

	p := &probe{admin: a}
	rec := &recorder{}
	start := time.Now()
	var wg sync.WaitGroup
	wg.Go(func() { p.run(runCtx) })
	if len(sessions) > 0 {
		wg.Go(func() { runCallers(runCtx, sessions, ids, selector, rec) })
	}
	wg.Wait()
	end := time.Now()

	s := runSummary{Name: name, Start: start, End: end, Dashboard: p.summary(), RuleFailures: map[string]float64{}}
	if len(sessions) > 0 {
		s.MCP = rec.summary()
	}
	readMetrics(ctx, a, &s)
	return s
}

// readMetrics fills in what VictoriaMetrics, Grafana and VMAlert report about the run.
func readMetrics(ctx context.Context, a *admin, s *runSummary) {
	var err error
	s.CPUCores, err = rangeStat(ctx, a, vmCPUQuery, s.Start, s.End)
	if err != nil {
		s.MetricsReadFail = append(s.MetricsReadFail, vmJob+" cpu: "+err.Error())
	}
	s.MemoryBytes, err = rangeStat(ctx, a, vmMemoryQuery, s.Start, s.End)
	if err != nil {
		s.MetricsReadFail = append(s.MetricsReadFail, vmJob+" memory: "+err.Error())
	}

	window := int(math.Ceil(s.End.Sub(s.Start).Seconds()))
	for _, job := range slices.Sorted(maps.Keys(ruleFailureMetrics)) {
		for _, name := range ruleFailureMetrics[job] {
			expr := fmt.Sprintf(`sum(increase(%s{job="%s"}[%ds]))`, name, job, window)
			v, found, qerr := a.instant(ctx, expr, s.End)
			switch {
			case qerr != nil:
				s.MetricsReadFail = append(s.MetricsReadFail, name+": "+qerr.Error())
			case !found:
				s.MetricsReadFail = append(s.MetricsReadFail, name+": no series for job "+job+": the metric is absent or the job is not scraped")
			default:
				if math.IsNaN(v) {
					v = 0
				}
				s.RuleFailures[name] = v
				s.RuleFailureSum += v
			}
		}
	}
}

// rangeStat is the maximum and 95th percentile over a run of every sample of the series expr returns.
func rangeStat(ctx context.Context, a *admin, expr string, start, end time.Time) (stat, error) {
	series, err := a.rangeQuery(ctx, expr, start, end, metricsStep)
	if err != nil {
		return stat{}, err
	}
	var values []float64
	for _, s := range series {
		for _, v := range s.Values {
			if len(v) == 2 {
				f := sampleValue(v[1])
				if !math.IsNaN(f) {
					values = append(values, f)
				}
			}
		}
	}
	if len(values) == 0 {
		return stat{}, fmt.Errorf("no samples for %s", expr)
	}
	return stat{Max: slices.Max(values), P95: p95(values)}, nil
}

func (c callStats) line() string {
	return fmt.Sprintf("calls %d  ok %d  rate_limited %d  other %v  p95 %.0f ms  ok p95 %.0f ms", c.Calls, c.Ok, c.RateLimited, c.Other, c.P95Ms, c.OkP95Ms)
}

func (s runSummary) show(t *testing.T) {
	t.Helper()

	lines := []string{
		fmt.Sprintf("== %s run, %s to %s", s.Name, s.Start.UTC().Format(time.RFC3339), s.End.UTC().Format(time.RFC3339)),
		fmt.Sprintf("VictoriaMetrics CPU (cores)   max %.3f  p95 %.3f", s.CPUCores.Max, s.CPUCores.P95),
		fmt.Sprintf("VictoriaMetrics memory (MiB)  max %.0f  p95 %.0f", s.MemoryBytes.Max/(1<<20), s.MemoryBytes.P95/(1<<20)),
		fmt.Sprintf("Dashboard probe               requests %d  errors %d  p95 %.0f ms", s.Dashboard.Requests, s.Dashboard.Errors, s.Dashboard.P95Ms),
		fmt.Sprintf("Rule-evaluation failures      +%.0f  %v", s.RuleFailureSum, s.RuleFailures),
	}
	if s.MCP != nil {
		lines = append(lines, "MCP calls                     "+s.MCP.line())
		for _, k := range slices.Sorted(maps.Keys(s.MCP.ByTool)) {
			lines = append(lines, "  tool "+k+": "+s.MCP.ByTool[k].line())
		}
		for _, k := range slices.Sorted(maps.Keys(s.MCP.ByQuery)) {
			lines = append(lines, "  query "+k+": "+s.MCP.ByQuery[k].line())
		}
	}
	for _, f := range s.MetricsReadFail {
		lines = append(lines, "METRICS READ FAILED: "+f)
	}
	fmt.Fprintln(os.Stdout, strings.Join(lines, "\n"))
}

// failures lists why a run does not count: a probe error, a metrics read failure, or a tool with no ok call.
func (s runSummary) failures() []string {
	var out []string
	if s.Dashboard.Requests == 0 {
		out = append(out, s.Name+": the dashboard probe made no request")
	}
	if s.Dashboard.Errors > 0 {
		out = append(out, fmt.Sprintf("%s: %d dashboard probe errors", s.Name, s.Dashboard.Errors))
	}
	for _, f := range s.MetricsReadFail {
		out = append(out, s.Name+": metrics read failed: "+f)
	}
	if s.MCP != nil {
		for _, tool := range []string{"pmm_query_metrics", "pmm_service_health"} {
			if s.MCP.ByTool[tool].Ok == 0 {
				out = append(out, fmt.Sprintf("%s: no ok call of %s", s.Name, tool))
			}
		}
	}
	return out
}

// TestMetricsLoad is the acceptance run of the metrics tools (spec 7.4): the dashboard probe alone, then with five callers of three loops each.
func TestMetricsLoad(t *testing.T) {
	tokens := readTokens(t)
	outPath := os.Getenv(envLoadOut)
	require.NotEmpty(t, outPath, "set %s to the path of the JSON summary", envLoadOut)
	d := loadDuration(t)
	ids := serviceIDs(t)

	ctx := pmmapitests.Context
	a, err := newAdmin(ctx)
	require.NoError(t, err)
	chosen := thirdQueryServices(ctx, t, a)
	selector := selectorQuery(chosen)

	sessions := make([]*mcp.ClientSession, len(tokens))
	for i, token := range tokens {
		sessions[i], err = connectWithToken(ctx, token)
		require.NoError(t, err)
		t.Cleanup(func() { _ = sessions[i].Close() })
	}

	report := loadReport{Duration: d.String()}
	for i := range chosen {
		report.ThirdQueryServices = append(report.ThirdQueryServices, "db"+strconv.Itoa(i+1))
	}
	report.Baseline = runOnce(ctx, t, a, "baseline", d, nil, ids, selector)
	report.Baseline.show(t)
	report.Loaded = runOnce(ctx, t, a, "loaded", d, sessions, ids, selector)
	report.Loaded.show(t)

	report.ExtraRuleFailures = report.Loaded.RuleFailureSum - report.Baseline.RuleFailureSum
	report.NoExtraRuleFailure = report.ExtraRuleFailures <= 0
	if report.Baseline.Dashboard.P95Ms > 0 {
		report.DashboardP95Ratio = report.Loaded.Dashboard.P95Ms / report.Baseline.Dashboard.P95Ms
	}
	report.DashboardWithinCap = report.Baseline.Dashboard.P95Ms > 0 && report.DashboardP95Ratio <= maxDashboardRatio
	report.Failures = slices.Concat(report.Baseline.failures(), report.Loaded.failures())
	report.Pass = report.NoExtraRuleFailure && report.DashboardWithinCap && len(report.Failures) == 0

	fmt.Fprintf(os.Stdout, "== verdict: pass=%v  dashboard p95 ratio %.2f (cap %.1f)  extra rule-evaluation failures %.0f  third query over %v\n",
		report.Pass, report.DashboardP95Ratio, maxDashboardRatio, report.ExtraRuleFailures, report.ThirdQueryServices)
	for _, f := range report.Failures {
		fmt.Fprintln(os.Stdout, "FAIL: "+f)
	}

	b, err := json.MarshalIndent(report, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(outPath, append(b, '\n'), 0o600))

	if !report.Pass {
		t.Errorf("the run did not pass: see the verdict above")
	}
}
