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
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/percona/pmm/managed/utils/envvars"
)

const (
	bundledGrafanaDB       = "grafana"
	bundledGrafanaUser     = "grafana"
	bundledGrafanaPassword = "grafana"
)

// DatabaseDSN works out how to reach Grafana's database.
//
// The fallback address and SSL parameters describe where Grafana's database lives when the deployment
// has not said. This is the bundled case: Grafana shares PMM's own PostgreSQL server, and the database
// and user are created by the initialization ansible role. The address is PMM's PostgreSQL address,
// host:port; the parameters are the encoded query string PMM uses for its own connection.
//
// The order matters. Since PMM 3.2.0 an external Grafana database is configured with the discrete
// GF_DATABASE_* variables, and that is what the pmm-ha Helm chart sets - with a user that is not
// necessarily "grafana". GF_DATABASE_URL is the older single-URL form. Only when neither is present
// is the database the bundled one, which is exactly the condition under which the ansible role
// creates it, so the branches cannot overlap.
func DatabaseDSN(fallbackAddr, sslParams string) (string, error) {
	if host := os.Getenv("GF_DATABASE_HOST"); host != "" {
		// Grafana overrides one setting per variable and takes the rest from grafana.ini, so every
		// field left unset here has to fall back to what that file ships, not to an empty value.
		dsn := url.URL{
			Scheme: "postgres",
			User: url.UserPassword(
				envvars.GetEnv("GF_DATABASE_USER", bundledGrafanaUser),
				envvars.GetEnv("GF_DATABASE_PASSWORD", bundledGrafanaPassword),
			),
			Host: host,
			Path: envvars.GetEnv("GF_DATABASE_NAME", bundledGrafanaDB),
		}

		query := make(url.Values)
		query.Set("sslmode", envvars.GetEnv("GF_DATABASE_SSL_MODE", "disable"))
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

	if fallbackAddr == "" {
		return "", errors.New("no Grafana database address available")
	}

	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(bundledGrafanaUser, bundledGrafanaPassword),
		Host:     fallbackAddr,
		Path:     bundledGrafanaDB,
		RawQuery: sslParams,
	}

	return dsn.String(), nil
}
