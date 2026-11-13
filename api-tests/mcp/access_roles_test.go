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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	accesscontrolClient "github.com/percona/pmm/api/accesscontrol/v1beta1/json/client"
	accesscontrol "github.com/percona/pmm/api/accesscontrol/v1beta1/json/client/access_control_service"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	"github.com/percona/pmm/api/inventory/v1/json/client/services_service"
)

const (
	matrixPassword = "mcp-matrix-Passw0rd"
	matrixWindow   = "now-12h"

	// The not_found text of a service outside the caller's service list, as managed/services/mcp/resolve.go words it.
	notInServiceList = "not in your service list (services that sent metrics in the window searched)"
	// The hint of the tools that take a service_id from pmm_inventory.
	inventoryHint = "; use pmm_inventory to list service ids"
	// What pmm_service_health says about node signals that an Access Role may hide.
	nodeHiddenNote = "an Access Role that filters on service labels hides the node's series"

	fullAccessTitle = "Full access"
)

// matrixChecks maps each tool to its checks; the matrix runs every entry and the keys must equal tools/list.
var matrixChecks = map[string]func(*matrixRun, *testing.T){
	"pmm_alerts":         (*matrixRun).alerts,
	"pmm_catalog":        (*matrixRun).catalog,
	"pmm_get_config":     (*matrixRun).config,
	"pmm_get_explain":    (*matrixRun).explain,
	"pmm_get_schema":     (*matrixRun).schema,
	"pmm_inventory":      (*matrixRun).inventory,
	"pmm_list_metrics":   (*matrixRun).listMetrics,
	"pmm_query_detail":   (*matrixRun).queryDetail,
	"pmm_query_metrics":  (*matrixRun).queryMetrics,
	"pmm_service_health": (*matrixRun).health,
	"pmm_top_queries":    (*matrixRun).topQueries,
	"pmm_version":        (*matrixRun).version,
}

// nodeSignals are the names of pmm_service_health's node signals.
var nodeSignals = []string{"cpu_busy", "load1", "memory_available", "disk_read_latency", "disk_write_latency", "min_space_available"}

// engineServices names the two services of one engine, made from the same image and initialisation.
type engineServices struct {
	engine string
	a, b   string
}

var matrixEngines = []engineServices{
	{engine: "mysql", a: "mysql-a", b: "mysql-b"},
	{engine: "postgresql", a: "pg-a", b: "pg-b"},
}

// accessSetup is one row of the matrix's setups.
type accessSetup struct {
	name string
	// callers are the callers that run the setup.
	callers []accessCaller
	// prepare configures the server and returns the Access Roles that each new caller is assigned.
	prepare func(t *testing.T, suffix int64) []int64
	// allowed returns the services a caller sees, from all of them.
	allowed func(all []string) []string
	// hidesNode is set when the Access Role filters out node_exporter's series, which carry no service_name.
	hidesNode bool
}

// accessCaller is a kind of caller: a Grafana user with an organization role.
type accessCaller struct {
	name        string
	grafanaRole string
}

var (
	viewerCaller = accessCaller{name: "viewer", grafanaRole: "Viewer"}
	editorCaller = accessCaller{name: "editor", grafanaRole: "Editor"}
	adminCaller  = accessCaller{name: "admin", grafanaRole: "Admin"}

	restrictedCallers = []accessCaller{viewerCaller, editorCaller, adminCaller}
)

// matrixFixture holds what the setups share: the services as the package's admin sees them, and the join's queryids.
type matrixFixture struct {
	// all is every service name pmm_inventory returns to the package's admin, sorted.
	all []string
	// ids are the service IDs of the inventory API by service name.
	ids map[string]string
	// mcpIDs are the service IDs of the package admin's pmm_inventory by service name.
	mcpIDs map[string]string
	// nodeNames are the node names of the package admin's pmm_inventory by service name.
	nodeNames map[string]string
	// queryIDs are the join's queryids by engine.
	queryIDs map[string]string
	// catalogs are the package admin's catalog results by call.
	catalogs map[string]any
	// alerts are the rules whose alerts the pmm_alerts checks look for.
	alerts matrixAlerts
}

// toolResponse is a tool result in the shapes the checks need.
type toolResponse struct {
	isError    bool
	code       string
	text       string
	structured map[string]any
}

// caller is an MCP session whose every response is checked for hidden services.
type caller struct {
	session   *mcp.ClientSession
	hiddenRE  *regexp.Regexp
	hiddenIDs []string
}

