// Copyright (C) 2024 Percona LLC
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

package ia

import (
	"fmt"
	"os"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	alertingClient "github.com/percona/pmm/api/managementpb/alerting/json/client"
	"github.com/percona/pmm/api/managementpb/alerting/json/client/alerting"
	"github.com/percona/pmm/api/managementpb/ia/json/client"
	"github.com/percona/pmm/api/managementpb/ia/json/client/channels"
	"github.com/percona/pmm/api/managementpb/ia/json/client/rules"
)

// Note: Even though the IA services check for alerting enabled or disabled before returning results
// we don't enable or disable IA explicit in our tests since it is enabled by default through
// ENABLE_ALERTING env var.
func TestRulesAPI(t *testing.T) {
	t.Parallel()
	rulesClient := client.Default.Rules
	templatesClient := alertingClient.Default.Alerting
	channelsClient := client.Default.Channels

	dummyFilter := &rules.CreateAlertRuleParamsBodyFiltersItems0{
		Type:  pointer.ToString("EQUAL"),
		Key:   "threshold",
		Value: "12",
	}

	templateName := createTemplate(t)
	channelID, _ := createChannel(t)
	newChannelID, _ := createChannel(t)
	t.Cleanup(func() {
		deleteTemplate(t, alertingClient.Default.Alerting, templateName)
		deleteChannel(t, channelsClient, channelID)
		deleteChannel(t, channelsClient, newChannelID)
	})

	t.Run("add", func(t *testing.T) {
		t.Parallel()

		t.Run("normal from template", func(t *testing.T) {
			t.Parallel()

			params := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			rule, err := rulesClient.CreateAlertRule(params)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, rule.Payload.RuleID)

			assert.NotEmpty(t, rule.Payload.RuleID)
		})

		t.Run("without channels and filters", func(t *testing.T) {
			t.Parallel()

			params := createAlertRuleParams(templateName, "", "", nil)
			rule, err := rulesClient.CreateAlertRule(params)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, rule.Payload.RuleID)

			assert.NotEmpty(t, rule.Payload.RuleID)
		})

		t.Run("builtin_template", func(t *testing.T) {
			t.Parallel()

			params := createAlertRuleParams("pmm_mongodb_restarted", "", channelID, dummyFilter)
			params.Body.Params = []*rules.CreateAlertRuleParamsBodyParamsItems0{{
				Name:  "threshold",
				Type:  pointer.ToString("FLOAT"),
				Float: 3.14,
			}}
			rule, err := rulesClient.CreateAlertRule(params)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, rule.Payload.RuleID)

			assert.NotEmpty(t, rule.Payload.RuleID)
		})

		t.Run("use default value for parameter", func(t *testing.T) {
			t.Parallel()

			params := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			rule, err := rulesClient.CreateAlertRule(params)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, rule.Payload.RuleID)

			assert.NotEmpty(t, rule.Payload.RuleID)
		})

		t.Run("normal from other rule", func(t *testing.T) {
			t.Parallel()

			sourceRuleParams := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			sourceRule, err := rulesClient.CreateAlertRule(sourceRuleParams)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, sourceRule.Payload.RuleID)

			copyRuleParams := createAlertRuleParams("", sourceRule.Payload.RuleID, channelID, dummyFilter)
			copyRule, err := rulesClient.CreateAlertRule(copyRuleParams)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, copyRule.Payload.RuleID)

			assert.NotEmpty(t, copyRule.Payload.RuleID)
		})

		t.Run("normal from other rule with deleted template", func(t *testing.T) {
			t.Parallel()

			sourceTemplateName := createTemplate(t)
			sourceRuleParams := createAlertRuleParams(sourceTemplateName, "", channelID, dummyFilter)
			sourceRule, err := rulesClient.CreateAlertRule(sourceRuleParams)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, sourceRule.Payload.RuleID)

			deleteTemplate(t, templatesClient, sourceTemplateName)

			copyRuleParams := createAlertRuleParams("", sourceRule.Payload.RuleID, channelID, dummyFilter)
			copyRule, err := rulesClient.CreateAlertRule(copyRuleParams)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, copyRule.Payload.RuleID)

			assert.NotEmpty(t, copyRule.Payload.RuleID)
		})

		t.Run("both template name and source rule id specified", func(t *testing.T) {
			t.Parallel()

			sourceRuleParams := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			sourceRule, err := rulesClient.CreateAlertRule(sourceRuleParams)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, sourceRule.Payload.RuleID)

			copyRuleParams := createAlertRuleParams(templateName, sourceRule.Payload.RuleID, channelID, dummyFilter)
			_, err = rulesClient.CreateAlertRule(copyRuleParams)
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Both template name and source rule id are specified.")
		})

		t.Run("both template name and source rule id are empty", func(t *testing.T) {
			t.Parallel()

			sourceRuleParams := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			sourceRule, err := rulesClient.CreateAlertRule(sourceRuleParams)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, sourceRule.Payload.RuleID)

			copyRuleParams := createAlertRuleParams("", "", channelID, dummyFilter)
			_, err = rulesClient.CreateAlertRule(copyRuleParams)
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Template name or source rule id should be specified.")
		})

		t.Run("unknown template", func(t *testing.T) {
			t.Parallel()

			templateName := uuid.New().String()
			params := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			_, err := rulesClient.CreateAlertRule(params)
			pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Unknown template %s.", templateName)
		})

		t.Run("unknown channel", func(t *testing.T) {
			t.Parallel()

			channelID := uuid.New().String()
			params := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			_, err := rulesClient.CreateAlertRule(params)
			pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Failed to find all required channels: [%s].", channelID)
		})

		t.Run("wrong parameter", func(t *testing.T) {
			t.Parallel()

			params := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			params.Body.Params = append(
				params.Body.Params,
				&rules.CreateAlertRuleParamsBodyParamsItems0{
					Name:  "unknown parameter",
					Type:  pointer.ToString("FLOAT"),
					Float: 12,
				})
			_, err := rulesClient.CreateAlertRule(params)
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Expression requires 2 parameters, but got 3.")
		})

		t.Run("wrong parameter type", func(t *testing.T) {
			t.Parallel()

			params := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			params.Body.Params = []*rules.CreateAlertRuleParamsBodyParamsItems0{
				{
					Name: "param1",
					Type: pointer.ToString("BOOL"),
					Bool: true,
				}, {
					Name:  "param2",
					Type:  pointer.ToString("FLOAT"),
					Float: 12,
				},
			}
			_, err := rulesClient.CreateAlertRule(params)
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Parameter param1 has type bool instead of float.")
		})
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()

			cParams := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			rule, err := rulesClient.CreateAlertRule(cParams)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, rule.Payload.RuleID)

			params := &rules.UpdateAlertRuleParams{
				Body: rules.UpdateAlertRuleBody{
					RuleID:   rule.Payload.RuleID,
					Disabled: false,
					Params: []*rules.UpdateAlertRuleParamsBodyParamsItems0{
						{
							Name:  "param1",
							Type:  pointer.ToString("FLOAT"),
							Float: 3.14,
						},
						{
							Name:  "param2",
							Type:  pointer.ToString("FLOAT"),
							Float: 21,
						},
					},
					For:          "10s",
					Severity:     pointer.ToString("SEVERITY_ERROR"),
					CustomLabels: map[string]string{"foo": "bar", "baz": "faz"},
					Filters: []*rules.UpdateAlertRuleParamsBodyFiltersItems0{{
						Type:  pointer.ToString("EQUAL"),
						Key:   "param1",
						Value: "21",
					}},
					ChannelIds: []string{channelID, newChannelID},
				},
				Context: pmmapitests.Context,
			}
			_, err = rulesClient.UpdateAlertRule(params)
			require.NoError(t, err)

			list, err := rulesClient.ListAlertRules(&rules.ListAlertRulesParams{Context: pmmapitests.Context})
			require.NoError(t, err)

			var found bool
			for _, r := range list.Payload.Rules {
				if r.RuleID == rule.Payload.RuleID {
					assert.Equal(t, params.Body.Name, r.Name)
					assert.Equal(t, "Test summary", r.Summary)
					assert.Equal(t, templateName, r.TemplateName)
					assert.False(t, r.Disabled)
					assert.Equal(t, "300s", r.DefaultFor)
					assert.Equal(t, "10s", r.For)
					assert.Equal(t, "SEVERITY_WARNING", pointer.GetString(r.DefaultSeverity))
					assert.Equal(t, params.Body.Severity, r.Severity)
					assert.Equal(t, params.Body.CustomLabels, r.CustomLabels)
					assert.Equal(t, map[string]string{"foo": "bar"}, r.Labels)
					assert.Equal(t, map[string]string{"description": "test description", "summary": "test summary"}, r.Annotations)
					assert.Len(t, r.ParamsValues, 2)
					assert.Equal(t, params.Body.Params[0].Type, r.ParamsValues[0].Type)
					assert.Equal(t, params.Body.Params[0].Name, r.ParamsValues[0].Name)
					assert.Equal(t, params.Body.Params[0].Float, r.ParamsValues[0].Float)
					assert.Equal(t, params.Body.Params[0].Bool, r.ParamsValues[0].Bool)
					assert.Equal(t, params.Body.Params[0].String, r.ParamsValues[0].String)
					assert.Equal(t, params.Body.Params[1].Type, r.ParamsValues[1].Type)
					assert.Equal(t, params.Body.Params[1].Name, r.ParamsValues[1].Name)
					assert.Equal(t, params.Body.Params[1].Float, r.ParamsValues[1].Float)
					assert.Equal(t, params.Body.Params[1].Bool, r.ParamsValues[1].Bool)
					assert.Equal(t, params.Body.Params[1].String, r.ParamsValues[1].String)
					assert.Equal(t, "[[ .param1 ]] > 2 and 2 < [[ .param2 ]]", r.ExprTemplate)
					assert.Equal(t, "3.14 > 2 and 2 < 21", r.Expr)
					found = true
				}
			}
			assert.Truef(t, found, "Rule with id %s not found", rule.Payload.RuleID)
		})

		t.Run("unknown channel", func(t *testing.T) {
			t.Parallel()

			cParams := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			rule, err := rulesClient.CreateAlertRule(cParams)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, rule.Payload.RuleID)

			unknownChannelID := uuid.New().String()
			params := &rules.UpdateAlertRuleParams{
				Body: rules.UpdateAlertRuleBody{
					RuleID:   rule.Payload.RuleID,
					Disabled: false,
					Params: []*rules.UpdateAlertRuleParamsBodyParamsItems0{
						{
							Name:  "param1",
							Type:  pointer.ToString("FLOAT"),
							Float: 3.14,
						}, {
							Name:  "param2",
							Type:  pointer.ToString("FLOAT"),
							Float: 21,
						},
					},
					For:          "10s",
					Severity:     pointer.ToString("SEVERITY_ERROR"),
					CustomLabels: map[string]string{"foo": "bar", "baz": "faz"},
					Filters: []*rules.UpdateAlertRuleParamsBodyFiltersItems0{{
						Type:  pointer.ToString("EQUAL"),
						Key:   "param1",
						Value: "21",
					}},
					ChannelIds: []string{channelID, unknownChannelID},
				},
				Context: pmmapitests.Context,
			}
			_, err = rulesClient.UpdateAlertRule(params)
			pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Failed to find all required channels: [%s].", unknownChannelID)
		})

		t.Run("wrong parameter", func(t *testing.T) {
			t.Parallel()

			cParams := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			rule, err := rulesClient.CreateAlertRule(cParams)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, rule.Payload.RuleID)

			params := &rules.UpdateAlertRuleParams{
				Body: rules.UpdateAlertRuleBody{
					RuleID:   rule.Payload.RuleID,
					Disabled: false,
					Params: []*rules.UpdateAlertRuleParamsBodyParamsItems0{{
						Name:  "param2",
						Type:  pointer.ToString("FLOAT"),
						Float: 12,
					}, {
						Name:  "unknown parameter",
						Type:  pointer.ToString("FLOAT"),
						Float: 21,
					}},
					For:          "10s",
					Severity:     pointer.ToString("SEVERITY_ERROR"),
					CustomLabels: map[string]string{"foo": "bar", "baz": "faz"},
					Filters: []*rules.UpdateAlertRuleParamsBodyFiltersItems0{{
						Type:  pointer.ToString("EQUAL"),
						Key:   "param1",
						Value: "21",
					}},
					ChannelIds: []string{channelID, newChannelID},
				},
				Context: pmmapitests.Context,
			}
			_, err = rulesClient.UpdateAlertRule(params)
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Parameter param1 is missing.")
		})

		t.Run("missing parameters", func(t *testing.T) {
			t.Parallel()

			cParams := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			rule, err := rulesClient.CreateAlertRule(cParams)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, rule.Payload.RuleID)

			params := &rules.UpdateAlertRuleParams{
				Body: rules.UpdateAlertRuleBody{
					RuleID:       rule.Payload.RuleID,
					Disabled:     false,
					Params:       nil,
					For:          "10s",
					Severity:     pointer.ToString("SEVERITY_ERROR"),
					CustomLabels: map[string]string{"foo": "bar", "baz": "faz"},
					Filters: []*rules.UpdateAlertRuleParamsBodyFiltersItems0{{
						Type:  pointer.ToString("EQUAL"),
						Key:   "param1",
						Value: "21",
					}},
					ChannelIds: []string{channelID, newChannelID},
				},
				Context: pmmapitests.Context,
			}
			_, err = rulesClient.UpdateAlertRule(params)
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Expression requires 2 parameters, but got 0.")
		})

		t.Run("wrong parameter type", func(t *testing.T) {
			t.Parallel()

			cParams := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			rule, err := rulesClient.CreateAlertRule(cParams)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, rule.Payload.RuleID)

			params := &rules.UpdateAlertRuleParams{
				Body: rules.UpdateAlertRuleBody{
					RuleID:   rule.Payload.RuleID,
					Disabled: false,
					Params: []*rules.UpdateAlertRuleParamsBodyParamsItems0{
						{
							Name: "param1",
							Type: pointer.ToString("BOOL"),
							Bool: true,
						}, {
							Name:  "param2",
							Type:  pointer.ToString("FLOAT"),
							Float: 3.14,
						},
					},
					For:          "10s",
					Severity:     pointer.ToString("SEVERITY_ERROR"),
					CustomLabels: map[string]string{"foo": "bar", "baz": "faz"},
					Filters: []*rules.UpdateAlertRuleParamsBodyFiltersItems0{{
						Type:  pointer.ToString("EQUAL"),
						Key:   "param1",
						Value: "21",
					}},
					ChannelIds: []string{channelID, newChannelID},
				},
				Context: pmmapitests.Context,
			}
			_, err = rulesClient.UpdateAlertRule(params)
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Parameter param1 has type bool instead of float.")
		})
	})

	t.Run("toggle", func(t *testing.T) {
		t.Parallel()

		cParams := createAlertRuleParams(templateName, "", channelID, dummyFilter)
		rule, err := rulesClient.CreateAlertRule(cParams)
		require.NoError(t, err)
		defer deleteRule(t, rulesClient, rule.Payload.RuleID)

		list, err := rulesClient.ListAlertRules(&rules.ListAlertRulesParams{Context: pmmapitests.Context})
		require.NoError(t, err)

		var found bool
		for _, r := range list.Payload.Rules {
			if r.RuleID == rule.Payload.RuleID {
				assert.True(t, r.Disabled)
				assert.Equal(t, "SEVERITY_WARNING", pointer.GetString(r.Severity))
				found = true
			}
		}
		assert.Truef(t, found, "Rule with id %s not found", rule.Payload.RuleID)

		_, err = rulesClient.ToggleAlertRule(&rules.ToggleAlertRuleParams{
			Body: rules.ToggleAlertRuleBody{
				RuleID:   rule.Payload.RuleID,
				Disabled: pointer.ToString(rules.ToggleAlertRuleBodyDisabledFALSE),
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		list, err = rulesClient.ListAlertRules(&rules.ListAlertRulesParams{Context: pmmapitests.Context})
		require.NoError(t, err)

		found = false
		for _, r := range list.Payload.Rules {
			if r.RuleID == rule.Payload.RuleID {
				assert.False(t, r.Disabled)
				assert.Equal(t, "SEVERITY_WARNING", pointer.GetString(r.Severity))
				found = true
			}
		}
		assert.Truef(t, found, "Rule with id %s not found", rule.Payload.RuleID)
	})

	t.Run("delete", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()

			params := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			rule, err := rulesClient.CreateAlertRule(params)
			require.NoError(t, err)

			_, err = rulesClient.DeleteAlertRule(&rules.DeleteAlertRuleParams{
				Body:    rules.DeleteAlertRuleBody{RuleID: rule.Payload.RuleID},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			list, err := rulesClient.ListAlertRules(&rules.ListAlertRulesParams{Context: pmmapitests.Context})
			require.NoError(t, err)

			for _, r := range list.Payload.Rules {
				assert.NotEqual(t, rule.Payload.RuleID, r.RuleID)
			}
		})

		t.Run("missing rule", func(t *testing.T) {
			t.Parallel()
			ruleID := uuid.New().String()
			_, err := rulesClient.DeleteAlertRule(&rules.DeleteAlertRuleParams{
				Body:    rules.DeleteAlertRuleBody{RuleID: ruleID},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Rule with ID %q not found.", ruleID)
		})
	})

	t.Run("list without pagination", func(t *testing.T) {
		t.Run("without pagination", func(t *testing.T) {
			params := createAlertRuleParams(templateName, "", channelID, dummyFilter)
			rule, err := rulesClient.CreateAlertRule(params)
			require.NoError(t, err)
			defer deleteRule(t, rulesClient, rule.Payload.RuleID)

			list, err := rulesClient.ListAlertRules(&rules.ListAlertRulesParams{Context: pmmapitests.Context})
			require.NoError(t, err)

			var found bool
			for _, r := range list.Payload.Rules {
				if r.RuleID == rule.Payload.RuleID {
					assert.Equal(t, params.Body.TemplateName, r.TemplateName)
					assert.Equal(t, "Test summary", r.Summary)
					assert.True(t, r.Disabled)
					assert.Equal(t, params.Body.Name, r.Name)
					assert.Len(t, r.ParamsValues, 2)
					assert.Equal(t, params.Body.Params[0].Type, r.ParamsValues[0].Type)
					assert.Equal(t, params.Body.Params[0].Name, r.ParamsValues[0].Name)
					assert.Equal(t, params.Body.Params[0].Float, r.ParamsValues[0].Float)
					assert.Equal(t, params.Body.Params[0].Bool, r.ParamsValues[0].Bool)
					assert.Equal(t, params.Body.Params[0].String, r.ParamsValues[0].String)
					assert.Equal(t, params.Body.Params[1].Type, r.ParamsValues[1].Type)
					assert.Equal(t, params.Body.Params[1].Name, r.ParamsValues[1].Name)
					assert.Equal(t, params.Body.Params[1].Float, r.ParamsValues[1].Float)
					assert.Equal(t, params.Body.Params[1].Bool, r.ParamsValues[1].Bool)
					assert.Equal(t, params.Body.Params[1].String, r.ParamsValues[1].String)
					assert.Equal(t, "300s", r.DefaultFor)
					assert.Equal(t, params.Body.For, r.For)
					assert.Equal(t, "SEVERITY_WARNING", pointer.GetString(r.DefaultSeverity))
					assert.Equal(t, params.Body.Severity, r.Severity)
					assert.Equal(t, params.Body.CustomLabels, r.CustomLabels)
					assert.Equal(t, map[string]string{"foo": "bar"}, r.Labels)
					assert.Equal(t, map[string]string{"description": "test description", "summary": "test summary"}, r.Annotations)
					assert.Len(t, params.Body.Filters, 1)
					assert.Equal(t, params.Body.Filters[0].Type, r.Filters[0].Type)
					assert.Equal(t, params.Body.Filters[0].Key, r.Filters[0].Key)
					assert.Equal(t, params.Body.Filters[0].Value, r.Filters[0].Value)
					assert.Len(t, r.Channels, 1)
					assert.Equal(t, r.Channels[0].ChannelID, channelID)
					assert.Equal(t, "[[ .param1 ]] > 2 and 2 < [[ .param2 ]]", r.ExprTemplate)
					assert.Equal(t, "4 > 2 and 2 < 12", r.Expr)
					found = true
				}
			}
			assert.Truef(t, found, "Rule with id %s not found", rule.Payload.RuleID)
		})

		t.Run("with pagination", func(t *testing.T) {
			const rulesCount = 5

			ruleIDs := make(map[string]struct{})

			for i := 0; i < rulesCount; i++ {
				params := createAlertRuleParams(templateName, "", channelID, dummyFilter)
				rule, err := rulesClient.CreateAlertRule(params)
				require.NoError(t, err)

				ruleIDs[rule.Payload.RuleID] = struct{}{}
			}
			defer func() {
				for id := range ruleIDs {
					deleteRule(t, rulesClient, id)
				}
			}()

			// list rules, so they are all on the first page
			body := rules.ListAlertRulesBody{
				PageParams: &rules.ListAlertRulesParamsBodyPageParams{
					PageSize: 20,
					Index:    0,
				},
			}
			list1, err := rulesClient.ListAlertRules(&rules.ListAlertRulesParams{Body: body, Context: pmmapitests.Context})
			require.NoError(t, err)

			lp1 := list1.Payload
			// some tests didn't remove rules, so expect more elements than created in current test
			assert.GreaterOrEqual(t, len(lp1.Rules), rulesCount)
			assert.Equal(t, int32(len(lp1.Rules)), lp1.Totals.TotalItems)
			assert.Equal(t, int32(1), lp1.Totals.TotalPages)
			for id := range ruleIDs {
				var found bool
				for _, r := range list1.Payload.Rules {
					if r.RuleID == id {
						found = true

						break
					}
				}

				assert.Truef(t, found, "rule (%s) not found", id)
			}

			// paginate page over page with page size 1 and check the order - it should be the same as in list1.
			// last iteration checks that there is no elements for not existing page.
			for pageIndex := 0; pageIndex <= len(lp1.Rules); pageIndex++ {
				body := rules.ListAlertRulesBody{
					PageParams: &rules.ListAlertRulesParamsBodyPageParams{
						PageSize: 1,
						Index:    int32(pageIndex),
					},
				}
				list2, err := rulesClient.ListAlertRules(&rules.ListAlertRulesParams{Body: body, Context: pmmapitests.Context})
				require.NoError(t, err)

				lp2 := list2.Payload
				assert.Equal(t, lp1.Totals.TotalItems, lp2.Totals.TotalItems)
				assert.GreaterOrEqual(t, lp2.Totals.TotalPages, int32(rulesCount))

				if pageIndex != len(lp1.Rules) {
					require.Len(t, lp2.Rules, 1)
					assert.Equal(t, lp1.Rules[pageIndex].RuleID, lp2.Rules[0].RuleID)
				} else {
					assert.Len(t, lp2.Rules, 0)
				}
			}
		})
	})
}

func deleteRule(t *testing.T, client rules.ClientService, id string) {
	t.Helper()

	_, err := client.DeleteAlertRule(&rules.DeleteAlertRuleParams{
		Body:    rules.DeleteAlertRuleBody{RuleID: id},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
}

func createAlertRuleParams(templateName, sourceRuleID, channelID string, filter *rules.CreateAlertRuleParamsBodyFiltersItems0) *rules.CreateAlertRuleParams {
	rule := &rules.CreateAlertRuleParams{
		Body: rules.CreateAlertRuleBody{
			TemplateName: templateName,
			SourceRuleID: sourceRuleID,
			Disabled:     true,
			Name:         "example rule",
			Params: []*rules.CreateAlertRuleParamsBodyParamsItems0{
				{
					Name:  "param1",
					Type:  pointer.ToString("FLOAT"),
					Float: 4,
				},
				{
					Name:  "param2",
					Type:  pointer.ToString("FLOAT"),
					Float: 12,
				},
			},
			For:          "5s",
			Severity:     pointer.ToString("SEVERITY_WARNING"),
			CustomLabels: map[string]string{"foo": "bar"},
		},
		Context: pmmapitests.Context,
	}

	if channelID != "" {
		rule.Body.ChannelIds = []string{channelID}
	}

	if filter != nil {
		rule.Body.Filters = []*rules.CreateAlertRuleParamsBodyFiltersItems0{filter}
	}

	return rule
}

func createTemplate(t *testing.T) string {
	t.Helper()

	b, err := os.ReadFile("../../testdata/ia/template.yaml")
	require.NoError(t, err)

	templateName := uuid.New().String()
	expression := "'[[ .param1 ]] > 2 and 2 < [[ .param2 ]]'"
	_, err = alertingClient.Default.Alerting.CreateTemplate(&alerting.CreateTemplateParams{
		Body: alerting.CreateTemplateBody{
			Yaml: fmt.Sprintf(string(b), templateName, expression, "%", "s"),
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)

	return templateName
}

func createChannel(t *testing.T) (string, channels.AddChannelBody) {
	t.Helper()

	body := channels.AddChannelBody{
		Summary:  gofakeit.Quote(),
		Disabled: gofakeit.Bool(),
		EmailConfig: &channels.AddChannelParamsBodyEmailConfig{
			SendResolved: false,
			To:           []string{gofakeit.Email()},
		},
	}
	resp, err := client.Default.Channels.AddChannel(&channels.AddChannelParams{
		Body:    body,
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	return resp.Payload.ChannelID, body
}

func deleteTemplate(t *testing.T, client alerting.ClientService, name string) {
	t.Helper()

	_, err := client.DeleteTemplate(&alerting.DeleteTemplateParams{
		Body: alerting.DeleteTemplateBody{
			Name: name,
		},
		Context: pmmapitests.Context,
	})
	assert.NoError(t, err)
}
