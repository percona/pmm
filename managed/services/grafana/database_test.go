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

package grafana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Deliberately not parallel: the subtests use t.Setenv, which panics in a parallel test.
func TestDatabaseDSN(t *testing.T) {
	const (
		fallbackAddr = "127.0.0.1:5432"
		sslParams    = "sslmode=disable"
	)

	t.Run("prefers the discrete GF_DATABASE variables", func(t *testing.T) {
		// This is what the pmm-ha Helm chart sets, and its user is not "grafana".
		t.Setenv("GF_DATABASE_HOST", "pg.example:5432")
		t.Setenv("GF_DATABASE_NAME", "grafana")
		t.Setenv("GF_DATABASE_USER", "gfuser")
		t.Setenv("GF_DATABASE_PASSWORD", "secret")
		t.Setenv("GF_DATABASE_SSL_MODE", "require")

		dsn, err := DatabaseDSN(fallbackAddr, sslParams)
		require.NoError(t, err)
		assert.Equal(t, "postgres://gfuser:secret@pg.example:5432/grafana?sslmode=require", dsn)
	})

	t.Run("falls back to what grafana.ini ships for the variables left unset", func(t *testing.T) {
		// Grafana overrides one setting per variable, so a deployment that sets only the host still
		// authenticates with the credentials in grafana.ini. Defaulting the password to empty here
		// would fail against a database Grafana itself reaches happily.
		t.Setenv("GF_DATABASE_HOST", "pg.example:5432")

		dsn, err := DatabaseDSN(fallbackAddr, sslParams)
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

		dsn, err := DatabaseDSN(fallbackAddr, sslParams)
		require.NoError(t, err)
		assert.Equal(t,
			"postgres://grafana:grafana@pg.example:5432/grafana"+
				"?sslcert=%2Fcerts%2Fclient.crt&sslkey=%2Fcerts%2Fclient.key&sslmode=verify-full&sslrootcert=%2Fcerts%2Fca.crt",
			dsn)
	})

	t.Run("accepts the legacy single URL", func(t *testing.T) {
		t.Setenv("GF_DATABASE_URL", "postgres://user:pass@pg.example:5432/gf?sslmode=verify-full")

		dsn, err := DatabaseDSN(fallbackAddr, sslParams)
		require.NoError(t, err)
		assert.Equal(t, "postgres://user:pass@pg.example:5432/gf?sslmode=verify-full", dsn)
	})

	t.Run("uses the bundled database when nothing is configured", func(t *testing.T) {
		// The ansible role creates this database and user under exactly this condition, so the
		// branches cannot disagree.
		dsn, err := DatabaseDSN(fallbackAddr, sslParams)
		require.NoError(t, err)
		assert.Equal(t, "postgres://grafana:grafana@127.0.0.1:5432/grafana?sslmode=disable", dsn)
	})

	t.Run("reports having nowhere to look", func(t *testing.T) {
		_, err := DatabaseDSN("", "")
		require.Error(t, err)
	})
}