// call calls a tool and fails the test when its response names a hidden service or ID, leaving out the echoes of arguments the tool repeats on purpose.
func (c *caller) call(t *testing.T, name string, args map[string]any, echoes ...string) toolResponse {
	t.Helper()

	res := callRetrying(t, c.session, name, args)
	require.NotEmpty(t, res.Content)

	texts := make([]string, 0, len(res.Content)+2)
	for _, content := range res.Content {
		text, ok := content.(*mcp.TextContent)
		require.True(t, ok, "unexpected content %T", content)
		texts = append(texts, text.Text)
	}
	out := toolResponse{isError: res.IsError, text: strings.Join(texts, "\n")}
	if meta, ok := res.Meta[errorMetaKey].(map[string]any); ok {
		out.code, _ = meta["code"].(string)
	}
	if res.StructuredContent != nil {
		b, err := json.Marshal(res.StructuredContent)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(b, &out.structured))
	}
	t.Logf("%s %v -> isError=%v\n%s", name, args, out.isError, out.text)

	structured, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	meta, err := json.Marshal(res.Meta)
	require.NoError(t, err)
	everything := strings.Join(append(texts, unescapeJSON(string(structured)), unescapeJSON(string(meta))), "\n")
	for _, echo := range echoes {
		escaped, err := json.Marshal(echo)
		require.NoError(t, err)
		everything = strings.ReplaceAll(everything, echo, "")
		everything = strings.ReplaceAll(everything, strings.Trim(string(escaped), `"`), "")
	}
	forms := []string{everything}
	decoded, err := url.QueryUnescape(everything)
	if err == nil {
		forms = append(forms, decoded)
	}
	for _, form := range forms {
		if c.hiddenRE != nil {
			assert.Empty(t, c.hiddenRE.FindString(form), "%s names a service the caller may not see", name)
		}
		for _, id := range c.hiddenIDs {
			assert.NotContains(t, form, id, "%s holds the ID of a service the caller may not see", name)
		}
	}
	return out
}

// unescapeJSON turns the escapes of a line break and a tab into the characters, so that a name after one is still found.
func unescapeJSON(s string) string {
	return strings.NewReplacer(`\n`, "\n", `\t`, "\t").Replace(s)
}

func (r toolResponse) requireOK(t *testing.T) {
	t.Helper()

	require.False(t, r.isError, r.text)
}

func (r toolResponse) requireError(t *testing.T, code string) {
	t.Helper()

	require.True(t, r.isError, r.text)
	assert.Equal(t, code, r.code, r.text)
	assert.True(t, strings.HasPrefix(r.text, "error: "+code+"\n"), r.text)
}

func (r toolResponse) decode(t *testing.T, out any) {
	t.Helper()

	b, err := json.Marshal(r.structured)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, out))
}

// requireNotInList checks the not_found of a service outside the caller's list: the fixed text, then the window searched, which is span long,
// then the hint; no service is named.
func (r toolResponse) requireNotInList(t *testing.T, hint string, span time.Duration) {
	t.Helper()

	r.requireError(t, "not_found")
	re := regexp.MustCompile(`^error: not_found\n` + regexp.QuoteMeta(notInServiceList) + `; searched (\S+) – (\S+)` + regexp.QuoteMeta(hint) + `$`)
	m := re.FindStringSubmatch(r.text)
	require.NotNil(t, m, r.text)
	from, err := time.Parse(time.RFC3339, m[1])
	require.NoError(t, err)
	to, err := time.Parse(time.RFC3339, m[2])
	require.NoError(t, err)
	assert.InDelta(t, span.Seconds(), to.Sub(from).Seconds(), 5, r.text)
}

func basicAuthorization(login, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(login+":"+password))
}

// waitForServiceList polls the service list of authorization once a second for up to 20 s until it is want, then logs the last set and returns on timeout.
func waitForServiceList(t *testing.T, authorization string, want []string) {
	t.Helper()

	want = sortedUnique(want)
	var got []string
	deadline := time.Now().Add(20 * time.Second)
	for {
		got = nil
		for _, row := range upRows(t, authorization) {
			if row["service_id"] != "" {
				got = append(got, row["service_name"])
			}
		}
		got = sortedUnique(got)
		if slices.Equal(got, want) {
			return
		}
		if time.Now().After(deadline) {
			t.Logf("the service list did not become %v within 20s; it is %v", want, got)
			return
		}
		select {
		case <-pmmapitests.Context.Done():
			require.FailNow(t, "canceled while waiting for the service list")
		case <-time.After(time.Second):
		}
	}
}

