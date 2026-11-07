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

package mcp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "rewrite golden files")

// failed is the fixture of a backing call that fails with HTTP 500.
var failed = fixture{file: "error_500.json", status: http.StatusInternalServerError}

// listedTools returns the tools a fresh server lists, by name.
func listedTools(t *testing.T) map[string]*mcp.Tool {
	t.Helper()

	return listTools(t, connect(t, newTestService(t, newFakePMM(t, nil))))
}

// requireEntries fails unless a contract test's list names exactly the tools in tools/list, so a new tool cannot skip it.
func requireEntries(t *testing.T, tools map[string]*mcp.Tool, names []string) {
	t.Helper()

	require.ElementsMatch(t, slices.Sorted(maps.Keys(tools)), names, "the test's list must have one entry for each tool in tools/list")
}

// withRoutes returns routes with extra added or replaced.
func withRoutes(routes, extra map[string]fixture) map[string]fixture {
	maps.Copy(routes, extra)
	return routes
}

// failWhen makes the fake PMM answer the requests match accepts with HTTP 500, ahead of its routes and stubs.
func failWhen(t *testing.T, fake *fakePMM, match func(*http.Request) bool) {
	t.Helper()

	body := readFixture(t, failed.file)
	next := fake.server.Config.Handler
	fake.server.Config.Handler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if !match(req) {
			next.ServeHTTP(rw, req)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(failed.status)
		_, _ = rw.Write(body)
	})
}

// validateOutput checks structured content against a tool's output schema as tools/list gives it.
func validateOutput(t *testing.T, tool *mcp.Tool, content any) {
	t.Helper()

	var schema jsonschema.Schema
	require.NoError(t, json.Unmarshal(mustJSON(t, tool.OutputSchema), &schema))
	resolved, err := schema.Resolve(nil)
	require.NoError(t, err)
	var instance any
	require.NoError(t, json.Unmarshal(mustJSON(t, content), &instance))
	assert.NoError(t, resolved.Validate(instance))
}

// TestToolsListGolden snapshots tools/list, so a change to a tool's contract shows up as a diff in review.
func TestToolsListGolden(t *testing.T) {
	t.Parallel()

	type goldenTool struct {
		Name         string               `json:"name"`
		Description  string               `json:"description"`
		InputSchema  any                  `json:"inputSchema"`
		OutputSchema any                  `json:"outputSchema"`
		Annotations  *mcp.ToolAnnotations `json:"annotations"`
	}
	tools := listedTools(t)
	golden := make([]goldenTool, 0, len(tools))
	for _, name := range slices.Sorted(maps.Keys(tools)) {
		tool := tools[name]
		golden = append(golden, goldenTool{
			Name:         tool.Name,
			Description:  tool.Description,
			InputSchema:  tool.InputSchema,
			OutputSchema: tool.OutputSchema,
			Annotations:  tool.Annotations,
		})
	}
	var got bytes.Buffer
	enc := json.NewEncoder(&got)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	require.NoError(t, enc.Encode(golden))

	path := filepath.Join("testdata", "tools_list.golden.json")
	if *update {
		require.NoError(t, os.WriteFile(path, got.Bytes(), 0o644))
	}
	want, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(want), got.String(), "tools/list changed: review the diff, then rerun with -update")
}

// untrustedPaths returns the properties under schema whose description starts with "Untrusted.", written as
// tool-contract.md §7 writes them: a.b for an object's property, a[].b for an array's items.
func untrustedPaths(schema map[string]any, prefix string) []string {
	var out []string
	props, _ := schema["properties"].(map[string]any)
	for name, p := range props {
		prop, _ := p.(map[string]any)
		desc, _ := prop["description"].(string)
		if strings.HasPrefix(desc, "Untrusted.") {
			out = append(out, prefix+name)
		}
		out = append(out, untrustedPaths(prop, prefix+name+".")...)
		items, ok := prop["items"].(map[string]any)
		if ok {
			out = append(out, untrustedPaths(items, prefix+name+"[].")...)
		}
	}
	return out
}

