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
	_, err := db.ExecContext(ctx, stmt)
	if err != nil {
		return fmt.Errorf("%s: %w", stmt, err)
	}
	return nil
}