func sortedUnique(names []string) []string {
	out := slices.Clone(names)
	slices.Sort(out)
	return slices.Compact(out)
}

func without(names []string, drop ...string) []string {
	return slices.DeleteFunc(slices.Clone(names), func(n string) bool { return slices.Contains(drop, n) })
}

func intersection(names []string, keep ...string) []string {
	return slices.DeleteFunc(slices.Clone(names), func(n string) bool { return !slices.Contains(keep, n) })
}

// inventoryStatus returns the HTTP status of GET /v1/inventory/services for authorization.
func inventoryStatus(t *testing.T, authorization string) int {
	t.Helper()

	u := pmmapitests.BaseURL.ResolveReference(&url.URL{Path: "v1/inventory/services"})
	u.User = nil
	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodGet, u.String(), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", authorization)
	res, err := httpClient(false).Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	_, err = io.Copy(io.Discard, res.Body)
	require.NoError(t, err)
	return res.StatusCode
}

type inventoryRow struct {
	ServiceID   string `json:"service_id"`
	ServiceName string `json:"service_name"`
	Engine      string `json:"engine"`
	NodeName    string `json:"node_name"`
}

type inventoryOutput struct {
	Services []inventoryRow `json:"services"`
	Page     struct {
		Total int `json:"total"`
	} `json:"page"`
}

// newMatrixFixture reads the services and the queryids as the package's admin, and creates the rules of the pmm_alerts checks.
func newMatrixFixture(t *testing.T, suffix int64) *matrixFixture {
	t.Helper()

	admin := &caller{session: connectWith(t, adminAuthorization())}
	f := &matrixFixture{
		ids:       map[string]string{},
		mcpIDs:    map[string]string{},
		nodeNames: map[string]string{},
		queryIDs:  map[string]string{},
		catalogs:  map[string]any{},
	}

	var inventory inventoryOutput
	res := admin.call(t, "pmm_inventory", map[string]any{"limit": 200})
	res.requireOK(t)
	res.decode(t, &inventory)
	require.Len(t, inventory.Services, inventory.Page.Total, "the package's admin must see every service on one page")
	for _, row := range inventory.Services {
		f.all = append(f.all, row.ServiceName)
		f.mcpIDs[row.ServiceName] = row.ServiceID
		f.nodeNames[row.ServiceName] = row.NodeName
	}
	f.all = sortedUnique(f.all)
	for _, name := range []string{"mysql-a", "mysql-b", "pg-a", "pg-b", "pmm-server-postgresql"} {
		require.Contains(t, f.all, name, "pmm_inventory of the package's admin lacks a service of the environment")
	}

	services, err := inventoryClient.Default.ServicesService.ListServices(&services_service.ListServicesParams{Context: t.Context()})
	require.NoError(t, err)
	b, err := json.Marshal(services.Payload)
	require.NoError(t, err)
	var byType map[string][]inventoryRow
	require.NoError(t, json.Unmarshal(b, &byType))
	for _, rows := range byType {
		for _, row := range rows {
			f.ids[row.ServiceName] = row.ServiceID
		}
	}
	for _, name := range []string{"mysql-a", "mysql-b", "pg-a", "pg-b"} {
		require.NotEmpty(t, f.ids[name], name)
		assert.Equal(t, f.ids[name], f.mcpIDs[name], "the inventory API and pmm_inventory disagree on the ID of %s", name)
	}

	// Anchored on SELECT: the EXPLAIN that the matrix's own pmm_get_explain calls run lands in QAN too.
	join := regexp.MustCompile(`(?is)^\s*select\b.*\borders\b.*\bjoin\b.*\bcustomers\b`)
	for _, e := range matrixEngines {
		found := map[string]string{}
		for _, name := range []string{e.a, e.b} {
			var top struct {
				Rows []struct {
					Dimension   string `json:"dimension"`
					Fingerprint string `json:"fingerprint"`
					QueryRef    *struct {
						QueryID string `json:"queryid"`
					} `json:"query_ref"`
				} `json:"rows"`
			}
			res := admin.call(t, "pmm_top_queries", map[string]any{"service_name": name, "search": "customers", "period_from": matrixWindow, "limit": 100})
			res.requireOK(t)
			res.decode(t, &top)
			for _, row := range top.Rows {
				if join.MatchString(row.Fingerprint) {
					found[name] = row.Dimension
					if row.QueryRef != nil {
						found[name] = row.QueryRef.QueryID
					}
					break
				}
			}
			require.NotEmpty(t, found[name], "no pmm_top_queries row of %s joins orders and customers: run the workload of testdata/explain on it", name)
		}
		require.Equal(t, found[e.a], found[e.b], "the join has different queryids on %s and %s", e.a, e.b)
		f.queryIDs[e.engine] = found[e.a]
	}

	for key, args := range map[string]map[string]any{
		"qan_columns": {"kind": "qan_columns"},
		"metrics":     {"kind": "metrics", "engine": "mysql"},
	} {
		res := admin.call(t, "pmm_catalog", args)
		res.requireOK(t)
		f.catalogs[key] = res.structured
	}

	f.alerts = createMatrixAlerts(t, suffix)
	return f
}

