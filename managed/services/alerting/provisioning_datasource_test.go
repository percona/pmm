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
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeriveMetricsDatasourceUID pins the derivation to the value a real PMM server reports, so a
// change to it cannot slip through: every provisioned rule on a fresh install names this UID, and a
// wrong one leaves the rules unable to query anything.
func TestDeriveMetricsDatasourceUID(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "PA58DA793C7250F1B", deriveMetricsDatasourceUID())
}

// Deliberately not parallel: the subtests use t.Setenv, which panics in a parallel test.
func TestGrafanaDatasourceDSN(t *testing.T) {
	fallback := grafanaDBFallback{Addr: "127.0.0.1:5432", SSLParams: "sslmode=disable"}

	t.Run("prefers the discrete GF_DATABASE variables", func(t *testing.T) {
		// This is what the pmm-ha Helm chart sets, and its user is not "grafana".
		t.Setenv("GF_DATABASE_HOST", "pg.example:5432")
		t.Setenv("GF_DATABASE_NAME", "grafana")
		t.Setenv("GF_DATABASE_USER", "gfuser")
		t.Setenv("GF_DATABASE_PASSWORD", "secret")
		t.Setenv("GF_DATABASE_SSL_MODE", "require")

		dsn, err := grafanaDatasourceDSN(fallback)
		require.NoError(t, err)
		assert.Equal(t, "postgres://gfuser:secret@pg.example:5432/grafana?sslmode=require", dsn)
	})

	t.Run("falls back to what grafana.ini ships for the variables left unset", func(t *testing.T) {
		// Grafana overrides one setting per variable, so a deployment that sets only the host still
		// authenticates with the credentials in grafana.ini. Defaulting the password to empty here
		// would fail against a database Grafana itself reaches happily.
		t.Setenv("GF_DATABASE_HOST", "pg.example:5432")

		dsn, err := grafanaDatasourceDSN(fallback)
		require.NoError(t, err)
		assert.Equal(t, "postgres://grafana:grafana@pg.example:5432/grafana?sslmode=disable", dsn)
	})

	t.Run("carries the TLS material for verify modes", func(t *testing.T) {
		// sslmode=verify-full without the certificate paths cannot connect, and PMM documents all
		// three paths alongside the mode.
		t.Setenv("GF_DATABASE_HOST", "pg.example:5432")
		t.Setenv("GF_DATABASE_SSL_MODE", "verify-full")
		t.Setenv("GF_DATABASE_CA_CERT_PATH", "/certs/ca.crt")
		t.Setenv("GF_DATABASE_CLIENT_CERT_PATH", "/certs/client.crt")
		t.Setenv("GF_DATABASE_CLIENT_KEY_PATH", "/certs/client.key")

		dsn, err := grafanaDatasourceDSN(fallback)
		require.NoError(t, err)
		assert.Equal(t,
			"postgres://grafana:grafana@pg.example:5432/grafana"+
				"?sslcert=%2Fcerts%2Fclient.crt&sslkey=%2Fcerts%2Fclient.key&sslmode=verify-full&sslrootcert=%2Fcerts%2Fca.crt",
			dsn)
	})

	t.Run("accepts the legacy single URL", func(t *testing.T) {
		t.Setenv("GF_DATABASE_URL", "postgres://user:pass@pg.example:5432/gf?sslmode=verify-full")

		dsn, err := grafanaDatasourceDSN(fallback)
		require.NoError(t, err)
		assert.Equal(t, "postgres://user:pass@pg.example:5432/gf?sslmode=verify-full", dsn)
	})

	t.Run("uses the bundled database when nothing is configured", func(t *testing.T) {
		// The ansible role creates this database and user under exactly this condition, so the
		// branches cannot disagree.
		dsn, err := grafanaDatasourceDSN(fallback)
		require.NoError(t, err)
		assert.Equal(t, "postgres://grafana:grafana@127.0.0.1:5432/grafana?sslmode=disable", dsn)
	})

	t.Run("reports having nowhere to look", func(t *testing.T) {
		_, err := grafanaDatasourceDSN(grafanaDBFallback{})
		require.Error(t, err)
	})
}

