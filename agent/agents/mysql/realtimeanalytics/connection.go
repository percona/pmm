// Copyright (C) 2023 Percona LLC
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

package realtimeanalytics

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/percona/pmm/agent/tlshelpers"
)

const (
	mysqlQueryTimeout = 5 * time.Second
)

// createConnection opens and pings a MySQL connection, returning it with the DSN's host:port.
func createConnection(ctx context.Context, dsn string, files map[string]string, tlsSkipVerify bool) (*sql.DB, string, error) {
	if files != nil {
		err := tlshelpers.RegisterMySQLCerts(files, tlsSkipVerify)
		if err != nil {
			return nil, "", fmt.Errorf("failed to register MySQL TLS certificates: %w", err)
		}
	}

	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse MySQL DSN: %w", err)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	// Collections never overlap, so one long-lived connection is reused across cycles.
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	pingCtx, cancel := context.WithTimeout(ctx, mysqlQueryTimeout)
	defer cancel()

	err = db.PingContext(pingCtx)
	if err != nil {
		_ = db.Close()
		return nil, "", fmt.Errorf("failed to ping MySQL at %s: %w", cfg.Addr, err)
	}

	return db, cfg.Addr, nil
}
