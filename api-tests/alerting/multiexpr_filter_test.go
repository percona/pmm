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

package alerting

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/prometheus/client_golang/api"
	promapi "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	alertingClient "github.com/percona/pmm/api/alerting/v1/json/client"
	alerting "github.com/percona/pmm/api/alerting/v1/json/client/alerting_service"
)

// basicAuthRoundTripper adds PMM Server basic auth to VictoriaMetrics API requests.
type basicAuthRoundTripper struct {
	user, pass string
	base       http.RoundTripper
}

func (rt basicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.SetBasicAuth(rt.user, rt.pass)
	return rt.base.RoundTrip(req)
}

// TestMultiExpressionFilterDoesNotEmptyConstantQuery reproduces the bug where a rule
// filter is applied to every query of a multi-expression template, including a constant
// query such as `vector([[ .threshold ]])`. A MATCH filter on a label the constant cannot
// carry (node_name) turns the query into `label_match(vector(80), "node_name", "db.*")`,
// which evaluates to an EMPTY vector in VictoriaMetrics. The math condition ($A < $B) then
// loses its threshold operand and the alert can never fire.
func TestMultiExpressionFilterDoesNotEmptyConstantQuery(t *testing.T) {
	client := alertingClient.Default.AlertingService

	matchType := "FILTER_TYPE_MATCH"
	floatType := "PARAM_TYPE_FLOAT"
	severity := "SEVERITY_WARNING"
	forceDelete := true

	// 1. A multi-expression template: query A = a real metric, query B = a constant threshold,
	//    math C = "$A < $B", condition C.
	templateName := pmmapitests.TestString(t, "test-filter-constant-query")
	yml := fmt.Sprintf(`templates:
  - name: %s
    version: 1
    summary: Filter empties constant query (bug repro)
    queries:
      - ref_id: A
        expr: node_cpu_seconds_total
      - ref_id: B
        expr: vector([[ .threshold ]])
    expressions:
      - ref_id: C
        type: math
        expression: "$A < $B"
    condition: C
    params:
      - name: threshold
        summary: A percentage from configured maximum
        type: float
        range: [0, 100]
        value: 80
    for: 60s
    severity: warning
    annotations:
      summary: repro
`, templateName)

	_, err := client.CreateTemplate(&alerting.CreateTemplateParams{
		Body:    alerting.CreateTemplateBody{Yaml: yml},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	t.Cleanup(func() { deleteTemplate(t, client, templateName) })

	// 2. Grafana folder for the rule.
	gClient := pmmapitests.GetGrafanaClient(t)
	createdFolder, err := gClient.Folders.CreateFolder(&models.CreateFolderCommand{Title: pmmapitests.TestString(t, "test-filter-folder")})
	require.NoError(t, err)
	folder := createdFolder.Payload
	t.Cleanup(func() {
		_, _ = gClient.Folders.DeleteFolder(folders.NewDeleteFolderParams().WithFolderUID(folder.UID).WithForceDeleteRules(&forceDelete))
	})

	// 3. Create a rule with a MATCH filter on node_name - a label the constant query B cannot
	//    carry. This exercises the real server-side buildMultiExpressionRuleData.
	ruleName := pmmapitests.TestString(t, "test-filter-rule")
	const group = "test"
	_, err = client.CreateRule(&alerting.CreateRuleParams{
		Body: alerting.CreateRuleBody{
			TemplateName: templateName,
			Name:         ruleName,
			FolderUID:    folder.UID,
			Group:        group,
			Interval:     "10s",
			For:          "60s",
			Severity:     &severity,
			Params: []*alerting.CreateRuleParamsBodyParamsItems0{
				{Name: "threshold", Type: &floatType, Float: 80},
			},
			Filters: []*alerting.CreateRuleParamsBodyFiltersItems0{
				{Type: &matchType, Label: "node_name", Regexp: "db.*"},
			},
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)

	// 4. Read back the server-built query for leg B from Grafana.
	grp, err := gClient.Provisioning.GetAlertRuleGroup(group, folder.UID)
	require.NoError(t, err)

	var exprB string
	for _, rule := range grp.Payload.Rules {
		if rule.Title == nil || *rule.Title != ruleName {
			continue
		}
		for _, q := range rule.Data {
			if q.RefID != "B" {
				continue
			}
			model, ok := q.Model.(map[string]any)
			require.Truef(t, ok, "query B model is not an object: %T", q.Model)
			exprB, _ = model["expr"].(string)
		}
	}
	require.NotEmpty(t, exprB, "could not find the server-built expression for query B")
	t.Logf("server-built query B: %s", exprB)

	// 5. Evaluate the built query B against VictoriaMetrics (through the PMM Server proxy,
	//    which requires basic auth and may serve a self-signed cert).
	var rt http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: pmmapitests.ServerInsecureTLS}, //nolint:gosec
	}
	if u := pmmapitests.BaseURL.User; u != nil {
		pass, _ := u.Password()
		rt = basicAuthRoundTripper{user: u.Username(), pass: pass, base: rt}
	}
	promHTTP, err := api.NewClient(api.Config{
		Address:      pmmapitests.BaseURL.ResolveReference(&url.URL{Path: "/prometheus"}).String(),
		RoundTripper: rt,
	})
	require.NoError(t, err)
	promClient := promapi.NewAPI(promHTTP)

	result, _, err := promClient.Query(t.Context(), exprB, time.Now())
	require.NoError(t, err)
	vec, ok := result.(prommodel.Vector)
	require.Truef(t, ok, "expected a vector result, got %s", result.Type())

	// The filtered constant threshold query must still return its value.
	assert.NotEmptyf(t, vec, "filtered constant query %q evaluated to an empty vector: the threshold operand collapses and the alert can never fire", exprB)
}
