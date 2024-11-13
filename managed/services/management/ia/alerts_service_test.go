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

package ia

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/go-openapi/strfmt"
	"github.com/percona-platform/saas/pkg/alert"
	"github.com/percona-platform/saas/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/api/alertmanager/ammodels"
	"github.com/percona/pmm/api/managementpb"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestSatisfiesFilters(t *testing.T) {
	t.Parallel()

	t.Run("alerts", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			alert  *ammodels.GettableAlert
			result bool
			errMsg string
		}{{
			name: "normal",
			alert: &ammodels.GettableAlert{Alert: ammodels.Alert{Labels: map[string]string{
				"label1": "value1",
				"label2": "value2",
			}}},
			result: true,
			errMsg: "",
		}, {
			name: "only one label match",
			alert: &ammodels.GettableAlert{Alert: ammodels.Alert{Labels: map[string]string{
				"label1": "value1",
				"label2": "wrong",
			}}},
			result: false,
			errMsg: "",
		}, {
			name: "only one label present",
			alert: &ammodels.GettableAlert{Alert: ammodels.Alert{Labels: map[string]string{
				"label2": "value2",
			}}},
			result: false,
			errMsg: "",
		}, {
			name: "no labels",
			alert: &ammodels.GettableAlert{
				Alert: ammodels.Alert{Labels: make(map[string]string)},
			},
			result: false,
			errMsg: "",
		}, {
			name: "match labels + unknown label",
			alert: &ammodels.GettableAlert{Alert: ammodels.Alert{Labels: map[string]string{
				"label1":  "value1",
				"label2":  "value2",
				"unknown": "value",
			}}},
			result: true,
			errMsg: "",
		}, {
			name: "not match labels + unknown label",
			alert: &ammodels.GettableAlert{Alert: ammodels.Alert{Labels: map[string]string{
				"label2":  "value2",
				"unknown": "value",
			}}},
			result: false,
			errMsg: "",
		}}

		filters := []*iav1beta1.Filter{{ //nolint:staticcheck
			Type:  iav1beta1.FilterType_EQUAL,
			Key:   "label1",
			Value: "value1",
		}, {
			Type:  iav1beta1.FilterType_REGEX,
			Key:   "label2",
			Value: "v.*2",
		}}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				res, err := satisfiesFilters(tt.alert, filters)
				if tt.errMsg != "" {
					assert.EqualError(t, err, tt.errMsg)
					return
				}

				require.NoError(t, err)
				assert.Equal(t, tt.result, res)
			})
		}
	})

	t.Run("filters", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			filters []*iav1beta1.Filter //nolint:staticcheck
			result  bool
			errMsg  string
		}{
			{
				name: "normal multiple filters",
				filters: []*iav1beta1.Filter{{ //nolint:staticcheck
					Type:  iav1beta1.FilterType_EQUAL,
					Key:   "label1",
					Value: "value1",
				}, {
					Type:  iav1beta1.FilterType_REGEX,
					Key:   "label2",
					Value: "v.*2",
				}},
				result: true,
				errMsg: "",
			}, {
				name: "normal simple filter",
				filters: []*iav1beta1.Filter{{ //nolint:staticcheck
					Type:  iav1beta1.FilterType_EQUAL,
					Key:   "label1",
					Value: "value1",
				}},
				result: true,
				errMsg: "",
			}, {
				name: "normal regex filter",
				filters: []*iav1beta1.Filter{{ //nolint:staticcheck
					Type:  iav1beta1.FilterType_REGEX,
					Key:   "label2",
					Value: "v.*2",
				}},
				result: true,
				errMsg: "",
			}, {
				name: "invalid type",
				filters: []*iav1beta1.Filter{{ //nolint:staticcheck
					Type:  iav1beta1.FilterType_FILTER_TYPE_INVALID,
					Key:   "label1",
					Value: "value1",
				}},
				result: false,
				errMsg: "rpc error: code = Internal desc = Unexpected filter type.",
			}, {
				name: "unknown type",
				filters: []*iav1beta1.Filter{{ //nolint:staticcheck
					Type:  iav1beta1.FilterType(12), //nolint:staticcheck
					Key:   "label1",
					Value: "value1",
				}},
				result: false,
				errMsg: "rpc error: code = Internal desc = Unexpected filter type.",
			}, {
				name: "bad regexp",
				filters: []*iav1beta1.Filter{{ //nolint:staticcheck
					Type:  iav1beta1.FilterType_REGEX,
					Key:   "label2",
					Value: ".***",
				}},
				result: false,
				errMsg: "rpc error: code = InvalidArgument desc = bad regular expression: +error parsing regexp: invalid nested repetition operator: `**`",
			},
		}

		alert := &ammodels.GettableAlert{
			Alert: ammodels.Alert{
				Labels: map[string]string{
					"label1": "value1",
					"label2": "value2",
				},
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				res, err := satisfiesFilters(alert, tt.filters)
				if tt.errMsg != "" {
					assert.EqualError(t, err, tt.errMsg)
					return
				}

				require.NoError(t, err)
				assert.Equal(t, tt.result, res)
			})
		}
	})
}

