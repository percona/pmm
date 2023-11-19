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
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/percona-platform/saas/pkg/alert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"gopkg.in/yaml.v3"

	pmmapitests "github.com/percona/pmm/api-tests"
	alertingClient "github.com/percona/pmm/api/management/v1/alerting/json/client"
	alerting "github.com/percona/pmm/api/management/v1/alerting/json/client/alerting_service"
	"github.com/percona/pmm/managed/services/grafana"
)

// Note: Even though the Alerting service checks for alerting enabled or disabled before returning results
// we don't enable or disable Alerting explicit in our tests since it is enabled by default through
// DISABLE_ALERTING env var.
func TestRulesAPI(t *testing.T) {
	t.Parallel()

	t.Parallel()
	client := alertingClient.Default.AlertingService

	// Create grafana folder for test alert rules
	grafanaClient := grafana.NewClient("127.0.0.1:3000")
	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:admin"))}))
	folder, err := grafanaClient.CreateFolder(ctx, "test-folder-"+uuid.NewString())
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, grafanaClient.DeleteFolder(ctx, folder.UID, true))
	})

	dummyFilter := &alerting.CreateRuleParamsBodyFiltersItems0{
		Type:   pointer.ToString("MATCH"),
		Label:  "threshold",
		Regexp: "12",
	}

	templateName := createTemplate(t)
	t.Cleanup(func() {
		deleteTemplate(t, alertingClient.Default.AlertingService, templateName)
	})

	t.Run("add", func(t *testing.T) {
		t.Parallel()

		t.Run("normal from template", func(t *testing.T) {
			t.Parallel()

			params := createAlertRuleParams(templateName, folder.UID, dummyFilter)
			_, err := client.CreateRule(params)
			require.NoError(t, err)
		})

		t.Run("builtin_template", func(t *testing.T) {
			t.Parallel()

			params := createAlertRuleParams("pmm_mongodb_restarted", folder.UID, dummyFilter)
			params.Body.Params = []*alerting.CreateRuleParamsBodyParamsItems0{{
				Name:  "threshold",
				Type:  pointer.ToString("FLOAT"),
				Float: 3.14,
			}}
			_, err := client.CreateRule(params)
			require.NoError(t, err)
		})

		t.Run("use default value for parameter", func(t *testing.T) {
			t.Parallel()

			params := createAlertRuleParams(templateName, folder.UID, dummyFilter)
			_, err := client.CreateRule(params)
			require.NoError(t, err)
		})

		t.Run("unknown template", func(t *testing.T) {
			t.Parallel()

			templateName := uuid.New().String()
			params := createAlertRuleParams(templateName, folder.UID, dummyFilter)
			_, err := client.CreateRule(params)
			pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Unknown template %s.", templateName)
		})

		t.Run("wrong parameter", func(t *testing.T) {
			t.Parallel()

			params := createAlertRuleParams(templateName, folder.UID, dummyFilter)
			params.Body.Params = append(
				params.Body.Params,
				&alerting.CreateRuleParamsBodyParamsItems0{
					Name:  "unknown parameter",
					Type:  pointer.ToString("FLOAT"),
					Float: 12,
				})
			_, err := client.CreateRule(params)
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Expression requires 2 parameters, but got 3.")
		})

		t.Run("wrong parameter type", func(t *testing.T) {
			t.Parallel()

			params := createAlertRuleParams(templateName, folder.UID, dummyFilter)
			params.Body.Params = []*alerting.CreateRuleParamsBodyParamsItems0{
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
			_, err := client.CreateRule(params)
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Parameter param1 has type bool instead of float.")
		})
	})
}

