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

package collectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/version"
)

func TestPostgresExporterCollectorArgs(t *testing.T) {
	t.Parallel()

	v360 := version.MustParse("3.6.0")
	tests := []struct {
		name     string
		enabled  []string
		disabled []string
		version  *version.Parsed
		want     []string
	}{
		{
			name:    "NothingByDefault",
			version: v360,
		},
		{
			name:     "DisableRunningCollector",
			disabled: []string{"locks", "locks"},
			version:  v360,
			want:     []string{"--no-collector.locks"},
		},
		{
			// The exporter doesn't run it anyway, so there is nothing to turn off.
			name:     "DisableStoppedCollector",
			disabled: []string{"stat_statements"},
			version:  v360,
		},
		{
			// custom_query.* belong to the --collect. flags; FilterOutCollectors handles them.
			name:     "DisableCustomQuery",
			disabled: []string{"custom_query.hr"},
			version:  v360,
		},
		{
			name:    "EnableStoppedCollectors",
			enabled: []string{"stat_wal_receiver", "long_running_transactions", "long_running_transactions"},
			version: v360,
			want:    []string{"--collector.long_running_transactions", "--collector.stat_wal_receiver"},
		},
		{
			// It runs by default, so it needs no flag.
			name:    "EnableRunningCollector",
			enabled: []string{"database_wraparound"},
			version: v360,
		},
		{
			name:     "DisablingWins",
			enabled:  []string{"postmaster"},
			disabled: []string{"postmaster"},
			version:  v360,
		},
		{
			name:    "UnknownName",
			enabled: []string{"no_such_collector"},
			version: v360,
		},
		{
			name:    "CollectorNewerThanExporter",
			enabled: []string{"stat_io", "postmaster"},
			version: version.MustParse("3.5.0"),
			want:    []string{"--collector.postmaster"},
		},
		{
			name:    "CollectorAsNewAsExporter",
			enabled: []string{"stat_io"},
			version: v360,
			want:    []string{"--collector.stat_io"},
		},
		{
			name:     "ExporterWithoutCollectorFlags",
			enabled:  []string{"postmaster"},
			disabled: []string{"locks"},
			version:  version.MustParse("2.40.1"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, PostgresExporterCollectorArgs(tt.enabled, tt.disabled, tt.version))
		})
	}
}

func TestValidatePostgresExporterCollectors(t *testing.T) {
	t.Parallel()

	v360 := version.MustParse("3.6.0")

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		err := ValidatePostgresExporterCollectors([]string{"wal", "stat_io", "locks"}, []string{"stat_statements"}, v360)
		require.NoError(t, err)
	})

	t.Run("UnknownName", func(t *testing.T) {
		t.Parallel()

		err := ValidatePostgresExporterCollectors([]string{"custom_query.hr"}, nil, v360)
		require.EqualError(t, err, `unknown postgres_exporter collector "custom_query.hr"; the collectors are: `+
			`database, database_wraparound, extensions, locks, long_running_transactions, postmaster, process_idle, `+
			`replication, replication_slot, stat_activity_autovacuum, stat_bgwriter, stat_database, stat_io, `+
			`stat_statements, stat_user_tables, stat_wal_receiver, statio_user_indexes, statio_user_tables, wal`)
	})

	t.Run("EnabledAndDisabled", func(t *testing.T) {
		t.Parallel()

		err := ValidatePostgresExporterCollectors([]string{"wal"}, []string{"locks", "wal"}, v360)
		require.EqualError(t, err, `collector "wal" is both enabled and disabled`)
	})

	t.Run("CollectorNewerThanExporter", func(t *testing.T) {
		t.Parallel()

		err := ValidatePostgresExporterCollectors([]string{"stat_io"}, nil, version.MustParse("3.5.0"))
		require.EqualError(t, err, `collector "stat_io" needs pmm-agent 3.6.0 or newer, but the pmm-agent is 3.5.0`)

		err = ValidatePostgresExporterCollectors([]string{"wal"}, nil, version.MustParse("2.40.1"))
		require.EqualError(t, err, `collector "wal" needs pmm-agent 2.41.0-0 or newer, but the pmm-agent is 2.40.1`)
	})

	t.Run("UnknownVersion", func(t *testing.T) {
		t.Parallel()

		err := ValidatePostgresExporterCollectors([]string{"stat_io"}, nil, nil)
		require.NoError(t, err)
	})
}
