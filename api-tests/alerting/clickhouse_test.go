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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	alertingClient "github.com/percona/pmm/api/alerting/v1/json/client"
	alerting "github.com/percona/pmm/api/alerting/v1/json/client/alerting_service"
)

// TestClickHouseTemplate checks that a template query on ClickHouse becomes a rule query on the
// ClickHouse datasource, and that Grafana's ClickHouse plugin accepts the model PMM builds for it.
func TestClickHouseTemplate(t *testing.T) {
	client := alertingClient.Default.AlertingService
	gClient := pmmapitests.GetGrafanaClient(t)

	templateName := pmmapitests.TestString(t, "test-clickhouse-template")
	yml := fmt.Sprintf(`templates:
  - name: %s
    version: 1
    summary: ClickHouse template
    queries:
      - ref_id: A
        datasource: clickhouse
        expr: |-
          SELECT $__toTime AS time, service_name, node_name, max(m_query_time_max) AS query_time
          FROM pmm.metrics
          WHERE period_start >= $__fromTime AND period_start <= $__toTime
          GROUP BY service_name, node_name
    expressions:
      - ref_id: B
        type: reduce
        reducer: max
        expression: $A
      - ref_id: C
        type: math
        expression: "$B > [[ .threshold ]]"
    condition: C
    params:
      - name: threshold
        summary: Query time
        unit: s
        type: float
        value: 10
    for: 60s
    severity: warning
    annotations:
      summary: Slow queries on {{ $labels.service_name }}
`, templateName)
	_, err := client.CreateTemplate(&alerting.CreateTemplateParams{
		Body:    alerting.CreateTemplateBody{Yaml: yml},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	t.Cleanup(func() { deleteTemplate(t, client, templateName) })

	listed, err := client.ListTemplates(&alerting.ListTemplatesParams{Reload: new(true), Context: pmmapitests.Context})
	require.NoError(t, err)
	idx := slices.IndexFunc(listed.Payload.Templates, func(tmpl *alerting.ListTemplatesOKBodyTemplatesItems0) bool {
		return tmpl.Name == templateName
	})
	require.GreaterOrEqual(t, idx, 0, "template %s is not listed", templateName)
	found := listed.Payload.Templates[idx]
	require.Len(t, found.Queries, 1)
	assert.Equal(t, "DATASOURCE_TYPE_CLICKHOUSE", *found.Queries[0].Datasource)
	require.Len(t, found.Expressions, 2)
	assert.Equal(t, "max", found.Expressions[0].Reducer)

	clickHouseDatasource, err := gClient.Datasources.GetDataSourceByName("ClickHouse")
	require.NoError(t, err)
	clickHouseUID := clickHouseDatasource.Payload.UID

	createdFolder, err := gClient.Folders.CreateFolder(&models.CreateFolderCommand{Title: pmmapitests.TestString(t, "test-clickhouse-folder")})
	require.NoError(t, err)
	folder := createdFolder.Payload
	t.Cleanup(func() {
		_, _ = gClient.Folders.DeleteFolder(folders.NewDeleteFolderParams().WithFolderUID(folder.UID).WithForceDeleteRules(new(true)))
	})

	createRule := func(t *testing.T, templateName, group string, filters []*alerting.CreateRuleParamsBodyFiltersItems0) []*models.AlertQuery {
		t.Helper()

		ruleName := pmmapitests.TestString(t, "test-clickhouse-rule")
		_, err := client.CreateRule(&alerting.CreateRuleParams{
			Body: alerting.CreateRuleBody{
				TemplateName: templateName,
				Name:         ruleName,
				FolderUID:    folder.UID,
				Group:        group,
				Interval:     "60s",
				For:          "60s",
				Severity:     new("SEVERITY_WARNING"),
				Params:       []*alerting.CreateRuleParamsBodyParamsItems0{{Name: "threshold", Type: new("PARAM_TYPE_FLOAT"), Float: 5}},
				Filters:      filters,
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		grp, err := gClient.Provisioning.GetAlertRuleGroup(group, folder.UID)
		require.NoError(t, err)
		for _, rule := range grp.Payload.Rules {
			if rule.Title != nil && *rule.Title == ruleName {
				return rule.Data
			}
		}
		require.FailNow(t, "rule not found in Grafana", ruleName)
		return nil
	}

	t.Run("rule queries ClickHouse and evaluates", func(t *testing.T) {
		data := createRule(t, templateName, "test-clickhouse", nil)

		query := queryByRefID(t, data, "A")
		assert.Equal(t, clickHouseUID, query.DatasourceUID)
		assert.Equal(t, "timeseries", query.QueryType)
		assertEvaluates(t, data)
	})

	t.Run("filters wrap the SQL and still evaluate", func(t *testing.T) {
		data := createRule(t, templateName, "test-clickhouse-filtered", []*alerting.CreateRuleParamsBodyFiltersItems0{
			{Type: new("FILTER_TYPE_MATCH"), Label: "service_name", Regexp: "pmm-server.*"},
		})

		model, ok := queryByRefID(t, data, "A").Model.(map[string]any)
		require.True(t, ok)
		assert.Contains(t, model["rawSql"], "WHERE match(ifNull(toString(`service_name`), ''), '^(?:(?:pmm-server.*)|)$')")
		assertEvaluates(t, data)
	})

	t.Run("built-in pmm_qan_slow_queries evaluates with its threshold step", func(t *testing.T) {
		data := createRule(t, "pmm_qan_slow_queries", "test-qan-slow-queries", nil)

		assert.Equal(t, clickHouseUID, queryByRefID(t, data, "A").DatasourceUID)
		// The per-node threshold step reads its overrides from VictoriaMetrics.
		assert.NotEqual(t, clickHouseUID, queryByRefID(t, data, "T_threshold").DatasourceUID)
		assertEvaluates(t, data)
	})
}

func queryByRefID(t *testing.T, data []*models.AlertQuery, refID string) *models.AlertQuery {
	t.Helper()

	for _, query := range data {
		if query.RefID == refID {
			return query
		}
	}
	require.FailNow(t, "rule has no query", refID)
	return nil
}

// assertEvaluates runs the rule's steps through Grafana's alert evaluation, which fails a step
// whose datasource rejects its query model.
func assertEvaluates(t *testing.T, data []*models.AlertQuery) {
	t.Helper()

	body, err := json.Marshal(map[string]any{"data": data})
	require.NoError(t, err)

	evalURL := pmmapitests.BaseURL.ResolveReference(&url.URL{Path: "/graph/api/v1/eval"})
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, evalURL.String(), bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if u := pmmapitests.BaseURL.User; u != nil {
		pass, _ := u.Password()
		req.SetBasicAuth(u.Username(), pass)
	}

	resp, err := pmmapitests.HTTPClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, resp.StatusCode, "eval failed: %s", respBody)

	var result struct {
		Results map[string]struct {
			Error string `json:"error"`
		} `json:"results"`
	}
	require.NoError(t, json.Unmarshal(respBody, &result))
	for refID, res := range result.Results {
		assert.Emptyf(t, res.Error, "step %s failed to evaluate", refID)
	}
}