// TestUntrustedFieldsMarked pins the untrusted fields of tool-contract.md §7: exactly these are marked.
func TestUntrustedFieldsMarked(t *testing.T) {
	t.Parallel()

	untrusted := map[string][]string{
		"pmm_version":     {},
		"pmm_inventory":   {"services[].service_name", "services[].node_name", "services[].version", "services[].address"},
		"pmm_top_queries": {"rows[].fingerprint", "rows[].dimension"},
		"pmm_query_detail": {
			"service_name", "version", "database", "schema", "fingerprint", "example", "tables",
			"metrics_error.message", "example_error.message",
		},
		"pmm_get_explain": {"service_name", "plan"},
		"pmm_get_schema": {
			"tables[].database", "tables[].name", "tables[].definition", "tables[].indexes",
			"tables[].definition_error.message", "tables[].indexes_error.message",
		},
		"pmm_get_config": {"service_name", "version", "variables[].value"},
		"pmm_catalog":    {},
	}
	tools := listedTools(t)
	requireEntries(t, tools, slices.Collect(maps.Keys(untrusted)))

	for name, want := range untrusted {
		schema, ok := tools[name].OutputSchema.(map[string]any)
		require.True(t, ok, "%s has no output schema", name)
		assert.ElementsMatch(t, want, untrustedPaths(schema, ""), "%s: the fields marked Untrusted.", name)
	}
}

// TestErrorsCarryNoStructuredContent pins the error envelope of tool-contract.md §3.1 for every tool.
func TestErrorsCarryNoStructuredContent(t *testing.T) {
	t.Parallel()

	// Each call passes the input schema and fails in the tool: pmm_catalog on its kind, the others on a backing
	// call to a fake PMM without routes.
	calls := map[string]map[string]any{
		"pmm_version":      nil,
		"pmm_inventory":    nil,
		"pmm_top_queries":  nil,
		"pmm_query_detail": {"queryid": "QID-AAA"},
		"pmm_get_explain":  {"service_id": "svc-1", "queryid": "QID-AAA"},
		"pmm_get_schema":   {"service_id": "svc-1", "database": "shop", "tables": []string{"customers"}},
		"pmm_get_config":   {"service_id": "svc-1"},
		"pmm_catalog":      {"kind": "bogus"},
	}
	requireEntries(t, listedTools(t), slices.Collect(maps.Keys(calls)))

	for name, args := range calls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			session := connect(t, newQANService(t, newFakePMM(t, nil), false))
			res := callTool(t, session, name, args)
			text := textOf(t, res)
			assert.True(t, res.IsError, text)
			assert.Nil(t, res.StructuredContent)
			meta, ok := res.Meta["com.percona/error"].(map[string]any)
			require.True(t, ok, "_meta: %v", res.Meta)
			code, _ := meta["code"].(string)
			assert.True(t, strings.HasPrefix(text, "error: "+code+"\n"), text)
		})
	}
}

// literal is a single-quoted SQL literal.
var literal = regexp.MustCompile(`'([^']+)'`)

