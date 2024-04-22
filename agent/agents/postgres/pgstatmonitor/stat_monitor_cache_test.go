// Copyright (C) 2024 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pgstatmonitor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/agent/utils/truncate"
)

func TestPGStatMonitorStructs(t *testing.T) {
	sqlDB := tests.OpenTestPostgreSQL(t)
	defer sqlDB.Close() //nolint:errcheck
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	engineVersion := tests.PostgreSQLVersion(t, sqlDB)
	if !supportedVersion(engineVersion) || !extensionExists(db) {
		t.Skip()
	}

	_, err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_stat_monitor SCHEMA public")
	assert.NoError(t, err)

	defer func() {
		_, err = db.Exec("DROP EXTENSION pg_stat_monitor")
		assert.NoError(t, err)
	}()

	m := setup(t, db, false, false)
	settings, err := m.getSettings()
	assert.NoError(t, err)
	normalizedQuery, err := settings.getNormalizedQueryValue()
	assert.NoError(t, err)

	current, cache, err := m.monitorCache.getStatMonitorExtended(context.TODO(), db.Querier, normalizedQuery, truncate.GetDefaultMaxQueryLength())

	require.NoError(t, err)
	require.NotNil(t, current)
	require.NotNil(t, cache)
}
