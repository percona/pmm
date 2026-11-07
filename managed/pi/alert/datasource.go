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

package alert

import "fmt"

// Supported query datasources, named after the built-in Grafana datasources.
const (
	Metrics    = Datasource("metrics")
	ClickHouse = Datasource("clickhouse")
)

// Datasource is the built-in Grafana datasource an alert template query runs against.
type Datasource string

// Validate returns error in case of invalid datasource.
func (d Datasource) Validate() error {
	switch d {
	case "":
		// Empty means Metrics.
		return nil
	case Metrics:
		return nil
	case ClickHouse:
		return nil
	}

	// do not add `default:` to make exhaustive linter do its job

	return fmt.Errorf("unhandled query datasource '%s', want '%s' or '%s'", d, Metrics, ClickHouse)
}
