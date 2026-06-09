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

package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2" // register database/sql driver
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/qan-api2/migrations"
)

func setupDB(t *testing.T) *sqlx.DB {
	t.Helper()

	cmdStr := `docker exec pmm-clickhouse-test clickhouse client -n --password=clickhouse --query='DROP DATABASE IF EXISTS pmm_test_parts; CREATE DATABASE pmm_test_parts;'`
	out, err := exec.CommandContext(t.Context(), "/bin/sh", "-c", cmdStr).Output()
	require.NoError(t, err, "Docker create db: %v", out)

	dsn, ok := os.LookupEnv("QANAPI_DSN_TEST")
	dsn = strings.Replace(dsn, "/pmm_test", "/pmm_test_parts", 1)
	if !ok {
		dsn = "clickhouse://default:clickhouse@127.0.0.1:19000/pmm_test_parts"
	}
	db, err := sqlx.Connect("clickhouse", dsn)
	require.NoError(t, err, "Connection failed")
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	data := map[string]any{
		"engine": migrations.GetEngine(false),
	}
	err = migrations.Run(dsn, data, false, "")
	require.NoError(t, err, "Migration failed")

	cmdStr = `cat fixture/metrics.part_*.json | docker exec -i pmm-clickhouse-test clickhouse client -d pmm_test_parts --password=clickhouse --query="INSERT INTO metrics FORMAT JSONEachRow"`
	out, err = exec.CommandContext(t.Context(), "/bin/sh", "-c", cmdStr).Output()
	require.NoError(t, err, "Docker load fixture: %v", out)

	return db
}

func cleanupDB(t *testing.T, dbName string) {
	t.Helper()

	cmdStr := fmt.Sprintf(`docker exec pmm-clickhouse-test clickhouse client --password=clickhouse --query='DROP DATABASE IF EXISTS %s;'`, dbName)
	out, err := exec.CommandContext(context.Background(), "/bin/sh", "-c", cmdStr).Output() //nolint:gosec
	assert.NoError(t, err, "Docker drop db: %v", out)
}

func TestDropOldPartition(t *testing.T) {
	t.Parallel()
	db := setupDB(t)
	t.Cleanup(func() {
		cleanupDB(t, "pmm_test_parts")
	})

	const query = `SELECT DISTINCT partition FROM system.parts WHERE database = 'pmm_test_parts' and visible = 1 ORDER BY partition`

	start := time.Now()
	// fixtures have two partition 20190101 and 20190102
	// here calculates how many days old partitions are.
	end := time.Date(2019, 1, 2, 0, 0, 0, 0, time.UTC)
	difference := end.Sub(start)
	daysNewestPartition := uint(math.Abs(difference.Hours()) / 24)

	t.Run("no so old partition", func(t *testing.T) {
		partitions := []string{}
		days := daysNewestPartition + 1
		DropOldPartition(db, "pmm_test_parts", days)
		err := db.Select(
			&partitions,
			query,
		)
		require.NoError(t, err, "Unexpected error in selecting metrics partition")
		require.Len(t, partitions, 2, "No one partition were truncated. Partition %+v, days %d", partitions, days)
		assert.Equal(t, "20190101", partitions[0], "Newest partition was not truncated")
		assert.Equal(t, "20190102", partitions[1], "Oldest partition was not truncated")
	})

	t.Run("delete one day old partition", func(t *testing.T) {
		partitions := []string{}
		days := daysNewestPartition
		DropOldPartition(db, "pmm_test_parts", days)
		err := db.Select(
			&partitions,
			query,
		)
		require.NoError(t, err, "Unexpected error in selecting metrics partition")
		require.Len(t, partitions, 1, "Only one partition should left. Partition %+v, days %d", partitions, days)
		assert.Equal(t, "20190102", partitions[0], "Newest partition was not truncated")
	})
}

func TestCreateDbIfNotExists(t *testing.T) {
	t.Parallel()

	t.Cleanup(func() {
		cleanupDB(t, "pmm_created_db")
	})

	t.Run("connect to db that is absent", func(t *testing.T) {
		dsn, ok := os.LookupEnv("QANAPI_DSN_TEST")

		dsn = strings.Replace(dsn, "/pmm_test", "/pmm_created_db", 1)
		if !ok {
			dsn = "clickhouse://default:clickhouse@127.0.0.1:19000/pmm_created_db"
		}

		err := createDB(dsn, "")

		require.NoError(t, err, "Check connection after we create database")
	})
}
