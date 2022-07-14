// Copyright (C) 2017 Percona LLC
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
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/percona-platform/saas/pkg/alert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"gopkg.in/yaml.v3"

	pmmapitests "github.com/percona/pmm/api-tests"
	templatesClient "github.com/percona/pmm/api/managementpb/ia/json/client"
	"github.com/percona/pmm/api/managementpb/ia/json/client/rules"
	"github.com/percona/pmm/api/managementpb/ia/json/client/templates"
)

// Note: Even though the IA services check for alerting enabled or disabled before returning results
// we don't enable or disable IA explicit in our tests since it is enabled by default through
// ENABLE_ALERTING env var.
func assertTemplate(t *testing.T, expectedTemplate alert.Template, listTemplates []*templates.ListTemplatesOKBodyTemplatesItems0) {
	convertParamUnit := func(u string) alert.Unit {
		switch u {
		case templates.ListTemplatesOKBodyTemplatesItems0ParamsItems0UnitPERCENTAGE:
			return alert.Percentage
		case templates.ListTemplatesOKBodyTemplatesItems0ParamsItems0UnitSECONDS:
			return alert.Seconds
		}
		return "INVALID"
	}
	convertParamType := func(u string) alert.Type {
		switch u {
		case templates.ListTemplatesOKBodyTemplatesItems0ParamsItems0TypeFLOAT:
			return alert.Float
		case templates.ListTemplatesOKBodyTemplatesItems0ParamsItems0TypeSTRING:
			return alert.String
		case templates.ListTemplatesOKBodyTemplatesItems0ParamsItems0TypeBOOL:
			return alert.Bool
		}
		return "INVALID"
	}
	var tmpl *templates.ListTemplatesOKBodyTemplatesItems0
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

	assert.NotEmpty(t, tmpl.CreatedAt)
}