func TestTemplatesAPI(t *testing.T) {
	t.Parallel()
	client := alertingClient.Default.AlertingService

	templateData, err := os.ReadFile("../../testdata/ia/template.yaml")
	require.NoError(t, err)

	multipleTemplatesData, err := os.ReadFile("../../testdata/ia/multiple-templates.yaml")
	require.NoError(t, err)

	invalidTemplateData, err := os.ReadFile("../../testdata/ia/invalid-template.yaml")
	require.NoError(t, err)

	t.Run("add", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()

			name := uuid.New().String()
			expr := uuid.New().String()
			alertTemplates, yml := formatTemplateYaml(t, fmt.Sprintf(string(templateData), name, expr, "%", "s"))
			_, err := client.CreateTemplate(&alerting.CreateTemplateParams{
				Body: alerting.CreateTemplateBody{
					Yaml: yml,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteTemplate(t, client, name)

			resp, err := client.ListTemplates(&alerting.ListTemplatesParams{
				Body: alerting.ListTemplatesBody{
					Reload: true,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			assertTemplate(t, alertTemplates[0], resp.Payload.Templates)
		})

		t.Run("multiple templates at once", func(t *testing.T) {
			t.Parallel()

			alertTemplates, yml := formatTemplateYaml(t, string(multipleTemplatesData))
			_, err := client.CreateTemplate(&alerting.CreateTemplateParams{
				Body: alerting.CreateTemplateBody{
					Yaml: yml,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			require.Len(t, alertTemplates, 2)
			t.Cleanup(func() {
				deleteTemplate(t, client, alertTemplates[0].Name)
				deleteTemplate(t, client, alertTemplates[1].Name)
			})

			resp, err := client.ListTemplates(&alerting.ListTemplatesParams{
				Body: alerting.ListTemplatesBody{
					Reload: true,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			assertTemplate(t, alertTemplates[0], resp.Payload.Templates)
			assertTemplate(t, alertTemplates[1], resp.Payload.Templates)
		})

		t.Run("duplicate", func(t *testing.T) {
			t.Parallel()

			name := uuid.New().String()
			yaml := fmt.Sprintf(string(templateData), name, uuid.New().String(), "s", "%")
			_, err := client.CreateTemplate(&alerting.CreateTemplateParams{
				Body: alerting.CreateTemplateBody{
					Yaml: yaml,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteTemplate(t, client, name)

			_, err = client.CreateTemplate(&alerting.CreateTemplateParams{
				Body: alerting.CreateTemplateBody{
					Yaml: yaml,
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, fmt.Sprintf("Template with name \"%s\" already exists.", name))
		})

		t.Run("invalid yaml", func(t *testing.T) {
			t.Parallel()

			_, err := client.CreateTemplate(&alerting.CreateTemplateParams{
				Body: alerting.CreateTemplateBody{
					Yaml: "not a yaml",
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Failed to parse rule template")
		})

		t.Run("invalid template", func(t *testing.T) {
			t.Parallel()

			_, err := client.CreateTemplate(&alerting.CreateTemplateParams{
				Body: alerting.CreateTemplateBody{
					Yaml: fmt.Sprintf(string(invalidTemplateData), uuid.New().String(), uuid.New().String()),
				},
				Context: pmmapitests.Context,
			})

			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Failed to parse rule template")
		})
	})

	t.Run("change", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()

			name := uuid.New().String()
			expr := uuid.New().String()
			_, err := client.CreateTemplate(&alerting.CreateTemplateParams{
				Body: alerting.CreateTemplateBody{
					Yaml: fmt.Sprintf(string(templateData), name, expr, "s", "%"),
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteTemplate(t, client, name)

			newExpr := uuid.New().String()
			alertTemplates, yml := formatTemplateYaml(t, fmt.Sprintf(string(templateData), name, newExpr, "s", "%"))
			_, err = client.UpdateTemplate(&alerting.UpdateTemplateParams{
				Body: alerting.UpdateTemplateBody{
					Name: name,
					Yaml: yml,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			resp, err := client.ListTemplates(&alerting.ListTemplatesParams{
				Body: alerting.ListTemplatesBody{
					Reload: true,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			assertTemplate(t, alertTemplates[0], resp.Payload.Templates)
		})

		t.Run("unknown template", func(t *testing.T) {
			t.Parallel()

			name := uuid.New().String()
			_, err := client.UpdateTemplate(&alerting.UpdateTemplateParams{
				Body: alerting.UpdateTemplateBody{
					Name: name,
					Yaml: fmt.Sprintf(string(templateData), name, uuid.New().String(), "s", "%"),
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, fmt.Sprintf("Template with name \"%s\" not found.", name))
		})

		t.Run("invalid yaml", func(t *testing.T) {
			t.Parallel()

			name := uuid.New().String()
			_, err := client.CreateTemplate(&alerting.CreateTemplateParams{
				Body: alerting.CreateTemplateBody{
					Yaml: fmt.Sprintf(string(templateData), name, uuid.New().String(), "s", "%"),
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteTemplate(t, client, name)

			_, err = client.UpdateTemplate(&alerting.UpdateTemplateParams{
				Body: alerting.UpdateTemplateBody{
					Name: name,
					Yaml: "not a yaml",
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Failed to parse rule template.")
		})

		t.Run("invalid template", func(t *testing.T) {
			t.Parallel()

			name := uuid.New().String()
			_, err := client.CreateTemplate(&alerting.CreateTemplateParams{
				Body: alerting.CreateTemplateBody{
					Yaml: fmt.Sprintf(string(templateData), name, uuid.New().String(), "s", "%"),
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteTemplate(t, client, name)

			_, err = client.UpdateTemplate(&alerting.UpdateTemplateParams{
				Body: alerting.UpdateTemplateBody{
					Name: name,
					Yaml: fmt.Sprintf(string(invalidTemplateData), name, uuid.New().String()),
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Failed to parse rule template.")
		})
	})

	t.Run("delete", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()

			name := uuid.New().String()
			_, err := client.CreateTemplate(&alerting.CreateTemplateParams{
				Body: alerting.CreateTemplateBody{
					Yaml: fmt.Sprintf(string(templateData), name, uuid.New().String(), "s", "%"),
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			_, err = client.DeleteTemplate(&alerting.DeleteTemplateParams{
				Body: alerting.DeleteTemplateBody{
					Name: name,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			resp, err := client.ListTemplates(&alerting.ListTemplatesParams{
				Body: alerting.ListTemplatesBody{
					Reload: true,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			for _, template := range resp.Payload.Templates {
				assert.NotEqual(t, name, template.Name)
			}
		})

		t.Run("unknown template", func(t *testing.T) {
			t.Parallel()

			name := uuid.New().String()
			_, err := client.DeleteTemplate(&alerting.DeleteTemplateParams{
				Body: alerting.DeleteTemplateBody{
					Name: name,
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, fmt.Sprintf("Template with name \"%s\" not found.", name))
		})
	})

	t.Run("list", func(t *testing.T) {
		t.Run("without pagination", func(t *testing.T) {
			name := uuid.New().String()
			expr := uuid.New().String()
			alertTemplates, yml := formatTemplateYaml(t, fmt.Sprintf(string(templateData), name, expr, "%", "s"))
			_, err := client.CreateTemplate(&alerting.CreateTemplateParams{
				Body: alerting.CreateTemplateBody{
					Yaml: yml,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteTemplate(t, client, name)

			resp, err := client.ListTemplates(&alerting.ListTemplatesParams{
				Body: alerting.ListTemplatesBody{
					Reload: true,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			assertTemplate(t, alertTemplates[0], resp.Payload.Templates)
		})

		t.Run("with pagination", func(t *testing.T) {
			const templatesCount = 5

			templateNames := make(map[string]struct{})

			for i := 0; i < templatesCount; i++ {
				name := uuid.New().String()
				expr := uuid.New().String()
				_, yml := formatTemplateYaml(t, fmt.Sprintf(string(templateData), name, expr, "%", "s"))
				_, err := client.CreateTemplate(&alerting.CreateTemplateParams{
					Body: alerting.CreateTemplateBody{
						Yaml: yml,
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)

				templateNames[name] = struct{}{}
			}
			defer func() {
				for name := range templateNames {
					deleteTemplate(t, client, name)
				}
			}()

			// list rules, so they are all on the first page
			body := alerting.ListTemplatesBody{
				PageParams: &alerting.ListTemplatesParamsBodyPageParams{
					PageSize: 30,
					Index:    0,
				},
			}
			listAllTemplates, err := client.ListTemplates(&alerting.ListTemplatesParams{
				Body:    body,
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			assert.GreaterOrEqual(t, len(listAllTemplates.Payload.Templates), templatesCount)
			assert.Equal(t, int32(len(listAllTemplates.Payload.Templates)), listAllTemplates.Payload.Totals.TotalItems)
			assert.Equal(t, int32(1), listAllTemplates.Payload.Totals.TotalPages)

			assertFindTemplate := func(list []*alerting.ListTemplatesOKBodyTemplatesItems0, name string) func() bool {
				return func() bool {
					for _, tmpl := range list {
						if tmpl.Name == name {
							return true
						}
					}
					return false
				}
			}

			for name := range templateNames {
				assert.Conditionf(t, assertFindTemplate(listAllTemplates.Payload.Templates, name), "template %s not found", name)
			}

			// paginate page over page with page size 1 and check the order - it should be the same as in listAllTemplates.
			// last iteration checks that there is no elements for not existing page.
			for pageIndex := 0; pageIndex <= len(listAllTemplates.Payload.Templates); pageIndex++ {
				body := alerting.ListTemplatesBody{
					PageParams: &alerting.ListTemplatesParamsBodyPageParams{
						PageSize: 1,
						Index:    int32(pageIndex),
					},
				}
				listOneTemplate, err := client.ListTemplates(&alerting.ListTemplatesParams{
					Body: body, Context: pmmapitests.Context,
				})
				require.NoError(t, err)

				assert.Equal(t, listAllTemplates.Payload.Totals.TotalItems, listOneTemplate.Payload.Totals.TotalItems)
				assert.GreaterOrEqual(t, listOneTemplate.Payload.Totals.TotalPages, int32(templatesCount))

				if pageIndex != len(listAllTemplates.Payload.Templates) {
					require.Len(t, listOneTemplate.Payload.Templates, 1)
					assert.Equal(t, listAllTemplates.Payload.Templates[pageIndex].Name, listOneTemplate.Payload.Templates[0].Name)
				} else {
					assert.Len(t, listOneTemplate.Payload.Templates, 0)
				}
			}
		})
	})
}

func assertTemplate(t *testing.T, expectedTemplate alert.Template, listTemplates []*alerting.ListTemplatesOKBodyTemplatesItems0) {
	t.Helper()
	convertParamUnit := func(u string) alert.Unit {
		switch u {
		case alerting.ListTemplatesOKBodyTemplatesItems0ParamsItems0UnitPARAMUNITPERCENTAGE:
			return alert.Percentage
		case alerting.ListTemplatesOKBodyTemplatesItems0ParamsItems0UnitPARAMUNITSECONDS:
			return alert.Seconds
		}
		return "INVALID"
	}
	convertParamType := func(u string) alert.Type {
		switch u {
		case alerting.ListTemplatesOKBodyTemplatesItems0ParamsItems0TypePARAMTYPEFLOAT:
			return alert.Float
		case alerting.ListTemplatesOKBodyTemplatesItems0ParamsItems0TypePARAMTYPESTRING:
			return alert.String
		case alerting.ListTemplatesOKBodyTemplatesItems0ParamsItems0TypePARAMTYPEBOOL:
			return alert.Bool
		}
		return "INVALID"
	}
	var tmpl *alerting.ListTemplatesOKBodyTemplatesItems0
	for _, listTmpl := range listTemplates {
		if listTmpl.Name == expectedTemplate.Name {
			tmpl = listTmpl
			break
		}
	}
	require.NotNilf(t, tmpl, "template %s not found", expectedTemplate.Name)
	// IDE doesn't recognize that require stops execution
	if tmpl == nil {
		return
	}
	assert.Equal(t, expectedTemplate.Expr, tmpl.Expr)
	assert.Equal(t, expectedTemplate.Summary, tmpl.Summary)
	assert.Equal(t, "USER_API", *tmpl.Source)
	assert.Equal(t, "SEVERITY_WARNING", *tmpl.Severity)

	forDuration := fmt.Sprintf("%.0fs", time.Duration(expectedTemplate.For).Seconds())
	assert.Equal(t, forDuration, tmpl.For)

	require.Len(t, tmpl.Params, len(expectedTemplate.Params))
	for i, expectedParam := range expectedTemplate.Params {
		param := tmpl.Params[i]
		assert.Equal(t, expectedParam.Name, param.Name)
		assert.Equal(t, expectedParam.Summary, param.Summary)
		assert.Equal(t, expectedParam.Type, convertParamType(*param.Type))
		assert.Equal(t, expectedParam.Unit, convertParamUnit(*param.Unit))
		switch expectedParam.Type {
		case alert.Float:
			if expectedParam.Value != nil {
				require.NotNil(t, param.Float)
				value, err := expectedParam.GetValueForFloat()
				require.NoError(t, err)
				assert.True(t, param.Float.HasDefault)
				assert.Equal(t, value, param.Float.Default)
			}

			if len(expectedParam.Range) != 0 {
				min, max, err := expectedParam.GetRangeForFloat()
				require.NoError(t, err)
				assert.True(t, param.Float.HasMax)
				assert.True(t, param.Float.HasMin)
				assert.Equal(t, min, param.Float.Min)
				assert.Equal(t, max, param.Float.Max)
			}

			assert.Nil(t, param.Bool)
			assert.Nil(t, param.String)
		default:
		}
	}

	assert.Equal(t, expectedTemplate.Labels, tmpl.Labels)
	assert.Equal(t, expectedTemplate.Annotations, tmpl.Annotations)

	expectedYAML, err := alert.ToYAML([]alert.Template{expectedTemplate})
	require.NoError(t, err)
	assert.Equal(t, expectedYAML, tmpl.Yaml)

	assert.NotEmpty(t, tmpl.CreatedAt)
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

func formatTemplateYaml(t *testing.T, yml string) ([]alert.Template, string) {
	t.Helper()
	params := &alert.ParseParams{
		DisallowUnknownFields:    true,
		DisallowInvalidTemplates: true,
	}
	r, err := alert.Parse(strings.NewReader(yml), params)
	require.NoError(t, err)
	type yamlTemplates struct {
		Templates []alert.Template `yaml:"templates"`
	}
	s, err := yaml.Marshal(&yamlTemplates{Templates: r})
	require.NoError(t, err)

	return r, string(s)
}

func createAlertRuleParams(templateName, folderUID string, filter *alerting.CreateRuleParamsBodyFiltersItems0) *alerting.CreateRuleParams {
	rule := &alerting.CreateRuleParams{
		Body: alerting.CreateRuleBody{
			TemplateName: templateName,
			Name:         "test-rule-" + uuid.NewString(),
			FolderUID:    folderUID,
			Group:        "test",
			Params: []*alerting.CreateRuleParamsBodyParamsItems0{
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
			For:          "90s",
			Severity:     pointer.ToString("SEVERITY_WARNING"),
			CustomLabels: map[string]string{"foo": "bar"},
		},
		Context: pmmapitests.Context,
	}

	if filter != nil {
		rule.Body.Filters = []*alerting.CreateRuleParamsBodyFiltersItems0{filter}
	}

	return rule
}

func createTemplate(t *testing.T) string {
	t.Helper()

	b, err := os.ReadFile("../../testdata/alerting/template.yaml")
	require.NoError(t, err)

	templateName := uuid.New().String()
	expression := "'[[ .param1 ]] > 2 and 2 < [[ .param2 ]]'"
	_, err = alertingClient.Default.AlertingService.CreateTemplate(&alerting.CreateTemplateParams{
		Body: alerting.CreateTemplateBody{
			Yaml: fmt.Sprintf(string(b), templateName, expression, "%", "s"),
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)

	return templateName
}
