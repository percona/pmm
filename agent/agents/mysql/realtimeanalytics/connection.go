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
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/percona/pmm/agent/tlshelpers"
)

const (
	// Timeout for MySQL queries and connection.
	mysqlQueryTimeout = 5 * time.Second
)

// createConnection opens a connection to MySQL and verifies it with a ping.
// It returns the *sql.DB, the instance address(host:port) parsed from the DSN
// and an error if connection can't be established.
func createConnection(ctx context.Context, dsn string, files map[string]string, tlsSkipVerify bool) (*sql.DB, string, error) {
	if files != nil {
		if err := tlshelpers.RegisterMySQLCerts(files, tlsSkipVerify); err != nil {
			return nil, "", err
		}
	}

	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, "", err
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, "", err
	}

	// The collector runs one query per interval, so a single long-lived connection
	// is kept open and reused across collection cycles (no maximum lifetime).
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	pingCtx, cancel := context.WithTimeout(ctx, mysqlQueryTimeout)
	defer cancel()

	if err = db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, "", err
	}

	return db, cfg.Addr, nil
}