// TestRedactionInBothRenderings pins ADR-07's masking on the result value: with raw SQL off, no literal of
// the fixture's SQL reaches the text or the structured content.
func TestRedactionInBothRenderings(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		fake func(t *testing.T) *fakePMM
		args map[string]any
		// sql returns the SQL texts of the fixture whose literals must not come back.
		sql func(t *testing.T) []string
	}{
		"pmm_top_queries": {
			fake: func(t *testing.T) *fakePMM {
				t.Helper()
				return newFakePMM(t, withRoutes(qanRoutes(), map[string]fixture{"POST /v1/qan/metrics:getReport": {file: "report_raw_fingerprint.json"}}))
			},
			sql: func(t *testing.T) []string {
				t.Helper()
				var report struct {
					Rows []struct {
						Fingerprint string `json:"fingerprint"`
					} `json:"rows"`
				}
				require.NoError(t, json.Unmarshal(readFixture(t, "report_raw_fingerprint.json"), &report))
				out := make([]string, 0, len(report.Rows))
				for _, r := range report.Rows {
					out = append(out, r.Fingerprint)
				}
				return out
			},
		},
		"pmm_query_detail": {
			fake: func(t *testing.T) *fakePMM {
				t.Helper()
				return newFakePMM(t, qanRoutes())
			},
			args: map[string]any{"queryid": "QID-AAA"},
			sql: func(t *testing.T) []string {
				t.Helper()
				var examples struct {
					QueryExamples []struct {
						Example            string `json:"example"`
						ExplainFingerprint string `json:"explain_fingerprint"`
					} `json:"query_examples"`
				}
				require.NoError(t, json.Unmarshal(readFixture(t, "example.json"), &examples))
				out := make([]string, 0, 2*len(examples.QueryExamples))
				for _, e := range examples.QueryExamples {
					out = append(out, e.Example, e.ExplainFingerprint)
				}
				return out
			},
		},
		"pmm_get_explain": {
			fake: func(t *testing.T) *fakePMM {
				t.Helper()
				fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
				return fake
			},
			args: map[string]any{"service_id": "svc-1", "queryid": "QID-AAA"},
			sql: func(t *testing.T) []string {
				t.Helper()
				var action struct {
					Output string `json:"output"`
				}
				require.NoError(t, json.Unmarshal(readFixture(t, "action_done_explain.json"), &action))
				var envelope struct {
					ExplainResult  string `json:"explain_result"`
					ExplainedQuery string `json:"explained_query"`
				}
				require.NoError(t, json.Unmarshal([]byte(action.Output), &envelope))
				plan, err := base64.StdEncoding.DecodeString(envelope.ExplainResult)
				require.NoError(t, err)
				return []string{envelope.ExplainedQuery, string(plan)}
			},
		},
	}
	exempt := map[string]string{
		"pmm_version":    "returns no statement text",
		"pmm_inventory":  "returns no statement text",
		"pmm_catalog":    "returns no statement text",
		"pmm_get_config": "returns no statement text",
		"pmm_get_schema": "returns DDL unmasked (ADR-07)",
	}
	requireEntries(t, listedTools(t), slices.Concat(slices.Collect(maps.Keys(cases)), slices.Collect(maps.Keys(exempt))))

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var literals []string
			for _, sql := range tc.sql(t) {
				for _, m := range literal.FindAllStringSubmatch(sql, -1) {
					literals = append(literals, m[1])
				}
			}
			require.NotEmpty(t, literals, "the fixture has no literal to mask")

			res := callTool(t, connect(t, newQANService(t, tc.fake(t), false)), name, tc.args)
			text := textOf(t, res)
			require.False(t, res.IsError, text)
			require.NotNil(t, res.StructuredContent)
			structured := string(mustJSON(t, res.StructuredContent))
			for _, lit := range literals {
				assert.NotContains(t, text, lit, "text")
				assert.NotContains(t, structured, lit, "structured content")
			}
		})
	}
}