func TestAccessRoleMatrix(t *testing.T) {
	if !pmmapitests.RunMCPAccessRoleTests {
		t.Skip("Needs two MySQL services, mysql-a and mysql-b, and two PostgreSQL services, pg-a and pg-b, each pair from the same image " +
			"and initialisation, with the workload of plan 04 Task 7 Step 4 (api-tests/mcp/testdata/shop-postgresql.sql for PostgreSQL), " +
			"and two minutes for QAN. Set -pmm.run-mcp-access-role-tests [PMM_RUN_MCP_ACCESS_ROLE_TESTS]. The test changes server-wide settings.")
	}

	pmmapitests.SetAccessControl(t, true)

	t.Run("ToolsCovered", func(t *testing.T) {
		res, err := connectWith(t, adminAuthorization()).ListTools(pmmapitests.Context, nil)
		require.NoError(t, err)
		names := make([]string, 0, len(res.Tools))
		for _, tool := range res.Tools {
			names = append(names, tool.Name)
		}
		assert.Equal(t, slices.Sorted(maps.Keys(matrixChecks)), sortedUnique(names), "tools/list and the matrix differ: cover the new tool in TestAccessRoleMatrix")
	})

	suffix := time.Now().UnixNano()
	f := newMatrixFixture(t, suffix)
	oneFilter := `{service_name="mysql-a"}`

	setups := []accessSetup{
		{
			name:      "one",
			callers:   restrictedCallers,
			hidesNode: true,
			prepare: func(t *testing.T, suffix int64) []int64 {
				t.Helper()
				return []int64{pmmapitests.CreateRole(t, fmt.Sprintf("mcp-matrix-one-%d", suffix), oneFilter)}
			},
			allowed: func([]string) []string { return []string{"mysql-a"} },
		},
		{
			name:      "two",
			callers:   restrictedCallers,
			hidesNode: true,
			prepare: func(t *testing.T, suffix int64) []int64 {
				t.Helper()
				return []int64{
					pmmapitests.CreateRole(t, fmt.Sprintf("mcp-matrix-two-a-%d", suffix), oneFilter),
					pmmapitests.CreateRole(t, fmt.Sprintf("mcp-matrix-two-b-%d", suffix), `{service_name="pg-a"}`),
				}
			},
			allowed: func([]string) []string { return []string{"mysql-a", "pg-a"} },
		},
		{
			name:    "negative",
			callers: restrictedCallers,
			prepare: func(t *testing.T, suffix int64) []int64 {
				t.Helper()
				return []int64{pmmapitests.CreateRole(t, fmt.Sprintf("mcp-matrix-negative-%d", suffix), `{service_name!~"mysql-b|pg-b"}`)}
			},
			allowed: func(all []string) []string { return without(all, "mysql-b", "pg-b") },
		},
		{
			name:      "default",
			callers:   restrictedCallers,
			hidesNode: true,
			prepare: func(t *testing.T, suffix int64) []int64 {
				t.Helper()
				roleID := pmmapitests.CreateRole(t, fmt.Sprintf("mcp-matrix-default-%d", suffix), oneFilter)
				pmmapitests.SetDefaultRole(t, roleID)
				return nil
			},
			allowed: func([]string) []string { return []string{"mysql-a"} },
		},
		{
			name:    "full",
			callers: []accessCaller{adminCaller},
			prepare: func(t *testing.T, _ int64) []int64 {
				t.Helper()
				res, err := accesscontrolClient.Default.AccessControlService.ListRoles(&accesscontrol.ListRolesParams{Context: t.Context()})
				require.NoError(t, err)
				for _, role := range res.Payload.Roles {
					if role.Title == fullAccessTitle {
						require.Empty(t, role.Filter, "the role %s has a filter", fullAccessTitle)
						return []int64{role.RoleID}
					}
				}
				require.FailNow(t, "no Access Role titled "+fullAccessTitle)
				return nil
			},
			allowed: func(all []string) []string { return all },
		},
		{
			name:    "off",
			callers: restrictedCallers,
			prepare: func(t *testing.T, _ int64) []int64 {
				t.Helper()
				pmmapitests.SetAccessControl(t, false)
				return nil
			},
			allowed: func(all []string) []string { return all },
		},
	}

	for _, s := range setups {
		t.Run(s.name, func(t *testing.T) {
			roleIDs := s.prepare(t, suffix)
			allowed := sortedUnique(s.allowed(f.all))
			hidden := without(f.all, allowed...)

			for _, c := range s.callers {
				t.Run(c.name, func(t *testing.T) {
					login := fmt.Sprintf("mcp-%s-%s-%d", s.name, c.name, suffix)
					userID := pmmapitests.CreateUser(t, login, matrixPassword, c.grafanaRole)
					if len(roleIDs) > 0 {
						pmmapitests.AssignRoles(t, userID, roleIDs...)
					}
					authorization := basicAuthorization(login, matrixPassword)
					waitForServiceList(t, authorization, allowed)

					want := inventoryStatusFor(c)
					assert.Equal(t, want, inventoryStatus(t, authorization), "GET /v1/inventory/services")

					ch := &caller{session: connectWith(t, authorization)}
					if len(hidden) > 0 {
						quoted := make([]string, 0, len(hidden))
						for _, name := range hidden {
							quoted = append(quoted, regexp.QuoteMeta(name))
							ch.hiddenIDs = append(ch.hiddenIDs, f.ids[name], f.mcpIDs[name])
						}
						ch.hiddenIDs = sortedUnique(slices.DeleteFunc(ch.hiddenIDs, func(id string) bool { return id == "" }))
						ch.hiddenRE = regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])(?:` + strings.Join(quoted, "|") + `)(?:$|[^A-Za-z0-9_-])`)
					}
					m := &matrixRun{caller: ch, f: f, allowed: allowed, hidden: hidden, hidesNode: s.hidesNode}
					m.run(t)
				})
			}
		})
	}
}

// inventoryStatusFor is the status GET /v1/inventory/services answers the caller: only an Admin may read it.
func inventoryStatusFor(c accessCaller) int {
	if c.grafanaRole == "Admin" {
		return http.StatusOK
	}
	return http.StatusForbidden
}

// matrixRun checks every tool for one caller under one setup.
type matrixRun struct {
	*caller

	f         *matrixFixture
	allowed   []string
	hidden    []string
	hidesNode bool
}

func (m *matrixRun) isAllowed(name string) bool {
	return slices.Contains(m.allowed, name)
}

// candidates are the services of the engine that the caller may see.
func (m *matrixRun) candidates(e engineServices) []string {
	return intersection(m.allowed, e.a, e.b)
}

func (m *matrixRun) run(t *testing.T) {
	t.Helper()

	for _, name := range slices.Sorted(maps.Keys(matrixChecks)) {
		t.Run(name, func(t *testing.T) { matrixChecks[name](m, t) })
	}
}

func (m *matrixRun) version(t *testing.T) {
	t.Helper()

	res := m.call(t, "pmm_version", nil)
	res.requireOK(t)
	assert.True(t, strings.HasPrefix(res.text, "PMM Server 3."), res.text)
}

// alerts checks that the fixture's server-level alert is shown, that the alert of mysql-a and of mysql-b is shown when the service is
// allowed and not when it is hidden, that no alert names a hidden service, and that a hidden service as a filter answers the 7-day not_found.
func (m *matrixRun) alerts(t *testing.T) {
	t.Helper()

	var out alertsOutput
	res := m.call(t, "pmm_alerts", map[string]any{"limit": alertsMaxLimit})
	res.requireOK(t)
	res.decode(t, &out)
	assert.False(t, out.Page.Truncated, "the alerts do not fit on one page")
	assert.Len(t, out.ofRule(m.f.alerts.server), 1, "an alert that names no service or node is shown")
	ofServices := labelValues(out.ofRule(m.f.alerts.services), "service_name")
	for _, name := range []string{"mysql-a", "mysql-b"} {
		if m.isAllowed(name) {
			assert.Contains(t, ofServices, name, "the alert of a service in the caller's list")
		} else {
			assert.NotContains(t, ofServices, name, "the alert of a hidden service")
		}
	}
	for _, a := range out.Alerts {
		assert.NotContains(t, m.hidden, a.Labels["service_name"], "alert %s", a.Name)
		assert.NotContains(t, m.hiddenIDs, a.Labels["service_id"], "alert %s", a.Name)
	}

	for _, name := range intersection(m.hidden, "mysql-b", "pg-b") {
		res := m.call(t, "pmm_alerts", map[string]any{"service_name": name})
		res.requireNotInList(t, "", alertListLookback)
	}
}

func (m *matrixRun) catalog(t *testing.T) {
	t.Helper()

	for key, args := range map[string]map[string]any{
		"qan_columns": {"kind": "qan_columns"},
		"metrics":     {"kind": "metrics", "engine": "mysql"},
	} {
		res := m.call(t, "pmm_catalog", args)
		res.requireOK(t)
		assert.Equal(t, m.f.catalogs[key], res.structured, key)
	}
}

func (m *matrixRun) inventory(t *testing.T) {
	t.Helper()

	var out inventoryOutput
	res := m.call(t, "pmm_inventory", map[string]any{"limit": 200})
	res.requireOK(t)
	res.decode(t, &out)

	names := make([]string, 0, len(out.Services))
	for _, row := range out.Services {
		names = append(names, row.ServiceName)
	}
	assert.Equal(t, m.allowed, sortedUnique(names))
	assert.Len(t, out.Services, out.Page.Total)

	if m.isAllowed("mysql-a") {
		i := slices.IndexFunc(out.Services, func(row inventoryRow) bool { return row.ServiceName == "mysql-a" })
		require.GreaterOrEqual(t, i, 0)
		assert.NotEmpty(t, out.Services[i].NodeName)
		assert.Equal(t, m.f.nodeNames["mysql-a"], out.Services[i].NodeName)
	}
}

type topQueriesOutput struct {
	Rows []struct {
		Dimension string `json:"dimension"`
	} `json:"rows"`
}

func (m *matrixRun) topQueries(t *testing.T) {
	t.Helper()

	var out topQueriesOutput
	res := m.call(t, "pmm_top_queries", map[string]any{"group_by": "service_name", "period_from": matrixWindow, "limit": 100})
	res.requireOK(t)
	res.decode(t, &out)

	dimensions := make([]string, 0, len(out.Rows))
	for _, row := range out.Rows {
		dimensions = append(dimensions, row.Dimension)
		assert.Contains(t, m.allowed, row.Dimension)
	}
	for _, name := range intersection(m.allowed, "mysql-a", "pg-a") {
		assert.Contains(t, dimensions, name)
	}

	for _, name := range intersection(m.hidden, "mysql-b", "pg-b") {
		res := m.call(t, "pmm_top_queries", map[string]any{"service_name": name, "period_from": matrixWindow, "limit": 100}, "var-service_name="+name)
		res.requireOK(t)
		var hiddenOut topQueriesOutput
		res.decode(t, &hiddenOut)
		assert.Empty(t, hiddenOut.Rows, name)
	}
}

// notSeen is the not_found of pmm_get_explain and pmm_get_schema for a query seen on none of the caller's services, with the window searched.
var notSeen = regexp.MustCompile(`the query was not seen on any of your services in \S+ – \S+`)

// noDataForQuery is the not_found of pmm_query_detail for a query with no data on the caller's services, with the window searched.
var noDataForQuery = regexp.MustCompile(`no data for this query in \S+ – \S+`)

func (m *matrixRun) queryDetail(t *testing.T) {
	t.Helper()

	for _, e := range matrixEngines {
		t.Run(e.engine, func(t *testing.T) {
			candidates := m.candidates(e)
			res := m.call(t, "pmm_query_detail", map[string]any{"queryid": m.f.queryIDs[e.engine], "period_from": matrixWindow})
			switch len(candidates) {
			case 0:
				res.requireError(t, "not_found")
				assert.Regexp(t, noDataForQuery, res.text)
			case 1:
				res.requireOK(t)
				var out struct {
					ServiceName string `json:"service_name"`
					QueryRef    struct {
						ServiceID string `json:"service_id"`
					} `json:"query_ref"`
				}
				res.decode(t, &out)
				assert.Equal(t, candidates[0], out.ServiceName)
				if out.QueryRef.ServiceID != "" {
					assert.Equal(t, m.f.ids[candidates[0]], out.QueryRef.ServiceID)
				}
			default:
				res.requireOK(t)
			}

			if slices.Contains(m.hidden, e.b) {
				res := m.call(t, "pmm_query_detail", map[string]any{"queryid": m.f.queryIDs[e.engine], "service_id": m.f.ids[e.b], "period_from": matrixWindow})
				res.requireNotInList(t, "", 12*time.Hour)
			}
		})
	}
}

// queryForm checks the queryid rows of pmm_get_explain and pmm_get_schema, calling onOne for the result of a single candidate.
func (m *matrixRun) queryForm(t *testing.T, tool string, onOne func(t *testing.T, e engineServices, service string, res toolResponse)) {
	t.Helper()

	for _, e := range matrixEngines {
		t.Run(e.engine, func(t *testing.T) {
			candidates := m.candidates(e)
			res := m.call(t, tool, map[string]any{"queryid": m.f.queryIDs[e.engine], "period_from": matrixWindow})
			switch len(candidates) {
			case 0:
				res.requireError(t, "not_found")
				assert.Regexp(t, notSeen, res.text)
			case 1:
				res.requireOK(t)
				onOne(t, e, candidates[0], res)
			default:
				res.requireError(t, "invalid_input")
				assert.Contains(t, res.text, fmt.Sprintf("%s (%s)", e.a, m.f.ids[e.a]))
				assert.Contains(t, res.text, fmt.Sprintf("%s (%s)", e.b, m.f.ids[e.b]))
			}
		})
	}
}

func (m *matrixRun) explain(t *testing.T) {
	t.Helper()

	m.queryForm(t, "pmm_get_explain", func(t *testing.T, e engineServices, service string, res toolResponse) {
		t.Helper()

		var out struct {
			Engine      string `json:"engine"`
			ServiceName string `json:"service_name"`
			QueryRef    struct {
				ServiceID string `json:"service_id"`
			} `json:"query_ref"`
		}
		res.decode(t, &out)
		assert.Equal(t, e.engine, out.Engine)
		assert.Equal(t, service, out.ServiceName)
		assert.Equal(t, m.f.ids[service], out.QueryRef.ServiceID)
		if e.engine == "mysql" {
			assert.Contains(t, res.text, "EXPLAIN (json, mysql "+service+")")
		} else {
			assert.Contains(t, res.text, "Stored plan (pg_stat_monitor")
		}
	})

	for _, e := range matrixEngines {
		if slices.Contains(m.hidden, e.b) {
			res := m.call(t, "pmm_get_explain", map[string]any{"queryid": m.f.queryIDs[e.engine], "service_id": m.f.ids[e.b], "period_from": matrixWindow})
			res.requireNotInList(t, "", 12*time.Hour)
		}
	}
}

func (m *matrixRun) schema(t *testing.T) {
	t.Helper()

	m.queryForm(t, "pmm_get_schema", func(t *testing.T, e engineServices, service string, res toolResponse) {
		t.Helper()

		prefix := "shop."
		if e.engine == "postgresql" {
			prefix = "shop.public."
		}
		var out struct {
			Tables []struct {
				Database string `json:"database"`
				Name     string `json:"name"`
			} `json:"tables"`
		}
		res.decode(t, &out)
		names := make([]string, 0, len(out.Tables))
		for _, table := range out.Tables {
			names = append(names, table.Database+"."+table.Name)
		}
		slices.Sort(names)
		assert.Equal(t, []string{prefix + "customers", prefix + "orders"}, names)
		assert.Contains(t, res.text, "service: "+service+" (")
		assert.Equal(t, 1, strings.Count(res.text, "### "+prefix+"orders\n"))
		assert.Equal(t, 1, strings.Count(res.text, "### "+prefix+"customers\n"))
	})

	for _, e := range matrixEngines {
		if slices.Contains(m.hidden, e.b) {
			res := m.call(t, "pmm_get_schema", map[string]any{"service_id": m.f.ids[e.b], "database": "shop", "tables": []string{"orders"}})
			res.requireNotInList(t, inventoryHint, time.Hour)
		}
	}
}

func (m *matrixRun) config(t *testing.T) {
	t.Helper()

	for _, name := range []string{"mysql-a", "mysql-b", "pg-a", "pg-b"} {
		for form, args := range map[string]map[string]any{
			"service_id":   {"service_id": m.f.ids[name]},
			"service_name": {"service_name": name},
		} {
			t.Run(name+"-"+form, func(t *testing.T) {
				res := m.call(t, "pmm_get_config", args)
				if !m.isAllowed(name) {
					res.requireNotInList(t, inventoryHint, time.Hour)
					return
				}
				res.requireOK(t)
				var out struct {
					ServiceName string `json:"service_name"`
					Variables   []struct {
						Name string `json:"name"`
					} `json:"variables"`
				}
				res.decode(t, &out)
				assert.Equal(t, name, out.ServiceName)
				assert.NotEmpty(t, out.Variables)
			})
		}
	}
}

func (m *matrixRun) health(t *testing.T) {
	t.Helper()

	for _, name := range []string{"mysql-a", "mysql-b"} {
		t.Run(name, func(t *testing.T) {
			res := m.call(t, "pmm_service_health", map[string]any{"service_id": m.f.ids[name]})
			if !m.isAllowed(name) {
				res.requireNotInList(t, inventoryHint, time.Hour)
				return
			}
			res.requireOK(t)
			var out struct {
				Signals []struct {
					Name   string `json:"name"`
					LastAt string `json:"last_at"`
					NoData bool   `json:"no_data"`
				} `json:"signals"`
				Note string `json:"note"`
			}
			res.decode(t, &out)

			engineWithData := 0
			for _, sig := range out.Signals {
				isNode := slices.Contains(nodeSignals, sig.Name)
				if !isNode && sig.LastAt != "" && !sig.NoData {
					engineWithData++
				}
				if isNode && m.hidesNode {
					assert.True(t, sig.NoData, "node signal %s", sig.Name)
				}
			}
			assert.Positive(t, engineWithData, "no MySQL signal has data")
			if m.hidesNode {
				assert.Contains(t, out.Note, nodeHiddenNote)
			}
		})
	}
}

type metricsOutput struct {
	SharedLabels map[string]string `json:"shared_labels"`
	Series       []struct {
		Labels map[string]string `json:"labels"`
	} `json:"series"`
	TotalSeries int `json:"total_series"`
}

func (m *matrixRun) queryMetrics(t *testing.T) {
	t.Helper()

	var out metricsOutput
	res := m.call(t, "pmm_query_metrics", map[string]any{"query": "count by (service_name) (up)", "limit": 100})
	res.requireOK(t)
	res.decode(t, &out)
	var names []string
	if name := out.SharedLabels["service_name"]; name != "" {
		names = append(names, name)
	}
	for _, series := range out.Series {
		if name := series.Labels["service_name"]; name != "" {
			names = append(names, name)
		}
	}
	assert.Equal(t, m.allowed, sortedUnique(names))

	if slices.Contains(m.hidden, "mysql-b") {
		query := `mysql_up{service_name="mysql-b"}`
		res := m.call(t, "pmm_query_metrics", map[string]any{"query": query}, query)
		res.requireOK(t)
		var hiddenOut metricsOutput
		res.decode(t, &hiddenOut)
		assert.Zero(t, hiddenOut.TotalSeries)
	}
}

func (m *matrixRun) listMetrics(t *testing.T) {
	t.Helper()

	var out struct {
		Values []string `json:"values"`
	}
	res := m.call(t, "pmm_list_metrics", map[string]any{"kind": "values", "label": "service_name", "limit": 1000})
	res.requireOK(t)
	res.decode(t, &out)
	assert.Equal(t, m.allowed, sortedUnique(out.Values))
	assert.Len(t, out.Values, len(sortedUnique(out.Values)))

	if slices.Contains(m.hidden, "mysql-b") {
		selector := `mysql_up{service_name="mysql-b"}`
		res := m.call(t, "pmm_list_metrics", map[string]any{"kind": "series", "match": []string{selector}}, selector)
		res.requireOK(t)
		var seriesOut struct {
			Series []map[string]string `json:"series"`
		}
		res.decode(t, &seriesOut)
		assert.Empty(t, seriesOut.Series)
	}
}