func TestListAlerts(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	q := db.Querier
	now := strfmt.DateTime(time.Now())

	tmpl, err := models.CreateTemplate(q, &models.CreateTemplateParams{
		Template: &alert.Template{
			Name:     "test_Template",
			Version:  1,
			Summary:  gofakeit.Quote(),
			Expr:     gofakeit.Quote(),
			Severity: common.Warning,
		},
		Source: "USER_FILE",
	})
	require.NoError(t, err)

	rule, err := models.CreateRule(q, &models.CreateRuleParams{
		TemplateName:    tmpl.Name,
		DefaultSeverity: tmpl.Severity,
		Severity:        models.Severity(common.Warning),
	})
	require.NoError(t, err)

	const alertsCount = 25
	mockAlert := &mockAlertManager{}
	var mockedAlerts []*ammodels.GettableAlert

	for i := 0; i < alertsCount-1; i++ {
		mockedAlerts = append(mockedAlerts, &ammodels.GettableAlert{
			Alert: ammodels.Alert{
				Labels: map[string]string{
					"ia":        "1",
					"alertname": rule.ID,
				},
			},
			Fingerprint: pointer.ToString(strconv.Itoa(i)),
			Status: &ammodels.AlertStatus{
				State: pointer.ToString("active"),
			},
			StartsAt:  &now,
			UpdatedAt: &now,
		})
	}

	// This additional alert emulates the one created by user directly, without IA UI.
	// Since user added "ia" label, he wants to see alert in IA UI. We want to be sure he will.
	mockedAlerts = append(mockedAlerts, &ammodels.GettableAlert{
		Alert: ammodels.Alert{
			Labels: map[string]string{
				"ia": "1",
			},
		},
		Fingerprint: pointer.ToString(strconv.Itoa(alertsCount - 1)),
		Status: &ammodels.AlertStatus{
			State: pointer.ToString("active"),
		},
		StartsAt:  &now,
		UpdatedAt: &now,
	})

	mockAlert.On("GetAlerts", ctx, mock.Anything).Return(mockedAlerts, nil)

	var tmplSvc mockTemplatesService
	require.NoError(t, err)
	svc := NewAlertsService(db, mockAlert, &tmplSvc)

	findAlerts := func(alerts []*iav1beta1.Alert, alertIDs ...string) bool { //nolint:staticcheck
		if len(alerts) != len(alertIDs) {
			return false
		}
		m := make(map[string]bool)
		for _, alertID := range alertIDs {
			m[alertID] = false
		}
		for _, a := range alerts {
			val, ok := m[a.AlertId]
			// Extra alert
			if !ok {
				return false
			}
			// Duplicate
			if val {
				return false
			}
			m[a.AlertId] = true
		}
		for _, v := range m {
			// AlertID was not in alerts
			if !v {
				return false
			}
		}
		return true
	}

	t.Run("without pagination", func(t *testing.T) {
		res, err := svc.ListAlerts(ctx, &iav1beta1.ListAlertsRequest{}) //nolint:staticcheck
		assert.NoError(t, err)
		var expect []string
		for _, m := range mockedAlerts {
			expect = append(expect, *m.Fingerprint)
		}
		assert.True(t, findAlerts(res.Alerts, expect...), "wrong alerts returned")
		assert.EqualValues(t, res.Totals.TotalItems, len(mockedAlerts))
	})

	t.Run("pagination", func(t *testing.T) {
		res, err := svc.ListAlerts(ctx, &iav1beta1.ListAlertsRequest{ //nolint:staticcheck
			PageParams: &managementpb.PageParams{
				PageSize: 1,
			},
		})
		assert.NoError(t, err)
		assert.Len(t, res.Alerts, 1)
		assert.True(t, findAlerts(res.Alerts, "0"), "wrong alerts returned")
		assert.EqualValues(t, res.Totals.TotalItems, alertsCount)
		assert.EqualValues(t, res.Totals.TotalPages, alertsCount)

		res, err = svc.ListAlerts(ctx, &iav1beta1.ListAlertsRequest{ //nolint:staticcheck
			PageParams: &managementpb.PageParams{
				PageSize: 10,
				Index:    2,
			},
		})
		assert.NoError(t, err)
		assert.Len(t, res.Alerts, 5)
		assert.True(t, findAlerts(res.Alerts, "20", "21", "22", "23", "24"), "wrong alerts returned")
		assert.EqualValues(t, res.Totals.TotalItems, alertsCount)
		assert.EqualValues(t, res.Totals.TotalPages, 3)
	})

	t.Run("fetch more than available", func(t *testing.T) {
		var expect []string
		for _, m := range mockedAlerts {
			expect = append(expect, *m.Fingerprint)
		}
		res, err := svc.ListAlerts(ctx, &iav1beta1.ListAlertsRequest{ //nolint:staticcheck
			PageParams: &managementpb.PageParams{
				PageSize: alertsCount * 2,
			},
		})
		assert.NoError(t, err)
		assert.True(t, findAlerts(res.Alerts, expect...), "wrong alerts returned")
		assert.EqualValues(t, res.Totals.TotalItems, len(mockedAlerts))

		res, err = svc.ListAlerts(ctx, &iav1beta1.ListAlertsRequest{ //nolint:staticcheck
			PageParams: &managementpb.PageParams{
				PageSize: 1,
				Index:    alertsCount * 2,
			},
		})
		assert.NoError(t, err)
		assert.Len(t, res.Alerts, 0)
		assert.EqualValues(t, res.Totals.TotalItems, len(mockedAlerts))
	})
}
