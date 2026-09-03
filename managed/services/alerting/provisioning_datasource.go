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

package alerting

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// A provisioned alert rule has to name its datasource by UID, and the UID of PMM's "Metrics"
// datasource is not the same on every server.
//
// Grafana derives a UID for a provisioned datasource from its name, but only when it first creates
// the row; updating a datasource leaves whatever UID it was born with. That derivation arrived in
// Grafana 8.3.4 - before it, a datasource created without a UID got a random one. PMM shipped older
// Grafanas until PMM 2.28.0, so every server first installed before that stores a random UID:
// percona/pmm-server:2.26.0 has "b_nGyianz" where 2.28.0 has the derived value. Nothing heals it
// afterwards - the v2 SQLite-to-PostgreSQL migrator copies the row verbatim, provisioning updates a
// datasource by name and leaves its UID alone, and the v2-to-v3 upgrade carries /srv forward.
// Re-pinning uid: in datasources.yml is not the answer either: that path does write the UID, so it
// would rewrite stored UIDs and orphan every dashboard and user alert rule naming the old one,
// which is why PMM-13943 was reverted before it shipped.
//
// The UID is therefore read from Grafana's own database, which is the source of truth, and derived
// only when Grafana has demonstrably not created the datasource yet - the fresh-install case, where
// the derivation is what Grafana is about to compute anyway. When the database cannot be consulted
// at all nothing is rendered, because a rule naming a UID that does not exist reports healthy while
// querying nothing, and no rules at all is a failure someone can see.
const (
	metricsDatasourceName = "Metrics"

	// The query timeout matches the one PMM already uses for its Grafana database queries in
	// telemetry.
	datasourceQueryTimeout = 5 * time.Second

	// How long a UID read from the database is trusted before it is confirmed again. One query per
	// reconcile is negligible against a database Grafana itself works far harder, and re-reading is
	// what lets a datasource deleted and recreated by hand be picked up without a restart.
	datasourceRecheckInterval = reconcileInterval

	// Postgres undefined_table. Grafana creates its own tables when it first starts, so this is the
	// error a query gets on a database Grafana has never run against.
	undefinedTableCode = "42P01"
)

// grafanaDBFallback describes where Grafana's database lives when the deployment has not said.
// This is the bundled case: Grafana shares PMM's own PostgreSQL server, and the database and user
// are created by the initialization ansible role.
type grafanaDBFallback struct {
	// Addr is PMM's PostgreSQL address, host:port.
	Addr string
	// The encoded query string PMM uses for its own connection.
	SSLParams string
}

const (
	bundledGrafanaDB       = "grafana"
	bundledGrafanaUser     = "grafana"
	bundledGrafanaPassword = "grafana"
)

// grafanaDatasourceDSN works out how to reach Grafana's database.
//
// The order matters. Since PMM 3.2.0 an external Grafana database is configured with the discrete
// GF_DATABASE_* variables, and that is what the pmm-ha Helm chart sets - with a user that is not
// necessarily "grafana". GF_DATABASE_URL is the older single-URL form. Only when neither is present
// is the database the bundled one, which is exactly the condition under which the ansible role
// creates it, so the branches cannot overlap.
func grafanaDatasourceDSN(fallback grafanaDBFallback) (string, error) {
	if host := os.Getenv("GF_DATABASE_HOST"); host != "" {
		// Grafana overrides one setting per variable and takes the rest from grafana.ini, so every
		// field left unset here has to fall back to what that file ships, not to an empty value.
		dsn := url.URL{
			Scheme: "postgres",
			User: url.UserPassword(
				envOrDefault("GF_DATABASE_USER", bundledGrafanaUser),
				envOrDefault("GF_DATABASE_PASSWORD", bundledGrafanaPassword),
			),
			Host: host,
			Path: envOrDefault("GF_DATABASE_NAME", bundledGrafanaDB),
		}

		query := make(url.Values)
		query.Set("sslmode", envOrDefault("GF_DATABASE_SSL_MODE", "disable"))
		// verify-ca and verify-full need the material as well, and PMM documents all three paths.
		for env, param := range map[string]string{
			"GF_DATABASE_CA_CERT_PATH":     "sslrootcert",
			"GF_DATABASE_CLIENT_CERT_PATH": "sslcert",
			"GF_DATABASE_CLIENT_KEY_PATH":  "sslkey",
		} {
			if value := os.Getenv(env); value != "" {
				query.Set(param, value)
			}
		}
		dsn.RawQuery = query.Encode()

		return dsn.String(), nil
	}

	if raw := os.Getenv("GF_DATABASE_URL"); raw != "" {
		_, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("failed to parse GF_DATABASE_URL: %w", err)
		}
		return raw, nil
	}

	if fallback.Addr == "" {
		return "", errors.New("no Grafana database address available")
	}

	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(bundledGrafanaUser, bundledGrafanaPassword),
		Host:     fallback.Addr,
		Path:     bundledGrafanaDB,
		RawQuery: fallback.SSLParams,
	}

	return dsn.String(), nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

