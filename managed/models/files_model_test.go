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

package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/managed/models"
)

func TestInsertFileParamsValidate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		description string
		fp          models.InsertFileParams
		wantError   bool
	}

	nameTests := []testCase{
		{
			description: `insert file with empty`,
			fp:          models.InsertFileParams{Name: ``, Content: []byte("test")},
			wantError:   true,
		},
		{
			description: `insert file with .`,
			fp:          models.InsertFileParams{Name: `.`, Content: []byte("test")},
			wantError:   true,
		},
		{
			description: `insert file with ..`,
			fp:          models.InsertFileParams{Name: `..`, Content: []byte("test")},
			wantError:   true,
		},
		{
			description: `insert file with /`,
			fp:          models.InsertFileParams{Name: `/`, Content: []byte("test")},
			wantError:   true,
		},
		{
			description: `insert file with //`,
			fp:          models.InsertFileParams{Name: `//`, Content: []byte("test")},
			wantError:   true,
		},
		{
			description: `insert file with ///`,
			fp:          models.InsertFileParams{Name: `///`, Content: []byte("test")},
			wantError:   true,
		},
		{
			description: `insert file with \\`,
			fp:          models.InsertFileParams{Name: `\\`, Content: []byte("test")},
			wantError:   true,
		},
		{
			description: `insert file with \\\`,
			fp:          models.InsertFileParams{Name: `\\\`, Content: []byte("test")},
			wantError:   true,
		},
		{
			description: `insert file with "/srv/prometheus/prometheus.base.yml"`,
			fp:          models.InsertFileParams{Name: `/srv/prometheus/prometheus.base.yml`, Content: []byte("test")},
			wantError:   true,
		},
		{
			description: `insert file with prometheus.base.yml`,
			fp:          models.InsertFileParams{Name: `prometheus.base.yml`, Content: []byte("test")},
			wantError:   false,
		},
	}

	for i := range nameTests {
		test := nameTests[i]
		t.Run(test.description, func(t *testing.T) {
			err := test.fp.Validate()
			if test.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
