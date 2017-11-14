// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package models

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3" // register SQL driver
	"gopkg.in/reform.v1"
)

var databaseSchema = []string{
	`CREATE TABLE schema_migrations (
		id integer PRIMARY KEY AUTOINCREMENT
	)`,
	`INSERT INTO schema_migrations DEFAULT VALUES`,

	`CREATE TABLE nodes (
		id integer PRIMARY KEY AUTOINCREMENT,
		type varchar NOT NULL,
		name varchar NOT NULL,

		region varchar NOT NULL, -- NOT NULL for unique index below

		UNIQUE (type, name, region)
	)`,

	`CREATE TABLE services (
		id integer PRIMARY KEY AUTOINCREMENT,
		type varchar NOT NULL,
		node_id integer NOT NULL,

		address varchar,
		port integer,
		engine varchar,
		engine_version varchar,

		FOREIGN KEY (node_id) REFERENCES nodes (id)
	)`,
}

func OpenDB(file string, logf reform.Printf) (*sql.DB, error) {
	// https://sqlite.org/uri.html
	// https://godoc.org/github.com/mattn/go-sqlite3#SQLiteDriver.Open
	dsn := fmt.Sprintf("file:%s?_foreign_keys=1", file)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// Use single connection so "PRAGMA foreign_keys" is always enforced, and to prevent data corruption
	// if SQLite3 is not build in thread-safe mode.
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	if _, err = db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}

	var count int
	if err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err == nil && count > 0 {
		return db, nil
	}

	for _, q := range databaseSchema {
		q = strings.TrimSpace(q)
		logf("\n%s\n", q)
		if _, err = db.Exec(q); err != nil {
			return nil, err
		}
	}

	return db, nil
}