// deriveMetricsDatasourceUID reproduces the UID Grafana generates for a provisioned datasource that
// does not declare one. Keeping the algorithm here rather than hard-coding its result documents why
// the value looks the way it does, and a unit test pins it to the value a real server reports.
func deriveMetricsDatasourceUID() string {
	sum := sha256.Sum256([]byte(metricsDatasourceName))
	return strings.ToUpper(fmt.Sprintf("P%x", sum[:8]))
}

// queryMetricsDatasourceUID reads the UID Grafana actually stores. Returns sql.ErrNoRows when
// Grafana has not provisioned its datasources yet, which is normal on a first boot.
func queryMetricsDatasourceUID(ctx context.Context, db *sql.DB) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, datasourceQueryTimeout)
	defer cancel()

	var uid string
	err := db.QueryRowContext(ctx,
		"SELECT uid FROM data_source WHERE org_id = $1 AND name = $2",
		provisionedOrgID, metricsDatasourceName).Scan(&uid)
	if err != nil {
		return "", err
	}
	if uid == "" {
		return "", errors.New("stored datasource UID is empty")
	}

	return uid, nil
}

// errDatasourceUnresolved reports that the Metrics datasource UID could not be determined, as
// opposed to the fresh-install case where deriving it is provably right. Callers tell the two apart
// with errors.Is rather than by matching on log text.
var errDatasourceUnresolved = errors.New("could not determine the Metrics datasource UID")

// grafanaReader answers the two questions PMM has to ask Grafana's own database before it can write
// a provisioning file: what the Metrics datasource UID is, and whether the rule UIDs PMM claims are
// still PMM's to claim.
type grafanaReader struct {
	dsn string
	l   *logrus.Entry

	m  sync.Mutex
	db *sql.DB
	// uid is the last answer, and checkedAt when the database last confirmed it. A zero checkedAt
	// means the answer is derived rather than read.
	uid       string
	checkedAt time.Time
	lastErr   error
}

func newGrafanaReader(dsn string, l *logrus.Entry) *grafanaReader {
	return &grafanaReader{dsn: dsn, l: l}
}

// Resolve returns the UID to render into the provisioning file, or an error when it cannot be
// determined at all.
//
// A UID read from the database is trusted until the recheck window expires, so that a datasource an
// administrator deleted and recreated is picked up without restarting pmm-managed, at the cost of
// one negligible query per window.
//
// A missing row, or a missing table, means Grafana has never created a datasource in this database:
// the first is a Grafana that has not provisioned yet, the second a Grafana that has never started
// at all, which is the state a fresh container is in when the rules are first written. Deriving is
// correct by construction there - it is what Grafana is about to compute - and the answer is not
// cached, so the real row is picked up the moment it exists.
//
// Anything else leaves the UID genuinely unknown, and guessing it is wrong on every server whose
// datasource predates Grafana 8.3.4. Since a rule naming a UID that does not exist reports healthy
// while querying nothing, this fails closed instead: the caller must not write a file.
func (r *grafanaReader) ResolveDatasourceUID(ctx context.Context) (string, error) {
	r.m.Lock()
	defer r.m.Unlock()

	if !r.checkedAt.IsZero() && time.Since(r.checkedAt) < datasourceRecheckInterval {
		return r.uid, nil
	}

	uid, err := r.queryLocked(ctx)
	switch {
	case err == nil:
		if r.uid != uid {
			r.l.Infof("Resolved %s datasource UID %q from Grafana's database.", metricsDatasourceName, uid)
		}
		r.uid, r.checkedAt, r.lastErr = uid, time.Now(), nil
		return uid, nil

	case isDatasourceAbsent(err):
		r.l.Debugf("Grafana has not created the %s datasource yet, using the derived UID.", metricsDatasourceName)
		r.uid, r.checkedAt = deriveMetricsDatasourceUID(), time.Time{}
		return r.uid, nil

	default:
		// Log once per distinct failure so a permanently unreachable database does not fill the log.
		if r.lastErr == nil || r.lastErr.Error() != err.Error() {
			r.l.Errorf("Cannot read the %s datasource UID from Grafana's database, so built-in alert rules "+
				"are not being provisioned. Rendering them with a guessed UID would leave them reporting "+
				"healthy while querying nothing. Fix the connection and they will be provisioned "+
				"automatically: %s.", metricsDatasourceName, err)
		}
		r.lastErr = err
		return "", fmt.Errorf("%w: %w", errDatasourceUnresolved, err)
	}
}

