// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package otel

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2" // register clickhouse driver for cluster checks
	"github.com/sirupsen/logrus"
)

// IsClickhouseClusterReady is copied from qan-api2/migrations.IsClickhouseClusterReady (system.clusters).
func IsClickhouseClusterReady(ctx context.Context, dsn string, clusterName string) (bool, error) {
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return false, err
	}
	defer db.Close() //nolint:errcheck

	sql := "SELECT sum(is_local = 0) AS remote_hosts FROM system.clusters"
	args := []any{}
	if clusterName != "" {
		sql += " WHERE cluster = ?"
		args = append(args, clusterName)
	}
	sql += " FORMAT TabSeparated"

	row := db.QueryRowContext(ctx, sql, args...)
	var remoteHosts int
	err = row.Scan(&remoteHosts)
	if err != nil {
		return false, err
	}
	return remoteHosts > 0, nil
}

// WaitForClickhouseClusterReady blocks until the cluster reports remote replicas, like qan-api2/db.NewDB.
// No-op when PMM_CLICKHOUSE_IS_CLUSTER is unset/false.
func WaitForClickhouseClusterReady(ctx context.Context, dsn string) {
	if !clickhouseIsCluster() {
		return
	}
	l := logrus.WithField("component", "otel_clickhouse")
	name := clickhouseClusterName()
	for {
		ready, err := IsClickhouseClusterReady(ctx, dsn, name)
		switch {
		case err != nil:
			l.WithError(err).Warn("ClickHouse cluster readiness check failed; retrying")
		case ready:
			l.Info("ClickHouse cluster is ready for OTEL DDL")
			return
		default:
			l.Info("Waiting for ClickHouse cluster to be ready (system.clusters, remote_hosts > 0)...")
		}
		select {
		case <-ctx.Done():
			l.Warn("Context done while waiting for ClickHouse cluster")
			return
		case <-time.After(time.Second):
		}
	}
}
