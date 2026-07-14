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
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/151" // register database/sql driver
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup() *sqlx.DB {
	cmdStr := `docker exec pmm-clickhouse-test clickhouse client -n --query='DROP DATABASE IF EXISTS pmm_test_parts; CREATE DATABASE pmm_test_parts;'`
	if out, err := exec.Command("/bin/sh", "-c", cmdStr).Output(); err != nil {
		log.Printf("Docker create db: %v, %v", out, err)
	}

	dsn, ok := os.LookupEnv("QANAPI_DSN_TEST")
	dsn = strings.Replace(dsn, "?database=pmm_test", "?database=pmm_test_parts", 1)
	if !ok {
		dsn = "clickhouse://127.0.0.1:19000?database=pmm_test_parts"
	}
	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		log.Fatal("Connection: ", err)
	}
	err = runMigrations(dsn)
	if err != nil {
		log.Fatal("Migration: ", err)
	}

	cmdStr = `cat fixture/metrics.part_*.json | docker exec -i pmm-clickhouse-test clickhouse client -d pmm_test_parts --query="INSERT INTO metrics FORMAT JSONEachRow"`
	if out, err := exec.Command("/bin/sh", "-c", cmdStr).Output(); err != nil {
		log.Fatalf("Docker load fixture: %v, %v", out, err)
	}

	return db
}

func cleanup() {
	cleanupDatabases := []string{"pmm_test_parts", "pmm_created_db"}
	for _, database := range cleanupDatabases {
		cmdStr := fmt.Sprintf(`docker exec pmm-clickhouse-test clickhouse client --query='DROP DATABASE IF EXISTS %s;'`, database)
		if out, err := exec.Command("/bin/sh", "-c", cmdStr).Output(); err != nil { //nolint:gosec
			log.Fatalf("Docker drop db: %v, %v", out, err)
		}
	}
}

func TestDropOldPartition(t *testing.T) {
	db := setup()

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
			query)
		require.NoError(t, err, "Unexpected error in selecting metrics partition")
		require.Equal(t, 2, len(partitions), "No one partition were truncated. Partition %+v, days %d", partitions, days)
		assert.Equal(t, "20190101", partitions[0], "Newest partition was not truncated")
		assert.Equal(t, "20190102", partitions[1], "Oldest partition was not truncated")
	})

	t.Run("delete one day old partition", func(t *testing.T) {
		partitions := []string{}
		days := daysNewestPartition
		DropOldPartition(db, "pmm_test_parts", days)
		err := db.Select(
			&partitions,
			query)
		require.NoError(t, err, "Unexpected error in selecting metrics partition")
		require.Equal(t, 1, len(partitions), "Only one partition should left. Partition %+v, days %d", partitions, days)
		assert.Equal(t, "20190102", partitions[0], "Newest partition was not truncated")
	})
	cleanup()
}

func TestCreateDbIfNotExists(t *testing.T) {
	t.Run("connect to db that doesnt exist", func(t *testing.T) {
		dsn, ok := os.LookupEnv("QANAPI_DSN_TEST")

		dsn = strings.Replace(dsn, "?database=pmm_test", "?database=pmm_created_db", 1)
		if !ok {
			dsn = "clickhouse://127.0.0.1:19000?database=pmm_created_db"
		}

		db := createDB(dsn)

		require.Equal(t, db, nil, "Check connection after we create database")
	})
}
