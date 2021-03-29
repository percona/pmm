// pmm-managed
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
	"testing"

	"github.com/percona/pmm/api/alertmanager/ammodels"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name:  "na labels",
			alert: &ammodels.GettableAlert{Alert: ammodels.Alert{Labels: map[string]string{
				// No labels
			}}},
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

		filters := []*iav1beta1.Filter{{
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
			filters []*iav1beta1.Filter
			result  bool
			errMsg  string
		}{
			{
				name: "normal multiple filters",
				filters: []*iav1beta1.Filter{{
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
				filters: []*iav1beta1.Filter{{
					Type:  iav1beta1.FilterType_EQUAL,
					Key:   "label1",
					Value: "value1",
				}},
				result: true,
				errMsg: "",
			}, {
				name: "normal regex filter",
				filters: []*iav1beta1.Filter{{
					Type:  iav1beta1.FilterType_REGEX,
					Key:   "label2",
					Value: "v.*2",
				}},
				result: true,
				errMsg: "",
			}, {
				name: "invalid type",
				filters: []*iav1beta1.Filter{{
					Type:  iav1beta1.FilterType_FILTER_TYPE_INVALID,
					Key:   "label1",
					Value: "value1",
				}},
				result: false,
				errMsg: "rpc error: code = Internal desc = Unexpected filter type.",
			}, {
				name: "unknown type",
				filters: []*iav1beta1.Filter{{
					Type:  iav1beta1.FilterType(12),
					Key:   "label1",
					Value: "value1",
				}},
				result: false,
				errMsg: "rpc error: code = Internal desc = Unexpected filter type.",
			}, {
				name: "bad regexp",
				filters: []*iav1beta1.Filter{{
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