// TestEmptyAndDegradedResults pins that empty data and a failing backing call give an error result or a result
// whose structured content passes the tool's output schema, never a JSON-RPC error.
func TestEmptyAndDegradedResults(t *testing.T) {
	t.Parallel()

	type scenario struct {
		name    string
		fake    func(t *testing.T) *fakePMM
		args    map[string]any
		wantErr bool
		// check inspects the fake PMM and the error text of a case that wants an error.
		check func(t *testing.T, fake *fakePMM, text string)
	}
	routed := func(routes map[string]fixture) func(t *testing.T) *fakePMM {
		return func(t *testing.T) *fakePMM {
			t.Helper()
			return newFakePMM(t, routes)
		}
	}
	const versionQuery = "GET /graph/api/datasources/proxy/uid/metrics-uid/api/v1/query"
	schemaArgs := map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"customers"}, "info_types": "all"}

	scenarios := map[string][]scenario{
		"pmm_version": {
			{name: "Empty", fake: routed(map[string]fixture{"GET /v1/server/version": {file: "empty.json"}})},
			{name: "Failed", fake: routed(map[string]fixture{"GET /v1/server/version": failed}), wantErr: true},
		},
		"pmm_inventory": {
			{name: "Empty", fake: routed(withRoutes(inventoryRoutes(), map[string]fixture{"GET /v1/inventory/services": {file: "empty.json"}}))},
			{
				name: "NodesAndVersionsFail",
				fake: routed(withRoutes(inventoryRoutes(), map[string]fixture{"GET /v1/inventory/nodes": failed, versionQuery: failed})),
			},
		},
		"pmm_top_queries": {
			{name: "Empty", fake: routed(withRoutes(qanRoutes(), map[string]fixture{"POST /v1/qan/metrics:getReport": {file: "empty.json"}}))},
			{
				name: "ServiceLookupFails",
				fake: routed(withRoutes(qanRoutes(), map[string]fixture{
					"POST /v1/qan/metrics:getReport": {file: "report_quoted_identifier.json"},
					"GET /v1/inventory/services":     failed,
				})),
				args: map[string]any{"service_name": "shop-mysql"},
			},
		},
		"pmm_query_detail": {
			{
				name: "Empty",
				fake: routed(withRoutes(qanRoutes(), map[string]fixture{
					"POST /v1/qan:getMetrics":       {file: "metrics_empty.json"},
					"POST /v1/qan/query:getExample": {file: "example_empty.json"},
				})),
				args:    map[string]any{"queryid": "QID-AAA"},
				wantErr: true,
			},
			{
				name: "ExampleFails",
				fake: routed(withRoutes(qanRoutes(), map[string]fixture{"POST /v1/qan/query:getExample": failed, versionQuery: {file: "vm_empty.json"}})),
				args: map[string]any{"queryid": "QID-AAA"},
			},
		},
		"pmm_get_explain": {
			{
				name: "NoStoredPlan",
				fake: routed(withRoutes(qanRoutes(), map[string]fixture{
					"POST " + reportPath:            {file: "report_by_service_postgres.json"},
					"GET /v1/qan/query/QID-PG/plan": {file: "plan_empty.json"},
				})),
				args:    map[string]any{"service_id": "svc-2", "queryid": "QID-PG"},
				wantErr: true,
				check: func(t *testing.T, fake *fakePMM, text string) {
					t.Helper()
					assert.Len(t, fake.requestsTo("/v1/qan/query/QID-PG/plan"), 1, "the resolver passed and the plan was looked up")
					assert.True(t, strings.HasPrefix(text, "error: not_found\nno stored plan for queryid 'QID-PG'"), text)
				},
			},
			{
				name: "PollFails",
				fake: func(t *testing.T) *fakePMM {
					t.Helper()
					fake, _ := newActionStub(t, 0)
					failWhen(t, fake, func(req *http.Request) bool { return req.URL.Path == "/v1/actions/explain-" })
					return fake
				},
				args:    map[string]any{"service_id": "svc-1", "queryid": "QID-AAA"},
				wantErr: true,
			},
		},
		"pmm_get_schema": {
			{
				name: "Empty",
				fake: func(t *testing.T) *fakePMM {
					t.Helper()
					fake, stub := newActionStub(t, 0)
					stub.outputs = map[string]string{"ddl-customers": "", "index-customers": ""}
					return fake
				},
				args: schemaArgs,
			},
			{
				name: "IndexPollFails",
				fake: func(t *testing.T) *fakePMM {
					t.Helper()
					fake, _ := newActionStub(t, 0)
					failWhen(t, fake, func(req *http.Request) bool { return req.URL.Path == "/v1/actions/index-customers" })
					return fake
				},
				args: schemaArgs,
			},
		},
		"pmm_get_config": {
			{
				name: "Empty",
				fake: routed(withRoutes(inventoryRoutes(), map[string]fixture{versionQuery: {file: "vm_empty.json"}})),
				args: map[string]any{"service_name": "shop-mysql"},
			},
			{
				name: "VersionFails",
				fake: func(t *testing.T) *fakePMM {
					t.Helper()
					fake := newFakePMM(t, withRoutes(inventoryRoutes(), map[string]fixture{versionQuery: {file: "vm_variables.json"}}))
					failWhen(t, fake, func(req *http.Request) bool {
						return strings.Contains(req.URL.Query().Get("query"), "mysql_version_info")
					})
					return fake
				},
				args: map[string]any{"service_name": "shop-mysql"},
			},
		},
		"pmm_catalog": {
			{name: "NoBackingCalls", fake: routed(nil), args: map[string]any{"kind": "qan_columns", "engine": "mongodb"}},
		},
	}
	tools := listedTools(t)
	requireEntries(t, tools, slices.Collect(maps.Keys(scenarios)))

	for name, list := range scenarios {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for _, sc := range list {
				t.Run(sc.name, func(t *testing.T) {
					t.Parallel()

					fake := sc.fake(t)
					res := callTool(t, connect(t, newQANService(t, fake, false)), name, sc.args)
					text := textOf(t, res)
					require.Equal(t, sc.wantErr, res.IsError, text)
					if sc.wantErr {
						if sc.check != nil {
							sc.check(t, fake, text)
						}
						return
					}
					require.NotNil(t, res.StructuredContent, text)
					validateOutput(t, tools[name], res.StructuredContent)
				})
			}
		})
	}
}