// isDatasourceAbsent reports whether err means Grafana has not created the datasource yet, rather
// than that the database could not be consulted. Either the row is missing from a schema Grafana
// has already built, or the table itself is not there because Grafana has never started against
// this database - the state of a first boot, before Grafana has run its own migrations.
//
// Only the one code qualifies. A missing database (3D000) means pmm-managed is pointed somewhere
// Grafana is not, and insufficient privilege (42501) means the table exists but is unreadable;
// both leave a real UID unread, so both must fail closed.
func isDatasourceAbsent(err error) bool {
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}

	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == undefinedTableCode
}

func (r *grafanaReader) queryLocked(ctx context.Context) (string, error) {
	db, err := r.openLocked()
	if err != nil {
		return "", err
	}

	return queryMetricsDatasourceUID(ctx, db)
}

// ConflictingRules reports which of the given UIDs Grafana records as belonging to someone other
// than file provisioning, mapped to the owner it records ("api", or "" for a rule a user made in
// the interface).
//
// PMM's rule UIDs are frozen, which makes them squattable: a UID is only PMM's while PMM provisions
// it, so a bundle that is switched off - or a server that predates this feature - leaves them free
// for anyone to take. Writing a file that claims one anyway is not a small mistake. Measured against
// Grafana 12.4.5:
//
//   - a rule under provenance "api" cannot be updated or deleted from a file, and Grafana treats
//     that refusal at startup as fatal: it exits, taking PMM's whole interface down with it;
//   - a rule a user made in the interface carries no provenance at all, so Grafana raises nothing -
//     it lets the file silently overwrite the rule, or delete it outright.
//
// So the same query guards against two different harms: an outage, and quietly destroying something
// a user made. Both are avoided the same way - by leaving a UID that is not ours alone.
func (r *grafanaReader) ConflictingRules(ctx context.Context, uids []string) (map[string]string, error) {
	if len(uids) == 0 {
		return map[string]string{}, nil
	}

	r.m.Lock()
	defer r.m.Unlock()

	db, err := r.openLocked()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, datasourceQueryTimeout)
	defer cancel()

	// Left join rather than reading provenance_type alone: a rule made in the interface has no row
	// there, and that is exactly the case that ends in silent data loss.
	rows, err := db.QueryContext(ctx, `
		SELECT a.uid, COALESCE(p.provenance, '')
		  FROM alert_rule a
		  LEFT JOIN provenance_type p
		    ON p.record_key = a.uid AND p.record_type = 'alertRule' AND p.org_id = a.org_id
		 WHERE a.org_id = $1 AND a.uid = ANY($2) AND COALESCE(p.provenance, '') <> 'file'`,
		provisionedOrgID, pq.Array(uids))
	if err != nil {
		if isDatasourceAbsent(err) {
			// Grafana has not created its alerting tables yet, so nothing can conflict.
			return map[string]string{}, nil
		}
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	conflicts := make(map[string]string)
	for rows.Next() {
		var uid, provenance string
		err = rows.Scan(&uid, &provenance)
		if err != nil {
			return nil, err
		}
		conflicts[uid] = provenance
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return conflicts, nil
}

// openLocked returns the connection, opening it on first use. Called with m held.
func (r *grafanaReader) openLocked() (*sql.DB, error) {
	if r.dsn == "" {
		return nil, errors.New("no Grafana database DSN configured")
	}

	if r.db == nil {
		db, err := sql.Open("postgres", r.dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open Grafana database: %w", err)
		}
		// One short-lived connection is enough; this runs at startup and on a slow reconcile tick.
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(time.Minute)
		r.db = db
	}

	return r.db, nil
}

// Close releases the connection to Grafana's database.
func (r *grafanaReader) Close() error {
	r.m.Lock()
	defer r.m.Unlock()

	if r.db == nil {
		return nil
	}
	err := r.db.Close()
	r.db = nil
	return err
}
