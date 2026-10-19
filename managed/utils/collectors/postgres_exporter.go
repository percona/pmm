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

package collectors

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/percona/pmm/version"
)

// PostgresExporterCollectorsVersion is the first pmm-agent version whose postgres_exporter takes the
// --collector.<name> and --no-collector.<name> flags of its built-in collectors.
var PostgresExporterCollectorsVersion = version.MustParse("2.41.0-0")

// postgresExporterCollector is a built-in collector of postgres_exporter.
type postgresExporterCollector struct {
	// enabledByDefault is true when the exporter runs the collector unless it gets --no-collector.<name>.
	enabledByDefault bool
	// since is the first pmm-agent version whose postgres_exporter has the collector, when that is later than
	// PostgresExporterCollectorsVersion. An older exporter doesn't know the collector's flags and exits on them.
	since *version.Parsed
}

// postgresExporterCollectors are the built-in collectors of the postgres_exporter that pmm-agent runs.
var postgresExporterCollectors = map[string]postgresExporterCollector{
	"database":                  {enabledByDefault: true},
	"database_wraparound":       {enabledByDefault: true},
	"extensions":                {enabledByDefault: true},
	"locks":                     {enabledByDefault: true},
	"long_running_transactions": {},
	"postmaster":                {},
	"process_idle":              {},
	"replication":               {enabledByDefault: true},
	"replication_slot":          {enabledByDefault: true},
	"stat_activity_autovacuum":  {},
	"stat_bgwriter":             {enabledByDefault: true},
	"stat_database":             {enabledByDefault: true},
	"stat_io":                   {since: version.MustParse("3.6.0")},
	"stat_statements":           {},
	"stat_user_tables":          {enabledByDefault: true},
	"stat_wal_receiver":         {},
	"statio_user_indexes":       {},
	"statio_user_tables":        {enabledByDefault: true},
	"wal":                       {},
}

// supported reports whether the postgres_exporter of the given pmm-agent version has the collector.
func (c postgresExporterCollector) supported(pmmAgentVersion *version.Parsed) bool {
	if pmmAgentVersion.Less(PostgresExporterCollectorsVersion) {
		return false
	}

	return c.since == nil || !pmmAgentVersion.Less(c.since)
}

// minVersion returns the first pmm-agent version whose postgres_exporter has the collector.
func (c postgresExporterCollector) minVersion() *version.Parsed {
	if c.since != nil {
		return c.since
	}

	return PostgresExporterCollectorsVersion
}

// ValidatePostgresExporterCollectors checks the collectors to enable in postgres_exporter: each one must be a
// built-in collector of the exporter, supported by the exporter of the given pmm-agent version, and not also
// disabled. A nil pmmAgentVersion, of a pmm-agent that hasn't reported one yet, skips the version check; the
// exporter configuration leaves out the flags that its exporter doesn't take.
func ValidatePostgresExporterCollectors(enabled, disabled []string, pmmAgentVersion *version.Parsed) error {
	for _, name := range enabled {
		c, ok := postgresExporterCollectors[name]
		if !ok {
			return fmt.Errorf("unknown postgres_exporter collector %q; the collectors are: %s",
				name, strings.Join(slices.Sorted(maps.Keys(postgresExporterCollectors)), ", "))
		}

		if slices.Contains(disabled, name) {
			return fmt.Errorf("collector %q is both enabled and disabled", name)
		}

		if pmmAgentVersion != nil && !c.supported(pmmAgentVersion) {
			return fmt.Errorf("collector %q needs pmm-agent %s or newer, but the pmm-agent is %s",
				name, c.minVersion(), pmmAgentVersion)
		}
	}

	return nil
}

// PostgresExporterCollectorArgs returns the postgres_exporter arguments that turn collectors on and off:
// --collector.<name> for an enabled collector that the exporter doesn't run by default, and
// --no-collector.<name> for a disabled collector that it does. Disabling wins. Names that aren't built-in
// collectors, or whose collector the exporter of this pmm-agent version doesn't have, give no argument:
// the exporter exits on a flag it doesn't know.
func PostgresExporterCollectorArgs(enabled, disabled []string, pmmAgentVersion *version.Parsed) []string {
	var args []string
	for _, name := range slices.Sorted(maps.Keys(postgresExporterCollectors)) {
		c := postgresExporterCollectors[name]
		if !c.supported(pmmAgentVersion) {
			continue
		}

		switch {
		case slices.Contains(disabled, name):
			if c.enabledByDefault {
				args = append(args, "--no-collector."+name)
			}
		case !c.enabledByDefault && slices.Contains(enabled, name):
			args = append(args, "--collector."+name)
		}
	}

	return args
}