func TestQueryMetricsDatasourceUID(t *testing.T) {
	t.Parallel()

	t.Run("returns the stored UID", func(t *testing.T) {
		t.Parallel()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		// A server first installed before PMM 2.28.0 stores a random UID here, not the derived one.
		mock.ExpectQuery("SELECT uid FROM data_source").
			WithArgs(provisionedOrgID, metricsDatasourceName).
			WillReturnRows(sqlmock.NewRows([]string{"uid"}).AddRow("prometheus"))

		uid, err := queryMetricsDatasourceUID(context.Background(), db)
		require.NoError(t, err)
		assert.Equal(t, "prometheus", uid)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("reports a missing row", func(t *testing.T) {
		t.Parallel()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("SELECT uid FROM data_source").
			WithArgs(provisionedOrgID, metricsDatasourceName).
			WillReturnRows(sqlmock.NewRows([]string{"uid"}))

		_, err = queryMetricsDatasourceUID(context.Background(), db)
		require.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestDatasourceResolver(t *testing.T) {
	t.Parallel()

	newResolver := func(t *testing.T, db *sql.DB) *grafanaReader {
		t.Helper()
		r := newGrafanaReader("dsn-not-used", logrus.WithField("test", t.Name()))
		r.db = db
		return r
	}

	t.Run("prefers the stored UID", func(t *testing.T) {
		t.Parallel()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("SELECT uid FROM data_source").
			WillReturnRows(sqlmock.NewRows([]string{"uid"}).AddRow("prometheus"))

		uid, err := newResolver(t, db).ResolveDatasourceUID(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "prometheus", uid)
	})

	t.Run("derives when Grafana has not provisioned its datasources yet", func(t *testing.T) {
		t.Parallel()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("SELECT uid FROM data_source").WillReturnRows(sqlmock.NewRows([]string{"uid"}))

		uid, err := newResolver(t, db).ResolveDatasourceUID(context.Background())
		require.NoError(t, err)
		assert.Equal(t, deriveMetricsDatasourceUID(), uid)
	})

	t.Run("derives when Grafana has never created its schema", func(t *testing.T) {
		t.Parallel()

		// The state a fresh container is in: the rules are written before supervisord is told to
		// start Grafana, so Grafana has not yet run the migrations that create data_source. Failing
		// closed here would leave every fresh install without built-in rules.
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("SELECT uid FROM data_source").
			WillReturnError(&pq.Error{Code: undefinedTableCode, Message: `relation "data_source" does not exist`})

		uid, err := newResolver(t, db).ResolveDatasourceUID(context.Background())
		require.NoError(t, err)
		assert.Equal(t, deriveMetricsDatasourceUID(), uid)
	})

	t.Run("fails closed when the database cannot be reached", func(t *testing.T) {
		t.Parallel()

		// Deriving here would be a guess, and it is wrong on every server whose datasource predates
		// Grafana 8.3.4. A rule naming a UID that does not exist reports healthy while querying
		// nothing, so no rules at all is the safer answer.
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("SELECT uid FROM data_source").WillReturnError(errors.New("connection refused"))

		uid, err := newResolver(t, db).ResolveDatasourceUID(context.Background())
		require.ErrorIs(t, err, errDatasourceUnresolved)
		assert.Empty(t, uid)
	})

	// Neither a missing database nor a permission failure means Grafana has no datasource; both
	// leave a real UID unread, so neither may derive.
	for _, code := range []pqerror.Code{"3D000", "42501"} {
		t.Run("fails closed on SQLSTATE "+string(code), func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			mock.ExpectQuery("SELECT uid FROM data_source").WillReturnError(&pq.Error{Code: code})

			_, err = newResolver(t, db).ResolveDatasourceUID(context.Background())
			require.ErrorIs(t, err, errDatasourceUnresolved)
		})
	}

	t.Run("does not query again within the recheck window", func(t *testing.T) {
		t.Parallel()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		// One query serves every call inside the window, so the reconcile tick, a settings change
		// and a retry do not each cost a round trip.
		mock.ExpectQuery("SELECT uid FROM data_source").
			WillReturnRows(sqlmock.NewRows([]string{"uid"}).AddRow("prometheus"))

		resolver := newResolver(t, db)
		for range 5 {
			uid, err := resolver.ResolveDatasourceUID(context.Background())
			require.NoError(t, err)
			assert.Equal(t, "prometheus", uid)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("re-reads once the recheck window has passed", func(t *testing.T) {
		t.Parallel()

		// An administrator who deletes and recreates the datasource mints a new UID. Re-reading is
		// what picks that up without restarting pmm-managed.
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("SELECT uid FROM data_source").
			WillReturnRows(sqlmock.NewRows([]string{"uid"}).AddRow("prometheus"))
		mock.ExpectQuery("SELECT uid FROM data_source").
			WillReturnRows(sqlmock.NewRows([]string{"uid"}).AddRow("recreated"))

		resolver := newResolver(t, db)
		uid, err := resolver.ResolveDatasourceUID(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "prometheus", uid)

		resolver.checkedAt = time.Now().Add(-datasourceRecheckInterval - time.Second)

		uid, err = resolver.ResolveDatasourceUID(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "recreated", uid)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("keeps looking while Grafana has not created the datasource", func(t *testing.T) {
		t.Parallel()

		// A derived answer is never cached, so the real row is picked up the moment Grafana
		// finishes its first start rather than a recheck window later.
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("SELECT uid FROM data_source").WillReturnRows(sqlmock.NewRows([]string{"uid"}))
		mock.ExpectQuery("SELECT uid FROM data_source").
			WillReturnRows(sqlmock.NewRows([]string{"uid"}).AddRow("b_nGyianz"))

		resolver := newResolver(t, db)
		uid, err := resolver.ResolveDatasourceUID(context.Background())
		require.NoError(t, err)
		assert.Equal(t, deriveMetricsDatasourceUID(), uid)

		uid, err = resolver.ResolveDatasourceUID(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "b_nGyianz", uid)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestConflictingRules(t *testing.T) {
	t.Parallel()

	newReader := func(t *testing.T, db *sql.DB) *grafanaReader {
		t.Helper()
		r := newGrafanaReader("dsn-not-used", logrus.WithField("test", t.Name()))
		r.db = db
		return r
	}

	t.Run("reports a UID owned by the provisioning API and one owned by a user", func(t *testing.T) {
		t.Parallel()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		// "api" makes Grafana refuse to start; "" is a rule made in the interface, which Grafana
		// would let the file overwrite silently. Both must be left alone.
		mock.ExpectQuery("FROM alert_rule").
			WillReturnRows(sqlmock.NewRows([]string{"uid", "provenance"}).
				AddRow("pmm-clickhouse-down", "api").
				AddRow("pmm-grafana-down", ""))

		conflicts, err := newReader(t, db).ConflictingRules(context.Background(), catalogUIDs())
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"pmm-clickhouse-down": "api", "pmm-grafana-down": ""}, conflicts)
	})

	t.Run("reports nothing when every UID is ours", func(t *testing.T) {
		t.Parallel()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("FROM alert_rule").WillReturnRows(sqlmock.NewRows([]string{"uid", "provenance"}))

		conflicts, err := newReader(t, db).ConflictingRules(context.Background(), catalogUIDs())
		require.NoError(t, err)
		assert.Empty(t, conflicts)
	})

	t.Run("treats a missing schema as nothing to conflict with", func(t *testing.T) {
		t.Parallel()

		// A fresh container writes the rules before Grafana has ever started, so its alerting
		// tables do not exist yet - and nothing can own a UID in a database with no rules.
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("FROM alert_rule").
			WillReturnError(&pq.Error{Code: undefinedTableCode, Message: `relation "alert_rule" does not exist`})

		conflicts, err := newReader(t, db).ConflictingRules(context.Background(), catalogUIDs())
		require.NoError(t, err)
		assert.Empty(t, conflicts)
	})

	t.Run("fails rather than guessing when the query cannot run", func(t *testing.T) {
		t.Parallel()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("FROM alert_rule").WillReturnError(errors.New("connection refused"))

		_, err = newReader(t, db).ConflictingRules(context.Background(), catalogUIDs())
		require.Error(t, err)
	})

	t.Run("asks nothing when there are no UIDs", func(t *testing.T) {
		t.Parallel()

		conflicts, err := newGrafanaReader("", logrus.WithField("test", t.Name())).
			ConflictingRules(context.Background(), nil)
		require.NoError(t, err)
		assert.Empty(t, conflicts)
	})
}

// TestCatalogUIDs pins that the guard checks every UID PMM claims, including a disabled bundle's -
// those are exactly the ones a user can squat.
func TestCatalogUIDs(t *testing.T) {
	t.Parallel()

	uids := catalogUIDs()
	assert.Len(t, uids, 9+len(retiredRuleUIDs))
	assert.Contains(t, uids, "pmm-ha-no-leader")
	assert.Contains(t, uids, "pmm-qan-api2-down")
}
