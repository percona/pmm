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

// Package telemetry provides telemetry functionality.
package telemetry

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrafanaSqliteDatasource(t *testing.T) {
	t.Parallel()
	logger := logrus.StandardLogger()
	logger.SetLevel(logrus.DebugLevel)
	logEntry := logrus.NewEntry(logger)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
	})

	config := &Config{
		ID:      "test",
		Source:  "GRAFANADB_SELECT",
		Query:   "count(*) AS total from user",
		Summary: "Simple query",
		Data: []ConfigData{
			{
				MetricName: "total_users_in_database",
				Column:     "total",
			},
		},
	}

	t.Run("get metrics from db", func(t *testing.T) {
		t.Parallel()
		databaseFile, err := filepath.Abs("../../testdata/telemetry/grafana_sqlite.db")
		require.NoError(t, err)

		conf := &DSGrafanaSqliteDB{
			Enabled: true,
			Timeout: time.Second * 10,
			DBFile:  databaseFile,
		}
		grafanaDB := NewDataSourceGrafanaSqliteDB(*conf, logEntry)

		err = grafanaDB.Init(ctx)
		require.NoError(t, err)

		metrics, err := grafanaDB.FetchMetrics(ctx, *config)
		require.NoError(t, err)
		assert.Equal(t, len(metrics), 1)

		err = grafanaDB.Dispose(ctx)
		require.NoError(t, err)

		serviceMetric := metrics[0]
		assert.Equal(t, serviceMetric.Key, "total_users_in_database")
		assert.Equal(t, serviceMetric.Value, "1")
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()
		conf := &DSGrafanaSqliteDB{
			Enabled: true,
			Timeout: time.Second * 10,
			DBFile:  "/invalid/path/",
		}

		grafanaDB := NewDataSourceGrafanaSqliteDB(*conf, logEntry)

		err := grafanaDB.Init(ctx)
		assert.Error(t, err, "no such file or directory")

		metrics, err := grafanaDB.FetchMetrics(ctx, *config)
		assert.Error(t, err, "temporary grafana database is not initialized")
		assert.Nil(t, metrics)
	})
}
