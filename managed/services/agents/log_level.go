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

// Package agents provides jobs functionality.
package agents

import (
	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/version"
)

// Log level available in exporters with pmm 2.28.
var exporterLogLevelCommandVersion = version.MustParse("2.27.99")

// withLogLevel - append CLI args --log.level
// mysqld_exporter, node_exporter and postgres_exporter don't support --log.level=fatal.
func withLogLevel(args []string, logLevel *string, pmmAgentVersion *version.Parsed, supportLogLevelFatal bool) []string {
	level := pointer.GetString(logLevel)

	if level != "" && !pmmAgentVersion.Less(exporterLogLevelCommandVersion) {
		// exists exporters that not support --log.level=fatal anymore after last update
		// so replace "fatal" to "error" for previous stored state
		if !supportLogLevelFatal && level == "fatal" {
			level = "error"
		}

		args = append(args, "--log.level="+level)
	}

	return args
}
