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

package alert

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/pi/common"
)

func TestClickHouseDatasourceTemplate(t *testing.T) {
	t.Run("built-in mysql_log_down parses with clickhouse datasource", func(t *testing.T) {
		f, err := os.Open("../../data/alerting-templates/mysql_log_down.yml")
		require.NoError(t, err)
		t.Cleanup(func() { _ = f.Close() })

		templates, err := Parse(f, &ParseParams{DisallowUnknownFields: true, DisallowInvalidTemplates: true})
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, "pmm_mysql_log_down", templates[0].Name)
		assert.Equal(t, DatasourceClickHouse, templates[0].Datasource)
	})

	t.Run("invalid datasource is rejected", func(t *testing.T) {
		tmpl := Template{Version: 1, Name: "t", Summary: "s", Expr: "e", Datasource: "loki"}
		require.Error(t, tmpl.Validate())
	})

	t.Run("empty datasource is accepted", func(t *testing.T) {
		tmpl := Template{Version: 1, Name: "t", Summary: "s", Expr: "e", Severity: common.Critical}
		require.NoError(t, tmpl.Validate())
	})
}