func TestTemplatesAPI(t *testing.T) {
	client := templatesClient.Default.Templates

	templateData, err := ioutil.ReadFile("../../testdata/ia/template.yaml")
	require.NoError(t, err)

	invalidTemplateData, err := ioutil.ReadFile("../../testdata/ia/invalid-template.yaml")
	require.NoError(t, err)

	t.Run("add", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()

			name := uuid.New().String()
			expr := uuid.New().String()
			alertTemplates, yml := formatTemplateYaml(t, fmt.Sprintf(string(templateData), name, expr, "%", "s"))
			_, err := client.CreateTemplate(&templates.CreateTemplateParams{
				Body: templates.CreateTemplateBody{
					Yaml: yml,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteTemplate(t, client, name)

			resp, err := client.ListTemplates(&templates.ListTemplatesParams{
				Body: templates.ListTemplatesBody{
					Reload: true,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			assertTemplate(t, alertTemplates[0], resp.Payload.Templates)
		})

		t.Run("duplicate", func(t *testing.T) {
			t.Parallel()

			name := uuid.New().String()
			yaml := fmt.Sprintf(string(templateData), name, uuid.New().String(), "s", "%")
			_, err := client.CreateTemplate(&templates.CreateTemplateParams{
				Body: templates.CreateTemplateBody{
					Yaml: yaml,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteTemplate(t, client, name)

			_, err = client.CreateTemplate(&templates.CreateTemplateParams{
				Body: templates.CreateTemplateBody{
					Yaml: yaml,
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, fmt.Sprintf("Template with name \"%s\" already exists.", name))
		})

		t.Run("invalid yaml", func(t *testing.T) {
			t.Parallel()

			_, err := client.CreateTemplate(&templates.CreateTemplateParams{
				Body: templates.CreateTemplateBody{
					Yaml: "not a yaml",
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Failed to parse rule template.")
		})

		t.Run("invalid template", func(t *testing.T) {
			t.Parallel()

			_, err := client.CreateTemplate(&templates.CreateTemplateParams{
				Body: templates.CreateTemplateBody{
					Yaml: fmt.Sprintf(string(invalidTemplateData), uuid.New().String(), uuid.New().String()),
				},
				Context: pmmapitests.Context,
			})

			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Failed to parse rule template.")
		})
	})

	t.Run("change", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()

			name := uuid.New().String()
			expr := uuid.New().String()
			_, err := client.CreateTemplate(&templates.CreateTemplateParams{
				Body: templates.CreateTemplateBody{
					Yaml: fmt.Sprintf(string(templateData), name, expr, "s", "%"),
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteTemplate(t, client, name)

			newExpr := uuid.New().String()
			alertTemplates, yml := formatTemplateYaml(t, fmt.Sprintf(string(templateData), name, newExpr, "s", "%"))
			_, err = client.UpdateTemplate(&templates.UpdateTemplateParams{
				Body: templates.UpdateTemplateBody{
					Name: name,
					Yaml: yml,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			resp, err := client.ListTemplates(&templates.ListTemplatesParams{
				Body: templates.ListTemplatesBody{
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
			_, err := client.UpdateTemplate(&templates.UpdateTemplateParams{
				Body: templates.UpdateTemplateBody{
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
			_, err := client.CreateTemplate(&templates.CreateTemplateParams{
				Body: templates.CreateTemplateBody{
					Yaml: fmt.Sprintf(string(templateData), name, uuid.New().String(), "s", "%"),
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteTemplate(t, client, name)

			_, err = client.UpdateTemplate(&templates.UpdateTemplateParams{
				Body: templates.UpdateTemplateBody{
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
			_, err := client.CreateTemplate(&templates.CreateTemplateParams{
				Body: templates.CreateTemplateBody{
					Yaml: fmt.Sprintf(string(templateData), name, uuid.New().String(), "s", "%"),
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteTemplate(t, client, name)

			_, err = client.UpdateTemplate(&templates.UpdateTemplateParams{
				Body: templates.UpdateTemplateBody{
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
			_, err := client.CreateTemplate(&templates.CreateTemplateParams{
				Body: templates.CreateTemplateBody{
					Yaml: fmt.Sprintf(string(templateData), name, uuid.New().String(), "s", "%"),
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			_, err = client.DeleteTemplate(&templates.DeleteTemplateParams{
				Body: templates.DeleteTemplateBody{
					Name: name,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			resp, err := client.ListTemplates(&templates.ListTemplatesParams{
				Body: templates.ListTemplatesBody{
					Reload: true,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			for _, template := range resp.Payload.Templates {
				assert.NotEqual(t, name, template.Name)
			}
		})

		t.Run("template in use", func(t *testing.T) {
			t.Parallel()

			name := uuid.New().String()
			_, err := client.CreateTemplate(&templates.CreateTemplateParams{
				Body: templates.CreateTemplateBody{
					Yaml: fmt.Sprintf(string(templateData), name, uuid.New().String(), "s", "%"),
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			channelID, _ := createChannel(t)
			defer deleteChannel(t, templatesClient.Default.Channels, channelID)

			params := createAlertRuleParams(name, "", channelID, &rules.CreateAlertRuleParamsBodyFiltersItems0{
				Type:  pointer.ToString("EQUAL"),
				Key:   "threshold",
				Value: "12",
			})

			rule, err := templatesClient.Default.Rules.CreateAlertRule(params)
			require.NoError(t, err)
			defer deleteRule(t, templatesClient.Default.Rules, rule.Payload.RuleID)

			_, err = client.DeleteTemplate(&templates.DeleteTemplateParams{
				Body: templates.DeleteTemplateBody{
					Name: name,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			resp, err := client.ListTemplates(&templates.ListTemplatesParams{
				Body: templates.ListTemplatesBody{
					Reload: true,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			for _, template := range resp.Payload.Templates {
				assert.Falsef(t, name == template.Name, "Template with id %s wasn't deleted", name)
			}
		})

		t.Run("unknown template", func(t *testing.T) {
			t.Parallel()

			name := uuid.New().String()
			_, err := client.DeleteTemplate(&templates.DeleteTemplateParams{
				Body: templates.DeleteTemplateBody{
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
			_, err := client.CreateTemplate(&templates.CreateTemplateParams{
				Body: templates.CreateTemplateBody{
					Yaml: yml,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteTemplate(t, client, name)

			resp, err := client.ListTemplates(&templates.ListTemplatesParams{
				Body: templates.ListTemplatesBody{
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
				_, err := client.CreateTemplate(&templates.CreateTemplateParams{
					Body: templates.CreateTemplateBody{
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
			body := templates.ListTemplatesBody{
				PageParams: &templates.ListTemplatesParamsBodyPageParams{
					PageSize: 30,
					Index:    0,
				},
			}
			listAllTemplates, err := client.ListTemplates(&templates.ListTemplatesParams{
				Body:    body,
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			assert.GreaterOrEqual(t, len(listAllTemplates.Payload.Templates), templatesCount)
			assert.Equal(t, int32(len(listAllTemplates.Payload.Templates)), listAllTemplates.Payload.Totals.TotalItems)
			assert.Equal(t, int32(1), listAllTemplates.Payload.Totals.TotalPages)

			assertFindTemplate := func(list []*templates.ListTemplatesOKBodyTemplatesItems0, name string) func() bool {
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
				body := templates.ListTemplatesBody{
					PageParams: &templates.ListTemplatesParamsBodyPageParams{
						PageSize: 1,
						Index:    int32(pageIndex),
					},
				}
				listOneTemplate, err := client.ListTemplates(&templates.ListTemplatesParams{
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

func deleteTemplate(t *testing.T, client templates.ClientService, name string) {
	_, err := client.DeleteTemplate(&templates.DeleteTemplateParams{
		Body: templates.DeleteTemplateBody{
			Name: name,
		},
		Context: pmmapitests.Context,
	})
	assert.NoError(t, err)
}

func formatTemplateYaml(t *testing.T, yml string) ([]alert.Template, string) {
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
