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

package agents

import (
	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/version"
)

// Log level available in exporters with pmm 2.28.
var exporterLogLevelCommandVersion = version.MustParse("2.27.99")

// withLogLevel appends the --log.level CLI arg used by the kingpin-based exporters, of which
// mysqld_exporter, node_exporter and postgres_exporter don't support --log.level=fatal.
func withLogLevel(args []string, logLevel *string, pmmAgentVersion *version.Parsed, supportLogLevelFatal bool) []string {
	return withLogLevelFlag(args, "--log.level", logLevel, pmmAgentVersion, supportLogLevelFatal)
}

// withLogLevelFlag appends "<flagName>=<level>" if pmm-agent is new enough, downgrading fatal
// to error for exporters which don't support it.
func withLogLevelFlag(args []string, flagName string, logLevel *string, pmmAgentVersion *version.Parsed, supportLogLevelFatal bool) []string {
	level := pointer.GetString(logLevel)
	if level == "" || pmmAgentVersion.Less(exporterLogLevelCommandVersion) {
		return args
	}

	// Keep a previously stored 'fatal' working on exporters which dropped that level.
	if !supportLogLevelFatal && level == "fatal" {
		level = "error"
	}

	return append(args, flagName+"="+level)
}
