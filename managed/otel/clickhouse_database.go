// Copyright (C) 2026 Percona LLC
//
// Licensed under the GNU Affero General Public License, Version 3 or later.

package otel

import (
	"context"
	"database/sql"
	"fmt"
)

// ensureOtelDatabase creates the otel database using the same pattern as qan-api2/db.go createDB.
func ensureOtelDatabase(ctx context.Context, db *sql.DB) error {
	clusterName := clickhouseClusterName()
	var stmt string
	if clusterName != "" {
		stmt = fmt.Sprintf(
			`CREATE DATABASE IF NOT EXISTS otel ON CLUSTER "%s" ENGINE = Replicated('/clickhouse/databases/{uuid}', '{shard}', '{replica}')`,
			clusterName,
		)
	} else {
		stmt = `CREATE DATABASE IF NOT EXISTS otel ENGINE = Atomic`
	}
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("%s: %w", stmt, err)
	}
	return nil
}
