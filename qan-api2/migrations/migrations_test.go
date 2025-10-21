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

package migrations

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddClusterSchemaMigrationsParams(t *testing.T) {
	tests := []struct {
		name        string
		inputURL    string
		clusterName string
		wantQuery   string
	}{
		{
			name:        "No cluster name",
			inputURL:    "clickhouse://localhost:9000?foo=bar",
			clusterName: "",
			wantQuery:   "foo=bar&x-migrations-table-engine=ReplicatedMergeTree('/clickhouse/tables/{shard}/{database}/schema_migrations', '{replica}') ORDER BY version",
		},
		{
			name:        "With cluster name",
			inputURL:    "clickhouse://localhost:9000?foo=bar",
			clusterName: "test-cluster",
			wantQuery:   "foo=bar&x-cluster-name=test-cluster&x-migrations-table-engine=ReplicatedMergeTree('/clickhouse/tables/{shard}/{database}/schema_migrations', '{replica}') ORDER BY version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.inputURL)
			require.NoError(t, err, "Failed to parse input URL")
			got, err := addClusterSchemaMigrationsParams(u, tt.clusterName)
			require.NoError(t, err, "addClusterSchemaMigrationsParams returned error")
			require.Equal(t, tt.wantQuery, got.RawQuery, "RawQuery mismatch")
		})
	}
}

// Skipping TestGetEngine until GetEngine is refactored for testability (dependency injection)
