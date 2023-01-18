// Copyright (C) 2017 Percona LLC
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

package models

// GrafanaParams - defines flags and settings for grafana.
type GrafanaParams struct {
	// PostgresAddr represent postgresql connection address.
	PostgresAddr string
	// PostgresDBName represent postgresql database name.
	PostgresDBName string
	// PostgresDBUsername represent postgresql database username.
	PostgresDBUsername string
	// PostgresDBPassword represent postgresql database user password.
	PostgresDBPassword string
	// PostgresSSLMode represent postgresql database ssl mode.
	PostgresSSLMode string
	// PostgresSSLCAPath represent postgresql database root ssl CA cert path.
	PostgresSSLCAPath string
	// PostgresSSLKeyPath represent postgresql database user ssl key path.
	PostgresSSLKeyPath string
	// PostgresSSLCertPath represent postgresql database user ssl cert path.
	PostgresSSLCertPath string
}
