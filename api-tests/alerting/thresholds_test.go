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
	"fmt"
	"net/http"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	alertingClient "github.com/percona/pmm/api/alerting/v1/json/client"
	alerting "github.com/percona/pmm/api/alerting/v1/json/client/alerting_service"
)

const (
	scopeNode    = "THRESHOLD_SCOPE_NODE"
	scopeService = "THRESHOLD_SCOPE_SERVICE"
	scopeCluster = "THRESHOLD_SCOPE_CLUSTER"
)

// thresholdFixture is a rule created from an overridable template, plus the node its
// thresholds are set against.
type thresholdFixture struct {
	client alerting.ClientService
	ruleID string
	nodeID string
}

// setupThresholdFixture registers an overridable template, creates a rule from it and a
// node to target, and cleans all three up afterwards.
func setupThresholdFixture(t *testing.T) *thresholdFixture {
	t.Helper()

	client := alertingClient.Default.AlertingService

	floatType := "PARAM_TYPE_FLOAT"
	severity := "SEVERITY_WARNING"
	forceDelete := true

	templateName := pmmapitests.TestString(t, "test-threshold-template")
	yml := fmt.Sprintf(`templates:
  - name: %s
    version: 1
    summary: Overridable threshold
    queries:
      - ref_id: A
        expr: |-
          (1 - avg by(node_name) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100
    expressions:
      - ref_id: C
        type: math
        expression: "$A > [[ .threshold ]]"
    condition: C
    params:
      - name: threshold
        summary: A percentage from configured maximum
        unit: "%%"
        type: float
        range: [0, 100]
        value: 80
        overridable: true
    for: 60s
    severity: warning
    annotations:
      summary: overridable threshold
`, templateName)

	_, err := client.CreateTemplate(&alerting.CreateTemplateParams{
		Body:    alerting.CreateTemplateBody{Yaml: yml},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	t.Cleanup(func() { deleteTemplate(t, client, templateName) })

	gClient := pmmapitests.GetGrafanaClient(t)
	createdFolder, err := gClient.Folders.CreateFolder(&models.CreateFolderCommand{
		Title: pmmapitests.TestString(t, "test-threshold-folder"),
	})
	require.NoError(t, err)
	folder := createdFolder.Payload
	t.Cleanup(func() {
		_, _ = gClient.Folders.DeleteFolder(
			folders.NewDeleteFolderParams().WithFolderUID(folder.UID).WithForceDeleteRules(&forceDelete))
	})

	created, err := client.CreateRule(&alerting.CreateRuleParams{
		Body: alerting.CreateRuleBody{
			TemplateName: templateName,
			Name:         pmmapitests.TestString(t, "test-threshold-rule"),
			FolderUID:    folder.UID,
			Group:        "test",
			Interval:     "10s",
			For:          "60s",
			Severity:     &severity,
			Params: []*alerting.CreateRuleParamsBodyParamsItems0{
				{Name: "threshold", Type: &floatType, Float: 80},
			},
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)

	// A rule built from an overridable template must come back with an identity to key
	// its overrides on; without one the whole feature is unreachable.
	require.NotEmpty(t, created.Payload.RuleID,
		"CreateRule must return a rule_id for an overridable template")

	node := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "test-threshold-node"))
	t.Cleanup(func() { pmmapitests.RemoveNodes(t, node.NodeID) })

	return &thresholdFixture{
		client: client,
		ruleID: created.Payload.RuleID,
		nodeID: node.NodeID,
	}
}

func (f *thresholdFixture) list(t *testing.T) []*alerting.ListThresholdsOKBodyThresholdsItems0 {
	t.Helper()

	res, err := f.client.ListThresholds(&alerting.ListThresholdsParams{
		Scope:   pointer.ToString(scopeNode),
		Target:  pointer.ToString(f.nodeID),
		RuleID:  pointer.ToString(f.ruleID),
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)

	return res.Payload.Thresholds
}

func (f *thresholdFixture) set(t *testing.T, value float64) (*alerting.SetThresholdOK, error) {
	t.Helper()

	return f.client.SetThreshold(&alerting.SetThresholdParams{
		Body: alerting.SetThresholdBody{
			Scope:     pointer.ToString(scopeNode),
			Target:    f.nodeID,
			RuleID:    f.ruleID,
			ParamName: "threshold",
			Value:     value,
		},
		Context: pmmapitests.Context,
	})
}

func TestThresholdOverrideLifecycle(t *testing.T) {
	t.Parallel()

	f := setupThresholdFixture(t)

	// An untouched target still reports the parameter, at the rule's default.
	before := f.list(t)
	require.Len(t, before, 1)
	assert.InDelta(t, 80, before[0].DefaultValue, 0.0001)
	assert.InDelta(t, 80, before[0].EffectiveValue, 0.0001)
	assert.False(t, before[0].IsOverridden)

	set, err := f.set(t, 95)
	require.NoError(t, err)
	assert.InDelta(t, 95, set.Payload.Threshold.EffectiveValue, 0.0001)
	assert.True(t, set.Payload.Threshold.IsOverridden)
	require.NotNil(t, set.Payload.Threshold.Scope)
	assert.Equal(t, scopeNode, *set.Payload.Threshold.Scope)

	after := f.list(t)
	require.Len(t, after, 1)
	assert.InDelta(t, 95, after[0].EffectiveValue, 0.0001)
	assert.True(t, after[0].IsOverridden)

	// Clearing returns the target to the default. The override row survives as a
	// tombstone so the emitted series keeps existing and merely changes value, but that
	// is invisible from here - what the API must report is "not overridden".
	_, err = f.client.ClearThreshold(&alerting.ClearThresholdParams{
		Scope:     pointer.ToString(scopeNode),
		Target:    pointer.ToString(f.nodeID),
		RuleID:    pointer.ToString(f.ruleID),
		ParamName: pointer.ToString("threshold"),
		Context:   pmmapitests.Context,
	})
	require.NoError(t, err)

	cleared := f.list(t)
	require.Len(t, cleared, 1)
	assert.InDelta(t, 80, cleared[0].EffectiveValue, 0.0001)
	assert.False(t, cleared[0].IsOverridden,
		"a cleared override must not read as overridden, or every target ever tuned reads as tuned forever")
}

func TestThresholdOverrideValidation(t *testing.T) {
	t.Parallel()

	f := setupThresholdFixture(t)

	t.Run("value outside the declared range", func(t *testing.T) {
		_, err := f.set(t, 150)
		pmmapitests.AssertAPIErrorf(t, err, http.StatusBadRequest, codes.InvalidArgument, "")
	})

	t.Run("unknown parameter", func(t *testing.T) {
		_, err := f.client.SetThreshold(&alerting.SetThresholdParams{
			Body: alerting.SetThresholdBody{
				Scope: pointer.ToString(scopeNode), Target: f.nodeID,
				RuleID: f.ruleID, ParamName: "not-overridable", Value: 90,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, http.StatusNotFound, codes.NotFound, "")
	})

	t.Run("unknown rule", func(t *testing.T) {
		_, err := f.client.SetThreshold(&alerting.SetThresholdParams{
			Body: alerting.SetThresholdBody{
				Scope: pointer.ToString(scopeNode), Target: f.nodeID,
				RuleID: "no-such-rule", ParamName: "threshold", Value: 90,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, http.StatusNotFound, codes.NotFound, "")
	})

	t.Run("target that does not exist", func(t *testing.T) {
		_, err := f.client.SetThreshold(&alerting.SetThresholdParams{
			Body: alerting.SetThresholdBody{
				Scope: pointer.ToString(scopeNode), Target: "no-such-node",
				RuleID: f.ruleID, ParamName: "threshold", Value: 90,
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, http.StatusNotFound, codes.NotFound, "")
	})

	// Service and cluster are already carried by the schema, the resolver and the proto,
	// so they report as not-yet-implemented rather than as a malformed request.
	t.Run("scopes that are not implemented yet", func(t *testing.T) {
		for _, scope := range []string{scopeService, scopeCluster} {
			_, err := f.client.SetThreshold(&alerting.SetThresholdParams{
				Body: alerting.SetThresholdBody{
					Scope: pointer.ToString(scope), Target: f.nodeID,
					RuleID: f.ruleID, ParamName: "threshold", Value: 90,
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, http.StatusNotImplemented, codes.Unimplemented, "")
		}
	})
}

func TestThresholdBatchUpdate(t *testing.T) {
	t.Parallel()

	f := setupThresholdFixture(t)

	t.Run("sets through the batch endpoint", func(t *testing.T) {
		res, err := f.client.BatchUpdateThresholds(&alerting.BatchUpdateThresholdsParams{
			Body: alerting.BatchUpdateThresholdsBody{
				Updates: []*alerting.BatchUpdateThresholdsParamsBodyUpdatesItems0{{
					Scope: pointer.ToString(scopeNode), Target: f.nodeID,
					RuleID: f.ruleID, ParamName: "threshold",
					Value: pointer.ToFloat64(70),
				}},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.Len(t, res.Payload.Thresholds, 1)
		assert.InDelta(t, 70, res.Payload.Thresholds[0].EffectiveValue, 0.0001)
	})

	t.Run("an update with no value clears instead of setting", func(t *testing.T) {
		res, err := f.client.BatchUpdateThresholds(&alerting.BatchUpdateThresholdsParams{
			Body: alerting.BatchUpdateThresholdsBody{
				Updates: []*alerting.BatchUpdateThresholdsParamsBodyUpdatesItems0{{
					Scope: pointer.ToString(scopeNode), Target: f.nodeID,
					RuleID: f.ruleID, ParamName: "threshold",
				}},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Empty(t, res.Payload.Thresholds, "cleared entries are omitted from the response")

		current := f.list(t)
		require.Len(t, current, 1)
		assert.False(t, current[0].IsOverridden)
	})

	// The reason the batch endpoint exists: a client editing several rows at once must
	// never land a partial result it cannot report.
	t.Run("one invalid update rolls the whole batch back", func(t *testing.T) {
		_, err := f.client.BatchUpdateThresholds(&alerting.BatchUpdateThresholdsParams{
			Body: alerting.BatchUpdateThresholdsBody{
				Updates: []*alerting.BatchUpdateThresholdsParamsBodyUpdatesItems0{
					{
						Scope: pointer.ToString(scopeNode), Target: f.nodeID,
						RuleID: f.ruleID, ParamName: "threshold",
						Value: pointer.ToFloat64(60),
					},
					{
						Scope: pointer.ToString(scopeNode), Target: f.nodeID,
						RuleID: f.ruleID, ParamName: "threshold",
						Value: pointer.ToFloat64(500), // outside the declared range
					},
				},
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, http.StatusBadRequest, codes.InvalidArgument, "")

		current := f.list(t)
		require.Len(t, current, 1)
		assert.False(t, current[0].IsOverridden,
			"the valid update must not survive the invalid one failing")
	})
}

// TestThresholdOverrideRemovedWithNode covers the cleanup that has no cascade to rely on:
// the override's target column is polymorphic, so it carries no foreign key and the rows
// are removed by the node removal API itself.
func TestThresholdOverrideRemovedWithNode(t *testing.T) {
	t.Parallel()

	f := setupThresholdFixture(t)

	_, err := f.set(t, 90)
	require.NoError(t, err)

	pmmapitests.RemoveNodes(t, f.nodeID)

	// With the node gone the override is unreachable by target, so ask for every
	// override of this rule instead.
	res, err := f.client.ListThresholds(&alerting.ListThresholdsParams{
		RuleID:  pointer.ToString(f.ruleID),
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	assert.Empty(t, res.Payload.Thresholds, "an override must not outlive the node it targets")
}
