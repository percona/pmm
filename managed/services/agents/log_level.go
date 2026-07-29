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

const (
	// Flag used by exporters which parse their command line with kingpin.
	logLevelFlag = "--log.level"

	// Flag used by valkey_exporter, which parses its command line with the standard library
	// flag package. That package rejects --log.level and exits with code 2, which used to
	// kill the exporter right after start and leave the agent DONE (PMM-15201).
	valkeyLogLevelFlag = "--log-level"
)

// withLogLevel appends the --log.level CLI arg. The mysqld_exporter, node_exporter and
// postgres_exporter binaries don't support --log.level=fatal.
func withLogLevel(args []string, logLevel *string, pmmAgentVersion *version.Parsed, supportLogLevelFatal bool) []string {
	return withLogLevelFlag(args, logLevelFlag, logLevel, pmmAgentVersion, supportLogLevelFatal)
}

// withLogLevelFlag appends "<flag>=<level>" for exporters which spell the log level flag
// differently than the kingpin-based majority.
func withLogLevelFlag(args []string, flag string, logLevel *string, pmmAgentVersion *version.Parsed, supportLogLevelFatal bool) []string {
	level := pointer.GetString(logLevel)
	if level == "" || pmmAgentVersion.Less(exporterLogLevelCommandVersion) {
		return args
	}

	// Some exporters dropped support for the fatal level, so fall back to error to keep a
	// previously stored "fatal" working.
	if !supportLogLevelFatal && level == "fatal" {
		level = "error"
	}

	return append(args, flag+"="+level)
}