// TestSchemaRefusalShape pins the SDK's input-schema refusal of tool-contract.md §3.3; a go-sdk upgrade that
// changes it fails here, and the contract changes with it.
func TestSchemaRefusalShape(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, nil)
	res := callTool(t, connect(t, newTestService(t, fake)), "pmm_top_queries", map[string]any{"limit": "ten"})
	text := textOf(t, res)
	assert.True(t, res.IsError)
	assert.True(t, strings.HasPrefix(text, `validating "arguments": `), text)
	assert.NotContains(t, res.Meta, "com.percona/error")
	assert.Nil(t, res.StructuredContent)
	assert.Zero(t, fake.requestCount(), "the handler ran")
}

// bareMaps returns the map fields reachable from typ whose json tag lacks omitempty, as "type.Field".
func bareMaps(typ reflect.Type, seen map[reflect.Type]bool) []string {
	for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array || typ.Kind() == reflect.Map {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct || seen[typ] {
		return nil
	}
	seen[typ] = true

	var out []string
	for f := range typ.Fields() {
		tag := f.Tag.Get("json")
		if (!f.IsExported() && !f.Anonymous) || tag == "-" {
			continue
		}
		_, opts, _ := strings.Cut(tag, ",")
		if f.Type.Kind() == reflect.Map && !slices.Contains(strings.Split(opts, ","), "omitempty") {
			out = append(out, typ.Name()+"."+f.Name)
		}
		out = append(out, bareMaps(f.Type, seen)...)
	}
	return out
}

// TestResultMapsAreOmitempty pins that no result can carry a nil map: its schema type is a non-nullable
// object, so the SDK's output check would turn the call into a JSON-RPC error.
func TestResultMapsAreOmitempty(t *testing.T) {
	t.Parallel()

	results := map[string]reflect.Type{
		"pmm_version":      reflect.TypeFor[versionResult](),
		"pmm_inventory":    reflect.TypeFor[inventoryResult](),
		"pmm_top_queries":  reflect.TypeFor[topQueriesResult](),
		"pmm_query_detail": reflect.TypeFor[queryDetailResult](),
		"pmm_get_explain":  reflect.TypeFor[explainResult](),
		"pmm_get_schema":   reflect.TypeFor[schemaResult](),
		"pmm_get_config":   reflect.TypeFor[configResult](),
		"pmm_catalog":      reflect.TypeFor[catalogResult](),
	}
	// Map fields without omitempty that their constructors always set, as "type.Field", each covered by
	// TestEmptyAndDegradedResults.
	alwaysSet := map[string]bool{}
	tools := listedTools(t)
	requireEntries(t, tools, slices.Collect(maps.Keys(results)))

	var bare []string
	seen := make(map[reflect.Type]bool)
	for name, typ := range results {
		schema, err := jsonschema.ForType(typ, nil)
		require.NoError(t, err)
		assert.JSONEq(t, string(mustJSON(t, schema)), string(mustJSON(t, tools[name].OutputSchema)), "%s does not return %s", name, typ.Name())
		bare = append(bare, bareMaps(typ, seen)...)
	}
	bare = slices.DeleteFunc(bare, func(field string) bool { return alwaysSet[field] })
	assert.Empty(t, bare, "map fields without omitempty")
}
