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

package dbaas

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestVersionServiceClient(t *testing.T) {
	c := NewVersionServiceClient("https://check.percona.com/versions/v1")

	for _, tt := range []struct {
		params componentsParams
	}{
		{params: componentsParams{operator: psmdbOperator}},
		{params: componentsParams{operator: psmdbOperator, operatorVersion: "1.6.0"}},
		{params: componentsParams{operator: psmdbOperator, operatorVersion: "1.7.0", dbVersion: "4.2.8-8"}},
		{params: componentsParams{operator: pxcOperator}},
		{params: componentsParams{operator: pxcOperator, operatorVersion: "1.7.0"}},
		{params: componentsParams{operator: pxcOperator, operatorVersion: "1.7.0", dbVersion: "8.0.20-11.2"}},
	} {
		t.Run("NotEmptyMatrix", func(t *testing.T) {
			response, err := c.Matrix(context.TODO(), tt.params)
			require.NoError(t, err)
			require.NotEmpty(t, response.Versions)
			for _, v := range response.Versions {
				switch tt.params.operator {
				case psmdbOperator:
					assert.NotEmpty(t, v.Matrix.Mongod)
				case pxcOperator:
					assert.NotEmpty(t, v.Matrix.Pxc)
					assert.NotEmpty(t, v.Matrix.Proxysql)
				}
				assert.NotEmpty(t, v.Matrix.Backup)
			}
		})
	}
}
