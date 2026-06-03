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

// Package clickhouse owns the lifecycle (schema + retention TTL) of the OpenTelemetry logs and traces
// tables in ClickHouse. It is deliberately part of pmm-managed and independent of qan-api2: qan-api2
// is the Query Analytics component and must not be aware of logging or tracing.
package clickhouse

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"text/template"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse" // register golang-migrate clickhouse driver
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/utils/envvars"
)

//go:embed migrations/sql/*.sql
var migrationsFS embed.FS

const (
	migrationsTable         = "logs_schema_migrations"
	engineSimple            = "MergeTree"
	engineCluster           = "ReplicatedMergeTree"
	migrationsEngineCluster = "ReplicatedMergeTree ORDER BY version"

	migrateRetryInterval = 5 * time.Second
	migrateMaxAttempts   = 24 // ~2 minutes, enough to cover ClickHouse warm-up.
)

// errNotReady marks a transient failure (ClickHouse unreachable) that is worth retrying, as opposed to
// a permanent migration error (e.g. malformed SQL) that retrying cannot fix.
var errNotReady = errors.New("clickhouse is not ready")

// Service owns the schema and TTL of the pmm.logs / pmm.traces tables.
type Service struct {
	db         *sql.DB
	migrateDSN string
	database   string
	isCluster  bool
	l          *logrus.Entry
}

// New returns a ClickHouse logs/traces lifecycle service. db is reused for DDL (TTL); golang-migrate
// opens its own connection from a DSN built out of the same connection parameters.
func New(db *sql.DB, addr, database, user, password string) *Service {
	isCluster, _ := strconv.ParseBool(envvars.GetEnv("PMM_CLICKHOUSE_IS_CLUSTER", "false"))
	dsn := url.URL{
		Scheme: "clickhouse",
		User:   url.UserPassword(user, password),
		Host:   addr,
		Path:   database,
	}
	return &Service{
		db:         db,
		migrateDSN: dsn.String(),
		database:   database,
		isCluster:  isCluster,
		l:          logrus.WithField("component", "clickhouse"),
	}
}

// Bootstrap brings the logs/traces schema up to date. It retries while ClickHouse is unreachable
// (bounded by migrateMaxAttempts) but fails fast on a permanent migration error, and returns when the
// schema is up to date, when it gives up, or when ctx is cancelled.
func (s *Service) Bootstrap(ctx context.Context) error {
	for attempt := 1; ; attempt++ {
		err := s.Migrate()
		switch {
		case err == nil:
			s.l.Info("logs/traces schema is up to date")
			return nil
		case !errors.Is(err, errNotReady):
			return errors.Wrap(err, "logs/traces migrations failed")
		case attempt >= migrateMaxAttempts:
			return errors.Wrapf(err, "ClickHouse still not ready after %d attempts", attempt)
		}

		s.l.Warnf("ClickHouse not ready (attempt %d/%d), retrying in %s: %s", attempt, migrateMaxAttempts, migrateRetryInterval, err)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(migrateRetryInterval):
		}
	}
}

// Migrate applies the embedded logs/traces migrations using a dedicated migrations table so it never
// collides with qan-api2's schema_migrations on the same database. A connect-time failure is returned
// wrapped in errNotReady (transient); a failed migration is recovered from a dirty state once, then
// returned as a permanent error.
func (s *Service) Migrate() error {
	data := map[string]any{"engine": s.engine()}

	entries, err := migrationsFS.ReadDir("migrations/sql")
	if err != nil {
		return errors.WithStack(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}

	res := bindata.Resource(names, func(name string) ([]byte, error) {
		b, err := migrationsFS.ReadFile("migrations/sql/" + name)
		if err != nil {
			return nil, err
		}
		tmpl, err := template.New(name).Parse(string(b))
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	})

	src, err := bindata.WithInstance(res)
	if err != nil {
		return errors.WithStack(err)
	}

	dsn, err := s.dsnForMigrate()
	if err != nil {
		return err
	}

	// The clickhouse migrate driver connects in Open, so a failure here means ClickHouse is not yet
	// reachable — transient, worth retrying.
	m, err := migrate.NewWithSourceInstance("go-bindata", src, dsn)
	if err != nil {
		return fmt.Errorf("%w: %w", errNotReady, err)
	}
	defer m.Close() //nolint:errcheck

	err = m.Up()
	if err == nil || errors.Is(err, migrate.ErrNoChange) {
		return nil
	}

	// Recover from a dirty migration state by forcing back one version and retrying (PMM-14305).
	var errDirty *migrate.ErrDirty
	if errors.As(err, &errDirty) {
		s.l.Warnf("Migration %d left the schema dirty, attempting recovery...", errDirty.Version)
		ver := errDirty.Version - 1
		if ver == 0 {
			ver = -1 // golang-migrate's "no migration applied" sentinel.
		}
		if ferr := m.Force(ver); ferr != nil {
			return errors.Wrapf(ferr, "can't force migration %d", ver)
		}
		if uerr := m.Up(); uerr != nil && !errors.Is(uerr, migrate.ErrNoChange) && !errors.Is(uerr, io.EOF) {
			return errors.WithStack(uerr)
		}
		return nil
	}

	return errors.WithStack(err)
}

// ApplyTTL sets the retention TTL on the logs and traces tables. In cluster mode the DDL is replicated
// automatically by the Replicated `pmm` database engine, so no ON CLUSTER clause is needed.
func (s *Service) ApplyTTL(retention time.Duration) error {
	days := max(int(retention.Hours())/24, 1) //nolint:mnd
	stmts := []string{
		fmt.Sprintf("ALTER TABLE %s.logs MODIFY TTL TimestampTime + INTERVAL %d DAY", s.database, days),
		fmt.Sprintf("ALTER TABLE %s.traces MODIFY TTL toDateTime(Timestamp) + INTERVAL %d DAY", s.database, days),
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return errors.Wrapf(err, "failed to apply TTL (%q)", stmt)
		}
		s.l.Infof("Applied retention TTL: %s", stmt)
	}
	return nil
}

func (s *Service) engine() string {
	if s.isCluster {
		return engineCluster
	}
	return engineSimple
}

func (s *Service) dsnForMigrate() (string, error) {
	u, err := url.Parse(s.migrateDSN)
	if err != nil {
		return "", errors.WithStack(err)
	}
	q := u.Query()
	q.Set("x-migrations-table", migrationsTable)
	if s.isCluster {
		q.Set("x-migrations-table-engine", migrationsEngineCluster)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
