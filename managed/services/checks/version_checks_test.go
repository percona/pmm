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

package checks

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/pi/check"
	"github.com/percona/pmm/managed/pi/starlark"
)

type finding struct {
	Severity string
	Summary  string
	Labels   map[string]string
}

func findings(results []check.Result) []finding {
	out := make([]finding, 0, len(results))
	for _, r := range results {
		f := finding{Severity: r.Severity.String(), Summary: r.Summary}
		if len(r.Labels) > 0 {
			f.Labels = r.Labels
		}

		out = append(out, f)
	}

	return out
}

func loadBuiltinCheck(t *testing.T, name string) check.Check {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("..", "..", "data", "checks", name+".yml"))
	require.NoError(t, err)

	parsed, err := check.ParseChecks(bytes.NewReader(b), &check.ParseParams{
		DisallowUnknownFields: true,
		DisallowInvalidChecks: true,
	})
	require.NoError(t, err)
	require.Len(t, parsed, 1)

	return parsed[0]
}

var tableEntryRE = regexp.MustCompile(`^\s*"([^"]+)":\s*"?([^",\s]+)"?,`)

// scriptTable returns the "key": value entries of the dict literal that opens on the script line containing start.
// It lets tests follow tables that other teams update, instead of pinning their values.
func scriptTable(t *testing.T, name, start string) map[string]string {
	t.Helper()

	script := loadBuiltinCheck(t, name).Script
	i := strings.Index(script, start)
	require.NotEqual(t, -1, i, "%s: %q not found", name, start)

	table := make(map[string]string)
	for _, line := range strings.Split(script[i+len(start):], "\n")[1:] {
		if strings.HasPrefix(strings.TrimSpace(line), "}") {
			break
		}

		m := tableEntryRE.FindStringSubmatch(line)
		require.NotNil(t, m, "%s: unexpected line %q in %q", name, line, start)
		table[m[1]] = m[2]
	}
	require.NotEmpty(t, table, "%s: %q is empty", name, start)

	return table
}

// latestSeries returns the newest "major" or "major.minor" key of a version table.
func latestSeries(t *testing.T, table map[string]string) string {
	t.Helper()

	var latest string
	var latestKey [2]int
	for k := range table {
		var key [2]int
		for i, part := range strings.SplitN(k, ".", 2) {
			n, err := strconv.Atoi(part)
			require.NoError(t, err, k)
			key[i] = n
		}

		if latest == "" || key[0] > latestKey[0] || (key[0] == latestKey[0] && key[1] > latestKey[1]) {
			latest, latestKey = k, key
		}
	}

	return latest
}

func tableInt(t *testing.T, table map[string]string, key string) int {
	t.Helper()

	n, err := strconv.Atoi(table[key])
	require.NoError(t, err, "%s: %q", key, table[key])

	return n
}

// mongoVersion formats a version number of mongodb_version's table, such as 80032, as 8.0.32.
func mongoVersion(num int) string {
	return fmt.Sprintf("%d.%d.%d", num/10000, num/100%100, num%100)
}

// pgVersion formats a server_version_num of PostgreSQL 10 or later, such as 170011, as 17.11.
func pgVersion(num int) string {
	return fmt.Sprintf("%d.%d", num/10000, num%100)
}

// runBuiltinCheck runs a built-in check script the way pmm-managed-starlark does, with the clock fixed to noon UTC of today.
func runBuiltinCheck(t *testing.T, name, today string, docs ...[]map[string]any) []check.Result {
	t.Helper()

	c := loadBuiltinCheck(t, name)

	day, err := time.Parse("2006-01-02", today)
	require.NoError(t, err)
	fixed := day.Add(12 * time.Hour)

	orig := now
	now = func() time.Time { return fixed }
	t.Cleanup(func() { now = orig })

	funcs, err := GetFuncsForVersion(c.Version)
	require.NoError(t, err)

	env, err := starlark.NewEnv(name, c.Script, funcs)
	require.NoError(t, err)

	input := make([]any, len(docs))
	for i, d := range docs {
		input[i] = d
	}

	results, err := env.Run(name, input, GetAdditionalContext(), t.Log)
	require.NoError(t, err)

	return results
}

func mysqlVariables(version, comment, basedir, datadir string, extra ...string) []map[string]any {
	vars := [][2]string{
		{"version", version},
		{"version_comment", comment},
		{"basedir", basedir},
		{"datadir", datadir},
	}
	for i := 0; i+1 < len(extra); i += 2 {
		vars = append(vars, [2]string{extra[i], extra[i+1]})
	}

	rows := make([]map[string]any, 0, len(vars))
	for _, v := range vars {
		rows = append(rows, map[string]any{"Variable_name": v[0], "Value": v[1]})
	}

	return rows
}

func mongoBuildInfo(version string) []map[string]any {
	return []map[string]any{{"version": version}}
}

// eolFinding builds the EOL finding of the version checks: product is the name used in the summary.
func eolFinding(product, severity, current, series, eolDate string) finding {
	summary := product + " " + series + " reached end of life"
	if severity == "warning" {
		summary = product + " " + series + " reaches end of life on " + eolDate
	}

	return finding{
		Severity: severity,
		Summary:  summary,
		Labels:   map[string]string{"current": current, "series": series, "eol_date": eolDate},
	}
}

func mongoEOL(severity, current, series, eolDate string) finding {
	return eolFinding("MongoDB", severity, current, series, eolDate)
}

func pgEOL(severity, current, series, eolDate string) finding {
	return eolFinding("PostgreSQL", severity, current, series, eolDate)
}

func mysqlEOL(product, severity, current, series, eolDate string) finding {
	return eolFinding(product, severity, current, series, eolDate)
}

// rdsEOL builds the RDS finding, which reports the end of standard support instead of end of life.
func rdsEOL(severity, current, series, eolDate string) finding {
	summary := "Amazon RDS for MySQL " + series + " reached end of standard support"
	if severity == "warning" {
		summary = "Amazon RDS for MySQL " + series + " reaches end of standard support on " + eolDate
	}

	return finding{
		Severity: severity,
		Summary:  summary,
		Labels:   map[string]string{"current": current, "series": series, "eol_date": eolDate},
	}
}

// newerMySQL builds the mysql_version finding: product is the name used in the summary.
func newerMySQL(product, current, latest string) finding {
	return finding{
		Severity: "warning",
		Summary:  "Newer version of " + product + " is available",
		Labels:   map[string]string{"current": current, "latest": latest},
	}
}

// newerPG builds the postgresql_version_check newer-version finding.
func newerPG(current, latest string) finding {
	return finding{
		Severity: "warning",
		Summary:  "Newer version of PostgreSQL is available",
		Labels:   map[string]string{"current": current, "latest": latest},
	}
}

func mongoCmdLineOpts(parsed map[string]any) []map[string]any {
	return []map[string]any{{"parsed": parsed}}
}

func pgVersionRows(serverVersionNum int) []map[string]any {
	return []map[string]any{{
		"setting": strconv.Itoa(serverVersionNum),
		"major":   int64(serverVersionNum / 10000),
	}}
}

func promVector(labels map[string]string, value string) []map[string]any {
	metric := make(map[string]any, len(labels))
	for k, v := range labels {
		metric[k] = v
	}

	return []map[string]any{{
		"metric": metric,
		"value":  []any{float64(1762948800), value},
	}}
}

func TestBuiltinVersionChecks(t *testing.T) {
	const today = "2026-11-12"

	// The PSMDB and PostgreSQL teams update these tables, so the cases below read their values instead of pinning them.
	mongoLatest := scriptTable(t, "mongodb_version", `"version_check": {`)
	mongoLatestSeries := latestSeries(t, mongoLatest)
	pgLatest := scriptTable(t, "postgresql_version_check", "latest_versions = {")
	pgExpires := scriptTable(t, "postgresql_version_check", "version_expires = {")
	pgLatestSeries := latestSeries(t, pgLatest)

	for _, tc := range []struct {
		name  string
		check string
		today string
		docs  [][]map[string]any
		want  []finding
		// descContains lists substrings that every finding's description must contain.
		descContains []string
		// descExcludes lists substrings that no finding's description may contain.
		descExcludes []string
	}{
		{
			name:         "mysql_version Oracle 8.4.0",
			check:        "mysql_version",
			today:        today,
			docs:         [][]map[string]any{mysqlVariables("8.4.0", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want:         []finding{newerMySQL("MySQL", "8.4.0", "8.4.13")},
			descContains: []string{"security fixes", "https://www.oracle.com/security-alerts/"},
		},
		{
			name:         "mysql_version Oracle 9.7.0",
			check:        "mysql_version",
			today:        today,
			docs:         [][]map[string]any{mysqlVariables("9.7.0", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want:         []finding{newerMySQL("MySQL", "9.7.0", "9.7.4")},
			descContains: []string{"security fixes"},
		},
		{
			name:  "mysql_version Oracle 9.8.0 not in tables",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("9.8.0", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want:  []finding{},
		},
		{
			name:  "mysql_version Oracle 26.10.0 not in tables",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("26.10.0", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want:  []finding{},
		},
		{
			name:         "mysql_version Percona Server 8.4.3-3",
			check:        "mysql_version",
			today:        today,
			docs:         [][]map[string]any{mysqlVariables("8.4.3-3", "Percona Server (GPL), Release 3, Revision 8f39d3a6", "/usr/", "/var/lib/mysql/")},
			want:         []finding{newerMySQL("Percona Server for MySQL", "8.4.3", "8.4.11")},
			descContains: []string{"security fixes", "https://www.oracle.com/security-alerts/"},
		},
		{
			name:         "mysql_version PXC 8.4.3-3.1",
			check:        "mysql_version",
			today:        today,
			docs:         [][]map[string]any{mysqlVariables("8.4.3-3.1", "Percona XtraDB Cluster (GPL), Release rel3, Revision 2a2b6d2, WSREP version 26.1.4.3", "/usr/", "/var/lib/mysql/")},
			want:         []finding{newerMySQL("Percona XtraDB Cluster", "8.4.3", "8.4.11")},
			descContains: []string{"security fixes"},
		},
		{
			name:         "mysql_version PXC 9.7.1-1",
			check:        "mysql_version",
			today:        today,
			docs:         [][]map[string]any{mysqlVariables("9.7.1-1", "Percona XtraDB Cluster (GPL), Release rel3, Revision 2a2b6d2, WSREP version 26.1.4.3", "/usr/", "/var/lib/mysql/")},
			want:         []finding{newerMySQL("Percona XtraDB Cluster", "9.7.1", "9.7.2")},
			descContains: []string{"security fixes"},
		},
		{
			name:         "mysql_version MariaDB 10.11.2",
			check:        "mysql_version",
			today:        today,
			docs:         [][]map[string]any{mysqlVariables("10.11.2-MariaDB", "MariaDB Server", "/usr/", "/var/lib/mysql/")},
			want:         []finding{newerMySQL("MariaDB", "10.11.2", "10.11.19")},
			descContains: []string{"security fixes", "https://mariadb.org/about/security-policy/"},
			descExcludes: []string{"10.9.2", "10.4.26", "https://www.oracle.com/"},
		},
		{
			name:         "mysql_version MariaDB 11.4.1",
			check:        "mysql_version",
			today:        today,
			docs:         [][]map[string]any{mysqlVariables("11.4.1-MariaDB-log", "mariadb.org binary distribution", "/usr/", "/var/lib/mysql/")},
			want:         []finding{newerMySQL("MariaDB", "11.4.1", "11.4.13")},
			descContains: []string{"security fixes"},
		},
		{
			name:  "mysql_version MariaDB 12.3.3 at the latest",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("12.3.3-MariaDB", "MariaDB Server", "/usr/", "/var/lib/mysql/")},
			want:  []finding{},
		},
		{
			name:  "mysql_version MariaDB 12.4.1 not in tables",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("12.4.1-MariaDB", "MariaDB Server", "/usr/", "/var/lib/mysql/")},
			want:  []finding{},
		},
		{
			name:         "mysql_version RDS 8.4.3",
			check:        "mysql_version",
			today:        today,
			docs:         [][]map[string]any{mysqlVariables("8.4.3", "Source distribution", "/rdsdbbin/mysql-8.4.3.R1/", "/rdsdbdata/db/")},
			want:         []finding{newerMySQL("MySQL", "8.4.3", "8.4.11")},
			descContains: []string{"security fixes"},
		},
		{
			name:         "mysql_version RDS MariaDB 10.11.8 follows MariaDB",
			check:        "mysql_version",
			today:        today,
			docs:         [][]map[string]any{mysqlVariables("10.11.8-MariaDB-log", "Source distribution", "/rdsdbbin/mariadb-10.11.8.R1/", "/rdsdbdata/db/")},
			want:         []finding{newerMySQL("MariaDB", "10.11.8", "10.11.19")},
			descContains: []string{"https://mariadb.org/about/security-policy/"},
		},
		{
			name:  "mysql_version Aurora",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("8.0.39", "Source distribution", "/rdsdbbin/oscar-8.0.mysql_aurora.3.05.2.0.36764.0/", "/rdsdbdata/db/", "aurora_version", "3.05.2")},
			want:  []finding{},
		},
		{
			name:  "mysql_version no rows",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{{}},
			want:  []finding{},
		},
		{
			name:         "mysql_version Oracle 8.0.37 past EOL gives only the EOL error",
			check:        "mysql_version",
			today:        today,
			docs:         [][]map[string]any{mysqlVariables("8.0.37", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want:         []finding{mysqlEOL("MySQL", "error", "8.0.37", "8.0", "2026-04-30")},
			descContains: []string{"2026-04-30", "supported release series"},
			descExcludes: []string{"8.0.46", "includes security fixes", "Newer version"},
		},
		{
			name:  "mysql_version Oracle 8.0.46 past EOL",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("8.0.46", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want:  []finding{mysqlEOL("MySQL", "error", "8.0.46", "8.0", "2026-04-30")},
		},
		{
			name:         "mysql_version Oracle 5.7.40-log past EOL",
			check:        "mysql_version",
			today:        today,
			docs:         [][]map[string]any{mysqlVariables("5.7.40-log", "MySQL Community Server (GPL)", "/usr/", "/var/lib/mysql/")},
			want:         []finding{mysqlEOL("MySQL", "error", "5.7.40-log", "5.7", "2023-10-31")},
			descExcludes: []string{"5.7.44"},
		},
		{
			name:  "mysql_version Oracle 5.5.5-log is not a MariaDB prefix",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("5.5.5-log", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want:  []finding{mysqlEOL("MySQL", "error", "5.5.5-log", "5.5", "2018-12-31")},
		},
		{
			name:  "mysql_version Oracle 8.2.0 Innovation past EOL",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("8.2.0", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want:  []finding{mysqlEOL("MySQL", "error", "8.2.0", "8.2", "2024-01-16")},
		},
		{
			name:  "mysql_version Oracle 9.6.0 Innovation past EOL",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("9.6.0", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want:  []finding{mysqlEOL("MySQL", "error", "9.6.0", "9.6", "2026-04-21")},
		},
		{
			name:  "mysql_version Oracle 26.7.1 Innovation past EOL",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("26.7.1", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want:  []finding{mysqlEOL("MySQL", "error", "26.7.1", "26.7", "2026-10-20")},
		},
		{
			name:  "mysql_version Percona Server 8.0.46-37 past EOL",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("8.0.46-37", "Percona Server (GPL), Release 37, Revision 2ad1a8b6", "/usr/", "/var/lib/mysql/")},
			want:  []finding{mysqlEOL("Percona Server for MySQL", "error", "8.0.46-37", "8.0", "2026-04-01")},
		},
		{
			name:  "mysql_version Percona Server 9.3.0-1 follows Oracle",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("9.3.0-1", "Percona Server (GPL), Release 37, Revision 2ad1a8b6", "/usr/", "/var/lib/mysql/")},
			want:  []finding{mysqlEOL("Percona Server for MySQL", "error", "9.3.0-1", "9.3", "2025-07-22")},
		},
		{
			name:  "mysql_version PXC 5.7.44-31.65 past EOL",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("5.7.44-31.65", "Percona XtraDB Cluster (GPL), Release rel3, Revision 2a2b6d2, WSREP version 26.1.4.3", "/usr/", "/var/lib/mysql/")},
			want:  []finding{mysqlEOL("Percona XtraDB Cluster", "error", "5.7.44-31.65", "5.7", "2023-10-01")},
		},
		{
			name:         "mysql_version MariaDB 10.6.1 past EOL",
			check:        "mysql_version",
			today:        today,
			docs:         [][]map[string]any{mysqlVariables("10.6.1-MariaDB", "MariaDB Server", "/usr/", "/var/lib/mysql/")},
			want:         []finding{mysqlEOL("MariaDB", "error", "10.6.1-MariaDB", "10.6", "2026-07-06")},
			descExcludes: []string{"10.6.28", "includes security fixes"},
		},
		{
			name:  "mysql_version MariaDB 5.5.5-10.6.18 replication prefix",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("5.5.5-10.6.18-MariaDB", "Source distribution", "/usr/", "/var/lib/mysql/")},
			want:  []finding{mysqlEOL("MariaDB", "error", "10.6.18-MariaDB", "10.6", "2026-07-06")},
		},
		{
			name:  "mysql_version MariaDB 5.5.68 past EOL",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("5.5.68-MariaDB", "MariaDB Server", "/usr/", "/var/lib/mysql/")},
			want:  []finding{mysqlEOL("MariaDB", "error", "5.5.68-MariaDB", "5.5", "2020-04-11")},
		},
		{
			name:  "mysql_version MariaDB 11.7.2 short-term past EOL",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("11.7.2-MariaDB", "mariadb.org binary distribution", "/usr/", "/var/lib/mysql/")},
			want:  []finding{mysqlEOL("MariaDB", "error", "11.7.2-MariaDB", "11.7", "2025-05-12")},
		},
		{
			name:         "mysql_version RDS 8.0.28 past end of standard support",
			check:        "mysql_version",
			today:        today,
			docs:         [][]map[string]any{mysqlVariables("8.0.28", "Source distribution", "/rdsdbbin/mysql-8.0.28.R4/", "/rdsdbdata/db/")},
			want:         []finding{rdsEOL("error", "8.0.28", "8.0", "2026-07-31")},
			descContains: []string{"RDS Extended Support"},
			descExcludes: []string{"8.0.46", "includes security fixes"},
		},
		{
			name:         "mysql_version RDS 5.7.44 by datadir",
			check:        "mysql_version",
			today:        today,
			docs:         [][]map[string]any{mysqlVariables("5.7.44", "Source distribution", "/usr/", "/rdsdbdata/db/")},
			want:         []finding{rdsEOL("error", "5.7.44", "5.7", "2024-02-29")},
			descContains: []string{"RDS Extended Support"},
		},
		{
			name:  "mysql_version RDS MariaDB 10.6.18 follows MariaDB",
			check: "mysql_version",
			today: today,
			docs:  [][]map[string]any{mysqlVariables("10.6.18-MariaDB-log", "Source distribution", "/rdsdbbin/mariadb-10.6.18.R1/", "/rdsdbdata/db/")},
			want:  []finding{mysqlEOL("MariaDB", "error", "10.6.18-MariaDB-log", "10.6", "2026-07-06")},
		},
		{
			name:  "mysql_version Oracle 8.4.3 184 days before EOL",
			check: "mysql_version",
			today: "2031-10-29",
			docs:  [][]map[string]any{mysqlVariables("8.4.3", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want:  []finding{newerMySQL("MySQL", "8.4.3", "8.4.13")},
		},
		{
			name:         "mysql_version Oracle 8.4.3 183 days before EOL",
			check:        "mysql_version",
			today:        "2031-10-30",
			docs:         [][]map[string]any{mysqlVariables("8.4.3", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want:         []finding{mysqlEOL("MySQL", "warning", "8.4.3", "8.4", "2032-04-30")},
			descContains: []string{"2032-04-30", "latest release of this series, 8.4.13", "security fixes"},
		},
		{
			name:         "mysql_version Oracle 8.4.13 on EOL day",
			check:        "mysql_version",
			today:        "2032-04-30",
			docs:         [][]map[string]any{mysqlVariables("8.4.13", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want:         []finding{mysqlEOL("MySQL", "warning", "8.4.13", "8.4", "2032-04-30")},
			descExcludes: []string{"latest release of this series", "includes security fixes"},
		},
		{
			name:         "mysql_version Oracle 8.4.3 day after EOL",
			check:        "mysql_version",
			today:        "2032-05-01",
			docs:         [][]map[string]any{mysqlVariables("8.4.3", "MySQL Community Server - GPL", "/usr/", "/var/lib/mysql/")},
			want:         []finding{mysqlEOL("MySQL", "error", "8.4.3", "8.4", "2032-04-30")},
			descExcludes: []string{"8.4.13"},
		},
		{
			name:         "mysql_version MariaDB 10.11.9 within 183 days",
			check:        "mysql_version",
			today:        "2027-12-01",
			docs:         [][]map[string]any{mysqlVariables("10.11.9-MariaDB", "MariaDB Server", "/usr/", "/var/lib/mysql/")},
			want:         []finding{mysqlEOL("MariaDB", "warning", "10.11.9-MariaDB", "10.11", "2028-02-16")},
			descContains: []string{"10.11.19", "https://mariadb.org/about/security-policy/"},
		},
		{
			name:         "mysql_version RDS 8.4.7 within 183 days",
			check:        "mysql_version",
			today:        "2029-02-01",
			docs:         [][]map[string]any{mysqlVariables("8.4.7", "Source distribution", "/rdsdbbin/mysql-8.4.7.R1/", "/rdsdbdata/db/")},
			want:         []finding{rdsEOL("warning", "8.4.7", "8.4", "2029-07-31")},
			descContains: []string{"RDS Extended Support", "8.4.11"},
		},
		{
			name:         "mongodb_version 6.0.19-16 past EOL gives only the EOL error",
			check:        "mongodb_version",
			today:        today,
			docs:         [][]map[string]any{mongoBuildInfo("6.0.19-16")},
			want:         []finding{mongoEOL("error", "6.0.19-16", "6.0", "2025-07-31")},
			descContains: []string{"2025-07-31"},
			descExcludes: []string{mongoVersion(tableInt(t, mongoLatest, "6.0")), "new minor version", "includes security fixes"},
		},
		{
			name:  "mongodb_version 5.0.31 past EOL",
			check: "mongodb_version",
			today: today,
			docs:  [][]map[string]any{mongoBuildInfo("5.0.31")},
			want:  []finding{mongoEOL("error", "5.0.31", "5.0", "2024-10-31")},
		},
		{
			name:  "mongodb_version 4.4.29 past EOL",
			check: "mongodb_version",
			today: today,
			docs:  [][]map[string]any{mongoBuildInfo("4.4.29")},
			want:  []finding{mongoEOL("error", "4.4.29", "4.4", "2024-02-29")},
		},
		{
			name:  "mongodb_version 4.2.25 past EOL, not in the latest table",
			check: "mongodb_version",
			today: today,
			docs:  [][]map[string]any{mongoBuildInfo("4.2.25")},
			want:  []finding{mongoEOL("error", "4.2.25", "4.2", "2023-04-30")},
		},
		{
			name:  "mongodb_version 8.2.1 past EOL, not in the latest table",
			check: "mongodb_version",
			today: today,
			docs:  [][]map[string]any{mongoBuildInfo("8.2.1")},
			want:  []finding{mongoEOL("error", "8.2.1", "8.2", "2026-07-31")},
		},
		{
			name:         "mongodb_version 7.0.14 184 days before EOL",
			check:        "mongodb_version",
			today:        "2027-02-28",
			docs:         [][]map[string]any{mongoBuildInfo("7.0.14")},
			want:         []finding{{Severity: "warning", Summary: "Newer version of MongoDB is available"}},
			descContains: []string{"new minor version " + mongoVersion(tableInt(t, mongoLatest, "7.0"))},
		},
		{
			name:         "mongodb_version 7.0.14 183 days before EOL",
			check:        "mongodb_version",
			today:        "2027-03-01",
			docs:         [][]map[string]any{mongoBuildInfo("7.0.14")},
			want:         []finding{mongoEOL("warning", "7.0.14", "7.0", "2027-08-31")},
			descContains: []string{"2027-08-31", "latest minor version " + mongoVersion(tableInt(t, mongoLatest, "7.0")), "security fixes"},
		},
		{
			name:         "mongodb_version latest 7.0 on EOL day",
			check:        "mongodb_version",
			today:        "2027-08-31",
			docs:         [][]map[string]any{mongoBuildInfo(mongoVersion(tableInt(t, mongoLatest, "7.0")))},
			want:         []finding{mongoEOL("warning", mongoVersion(tableInt(t, mongoLatest, "7.0")), "7.0", "2027-08-31")},
			descExcludes: []string{"latest minor version", "includes security fixes"},
		},
		{
			name:         "mongodb_version 7.0.14 day after EOL",
			check:        "mongodb_version",
			today:        "2027-09-01",
			docs:         [][]map[string]any{mongoBuildInfo("7.0.14")},
			want:         []finding{mongoEOL("error", "7.0.14", "7.0", "2027-08-31")},
			descExcludes: []string{mongoVersion(tableInt(t, mongoLatest, "7.0"))},
		},
		{
			name:         "mongodb_version 8.0.4",
			check:        "mongodb_version",
			today:        today,
			docs:         [][]map[string]any{mongoBuildInfo("8.0.4")},
			want:         []finding{{Severity: "warning", Summary: "Newer version of MongoDB is available"}},
			descContains: []string{"new minor version " + mongoVersion(tableInt(t, mongoLatest, "8.0")), "security fixes", "https://www.mongodb.com/resources/products/alerts"},
		},
		{
			name:         "mongodb_version latest 7.0",
			check:        "mongodb_version",
			today:        today,
			docs:         [][]map[string]any{mongoBuildInfo(mongoVersion(tableInt(t, mongoLatest, "7.0")))},
			want:         []finding{{Severity: "warning", Summary: "Newer version of MongoDB is available"}},
			descContains: []string{"This is the latest minor patch for this version", "Major version " + mongoLatestSeries + " available", "security fixes"},
			descExcludes: []string{"new minor version"},
		},
		{
			name:  "mongodb_version latest of the latest series",
			check: "mongodb_version",
			today: today,
			docs:  [][]map[string]any{mongoBuildInfo(mongoVersion(tableInt(t, mongoLatest, mongoLatestSeries)))},
			want:  []finding{},
		},
		{
			name:  "mongodb_version 8.0.4 day after EOL",
			check: "mongodb_version",
			today: "2029-11-01",
			docs:  [][]map[string]any{mongoBuildInfo("8.0.4")},
			want:  []finding{mongoEOL("error", "8.0.4", "8.0", "2029-10-31")},
		},
		{
			name:         "mongodb_version 8.3.0 well before EOL",
			check:        "mongodb_version",
			today:        today,
			docs:         [][]map[string]any{mongoBuildInfo("8.3.0")},
			want:         []finding{{Severity: "warning", Summary: "Newer version of MongoDB is available"}},
			descContains: []string{"new minor version " + mongoVersion(tableInt(t, mongoLatest, "8.3")) + " available"},
			descExcludes: []string{"Major version"},
		},
		{
			name:  "mongodb_version 8.3.0 183 days before EOL",
			check: "mongodb_version",
			today: "2029-05-01",
			docs:  [][]map[string]any{mongoBuildInfo("8.3.0")},
			want:  []finding{mongoEOL("warning", "8.3.0", "8.3", "2029-10-31")},
		},
		{
			name:  "mongodb_version 3.4.24 past EOL",
			check: "mongodb_version",
			today: today,
			docs:  [][]map[string]any{mongoBuildInfo("3.4.24")},
			want:  []finding{mongoEOL("error", "3.4.24", "3.4", "2020-01-31")},
		},
		{
			name:  "mongodb_version 9.0.1 not in the latest table",
			check: "mongodb_version",
			today: today,
			docs:  [][]map[string]any{mongoBuildInfo("9.0.1")},
			want:  []finding{},
		},
		{
			name:  "mongodb_version 9.0.1 183 days before EOL",
			check: "mongodb_version",
			today: "2031-05-01",
			docs:  [][]map[string]any{mongoBuildInfo("9.0.1")},
			want:  []finding{mongoEOL("warning", "9.0.1", "9.0", "2031-10-31")},
		},
		{
			name:  "mongodb_version 9.0.1 day after EOL",
			check: "mongodb_version",
			today: "2031-11-01",
			docs:  [][]map[string]any{mongoBuildInfo("9.0.1")},
			want:  []finding{mongoEOL("error", "9.0.1", "9.0", "2031-10-31")},
		},
		{
			name:  "mongodb_version 99.0.0 not in tables",
			check: "mongodb_version",
			today: today,
			docs:  [][]map[string]any{mongoBuildInfo("99.0.0")},
			want:  []finding{},
		},
		{
			name:  "mongodb_version no rows",
			check: "mongodb_version",
			today: today,
			docs:  [][]map[string]any{{}},
			want:  []finding{},
		},
		{
			name:         "postgresql_version_check 130005 past EOL gives only the EOL error",
			check:        "postgresql_version_check",
			today:        today,
			docs:         [][]map[string]any{pgVersionRows(130005)},
			want:         []finding{pgEOL("error", "13.5", "13", "2025-11-13")},
			descContains: []string{"2025-11-13", "supported major version"},
			descExcludes: []string{pgVersion(tableInt(t, pgLatest, "13")), "newer minor version", "includes security fixes"},
		},
		{
			name:  "postgresql_version_check 110022 past EOL",
			check: "postgresql_version_check",
			today: today,
			docs:  [][]map[string]any{pgVersionRows(110022)},
			want:  []finding{pgEOL("error", "11.22", "11", "2023-11-09")},
		},
		{
			name:  "postgresql_version_check 120022 past EOL",
			check: "postgresql_version_check",
			today: today,
			docs:  [][]map[string]any{pgVersionRows(120022)},
			want:  []finding{pgEOL("error", "12.22", "12", "2024-11-21")},
		},
		{
			name:  "postgresql_version_check 100023 past EOL",
			check: "postgresql_version_check",
			today: today,
			docs:  [][]map[string]any{pgVersionRows(100023)},
			want:  []finding{pgEOL("error", "10.23", "10", "2022-11-10")},
		},
		{
			name:  "postgresql_version_check 90624 past EOL",
			check: "postgresql_version_check",
			today: today,
			docs:  [][]map[string]any{pgVersionRows(90624)},
			want:  []finding{pgEOL("error", "9.6.24", "9.6", "2021-11-11")},
		},
		{
			name:  "postgresql_version_check 140024 day before EOL",
			check: "postgresql_version_check",
			today: "2026-11-11",
			docs:  [][]map[string]any{pgVersionRows(140024)},
			want:  []finding{pgEOL("warning", "14.24", "14", "2026-11-12")},
		},
		{
			name:  "postgresql_version_check 140024 day after EOL",
			check: "postgresql_version_check",
			today: "2026-11-13",
			docs:  [][]map[string]any{pgVersionRows(140024)},
			want:  []finding{pgEOL("error", "14.24", "14", "2026-11-12")},
		},
		{
			name:         "postgresql_version_check 170005",
			check:        "postgresql_version_check",
			today:        today,
			docs:         [][]map[string]any{pgVersionRows(170005)},
			want:         []finding{newerPG("17.5", pgVersion(tableInt(t, pgLatest, "17")))},
			descContains: []string{"Current version is 17.5, the latest release of PostgreSQL 17 is " + pgVersion(tableInt(t, pgLatest, "17")) + ".", "PostgreSQL 17 reaches end of life on " + pgExpires["17"], "security fixes", "https://www.postgresql.org/support/security/"},
			descExcludes: []string{"newer major version"},
		},
		{
			name:         "postgresql_version_check 170005 on EOL day",
			check:        "postgresql_version_check",
			today:        pgExpires["17"],
			docs:         [][]map[string]any{pgVersionRows(170005)},
			want:         []finding{pgEOL("warning", "17.5", "17", pgExpires["17"])},
			descContains: []string{"newer minor version (" + pgVersion(tableInt(t, pgLatest, "17")) + ")", "security fixes"},
		},
		{
			name:         "postgresql_version_check latest 17 on EOL day",
			check:        "postgresql_version_check",
			today:        pgExpires["17"],
			docs:         [][]map[string]any{pgVersionRows(tableInt(t, pgLatest, "17"))},
			want:         []finding{pgEOL("warning", pgVersion(tableInt(t, pgLatest, "17")), "17", pgExpires["17"])},
			descExcludes: []string{"newer minor version", "includes security fixes"},
		},
		{
			name:         "postgresql_version_check 170005 day after EOL",
			check:        "postgresql_version_check",
			today:        "2029-11-09",
			docs:         [][]map[string]any{pgVersionRows(170005)},
			want:         []finding{pgEOL("error", "17.5", "17", "2029-11-08")},
			descExcludes: []string{pgVersion(tableInt(t, pgLatest, "17"))},
		},
		{
			name:         "postgresql_version_check latest 16",
			check:        "postgresql_version_check",
			today:        today,
			docs:         [][]map[string]any{pgVersionRows(tableInt(t, pgLatest, "16"))},
			want:         []finding{newerPG(pgVersion(tableInt(t, pgLatest, "16")), pgVersion(tableInt(t, pgLatest, pgLatestSeries)))},
			descContains: []string{"the latest release of PostgreSQL 16. A newer major version, " + pgVersion(tableInt(t, pgLatest, pgLatestSeries)) + ", is available.", "PostgreSQL 16 reaches end of life on " + pgExpires["16"], "security fixes"},
		},
		{
			name:  "postgresql_version_check latest of the latest major",
			check: "postgresql_version_check",
			today: today,
			docs:  [][]map[string]any{pgVersionRows(tableInt(t, pgLatest, pgLatestSeries))},
			want:  []finding{},
		},
		{
			name:  "postgresql_version_check 990000 not in tables",
			check: "postgresql_version_check",
			today: today,
			docs:  [][]map[string]any{pgVersionRows(990000)},
			want:  []finding{},
		},
		{
			name:  "postgresql_version_check no rows",
			check: "postgresql_version_check",
			today: today,
			docs:  [][]map[string]any{{}},
			want:  []finding{},
		},
		{
			name:  "mongodb_multiple_services two pids without cluster label",
			check: "mongodb_multiple_services",
			today: today,
			docs: [][]map[string]any{append(
				promVector(map[string]string{"node_name": "n1", "service_name": "s1", "rs_nm": "rs"}, "101"),
				promVector(map[string]string{"node_name": "n1", "service_name": "s2", "rs_nm": "rs2"}, "202")...)},
			want: []finding{{Severity: "warning", Summary: "MongoDB - multiple mongod services running"}},
		},
		{
			name:  "mongodb_multiple_services two standalones on one node",
			check: "mongodb_multiple_services",
			today: today,
			docs: [][]map[string]any{append(
				promVector(map[string]string{"node_name": "n1", "service_name": "s1", "rs_nm": ""}, "101"),
				promVector(map[string]string{"node_name": "n1", "service_name": "s2", "rs_nm": ""}, "202")...)},
			want: []finding{{Severity: "warning", Summary: "MongoDB - multiple mongod services running"}},
		},
		{
			name:  "mongodb_swap_allocation no swap metric",
			check: "mongodb_swap_allocation",
			today: today,
			docs:  [][]map[string]any{promVector(map[string]string{"node_name": "n1"}, "8"), {}, {}},
			want:  []finding{},
		},
		{
			name:  "mongodb_swap_allocation standalone without swap",
			check: "mongodb_swap_allocation",
			today: today,
			docs:  [][]map[string]any{promVector(map[string]string{"node_name": "n1"}, "8"), promVector(map[string]string{"node_name": "n1"}, "0"), {}},
			want:  []finding{{Severity: "warning", Summary: "Need to allocate swap memory and set vm.swappiness."}},
		},
		{
			name:  "mongodb_swap_allocation arbiter without swap",
			check: "mongodb_swap_allocation",
			today: today,
			docs:  [][]map[string]any{promVector(map[string]string{"node_name": "n1"}, "8"), promVector(map[string]string{"node_name": "n1"}, "0"), promVector(map[string]string{"node_name": "n1"}, "7")},
			want:  []finding{},
		},
		{
			name:  "mongodb_swap_allocation all empty",
			check: "mongodb_swap_allocation",
			today: today,
			docs:  [][]map[string]any{{}, {}, {}},
			want:  []finding{},
		},
		{
			name:  "mongodb_journal no options 8.0.4",
			check: "mongodb_journal",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{}), mongoBuildInfo("8.0.4")},
			want:  []finding{},
		},
		{
			name:  "mongodb_journal no options 6.0.19",
			check: "mongodb_journal",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{}), mongoBuildInfo("6.0.19")},
			want:  []finding{},
		},
		{
			name:  "mongodb_journal nested disabled 6.0.19",
			check: "mongodb_journal",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"storage": map[string]any{"journal": map[string]any{"enabled": "false"}}}), mongoBuildInfo("6.0.19")},
			want:  []finding{{Severity: "warning", Summary: "MongoDB journaling is disabled"}},
		},
		{
			name:  "mongodb_journal dotted disabled 6.0.19",
			check: "mongodb_journal",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"storage.journal": map[string]any{"enabled": "false"}}), mongoBuildInfo("6.0.19")},
			want:  []finding{{Severity: "warning", Summary: "MongoDB journaling is disabled"}},
		},
		{
			name:  "mongodb_journal nested disabled bool 6.0.19",
			check: "mongodb_journal",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"storage": map[string]any{"journal": map[string]any{"enabled": false}}}), mongoBuildInfo("6.0.19")},
			want:  []finding{{Severity: "warning", Summary: "MongoDB journaling is disabled"}},
		},
		{
			name:  "mongodb_journal nested enabled bool 6.0.19",
			check: "mongodb_journal",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"storage": map[string]any{"journal": map[string]any{"enabled": true}}}), mongoBuildInfo("6.0.19")},
			want:  []finding{},
		},
		{
			name:  "mongodb_journal nested disabled 8.0.4",
			check: "mongodb_journal",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"storage": map[string]any{"journal": map[string]any{"enabled": "false"}}}), mongoBuildInfo("8.0.4")},
			want:  []finding{},
		},
		{
			name:  "mongodb_journal nested disabled without buildInfo",
			check: "mongodb_journal",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"storage": map[string]any{"journal": map[string]any{"enabled": "false"}}}), {}},
			want:  []finding{{Severity: "warning", Summary: "MongoDB journaling is disabled"}},
		},
		{
			name:  "mongodb_loglevel no options",
			check: "mongodb_loglevel",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{})},
			want:  []finding{},
		},
		{
			name:  "mongodb_loglevel logLevel 0",
			check: "mongodb_loglevel",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"setParameter": map[string]any{"logLevel": "0"}})},
			want:  []finding{},
		},
		{
			name:  "mongodb_loglevel logLevel int64 0",
			check: "mongodb_loglevel",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"setParameter": map[string]any{"logLevel": int64(0)}})},
			want:  []finding{},
		},
		{
			name:  "mongodb_loglevel systemLog verbosity 0",
			check: "mongodb_loglevel",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"systemLog": map[string]any{"verbosity": int64(0)}})},
			want:  []finding{},
		},
		{
			name:  "mongodb_loglevel logLevel 2",
			check: "mongodb_loglevel",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"setParameter": map[string]any{"logLevel": "2"}})},
			want:  []finding{{Severity: "warning", Summary: "MongoDB is not using the default log level."}},
		},
		{
			name:  "mongodb_loglevel systemLog verbosity 1",
			check: "mongodb_loglevel",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"systemLog": map[string]any{"verbosity": int64(1)}})},
			want:  []finding{{Severity: "warning", Summary: "MongoDB is not using the default log level."}},
		},
		{
			name:  "mongodb_maxsessions no setParameter",
			check: "mongodb_maxsessions",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{})},
			want:  []finding{},
		},
		{
			name:  "mongodb_maxsessions setParameter without maxSessions",
			check: "mongodb_maxsessions",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"setParameter": map[string]any{"authenticationMechanisms": "SCRAM-SHA-1"}})},
			want:  []finding{},
		},
		{
			name:  "mongodb_maxsessions default int64",
			check: "mongodb_maxsessions",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"setParameter": map[string]any{"maxSessions": int64(1000000)}})},
			want:  []finding{},
		},
		{
			name:  "mongodb_maxsessions string 2000000",
			check: "mongodb_maxsessions",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"setParameter": map[string]any{"maxSessions": "2000000"}})},
			want:  []finding{{Severity: "warning", Summary: "MongoDB is using maxSessions - 2000000, which is more than the default value - 1000000"}},
		},
		{
			name:  "mongodb_maxsessions int64 2000000",
			check: "mongodb_maxsessions",
			today: today,
			docs:  [][]map[string]any{mongoCmdLineOpts(map[string]any{"setParameter": map[string]any{"maxSessions": int64(2000000)}})},
			want:  []finding{{Severity: "warning", Summary: "MongoDB is using maxSessions - 2000000, which is more than the default value - 1000000"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			results := runBuiltinCheck(t, tc.check, tc.today, tc.docs...)
			assert.Equal(t, tc.want, findings(results))

			for _, r := range results {
				for _, sub := range tc.descContains {
					assert.Contains(t, r.Description, sub)
				}

				for _, sub := range tc.descExcludes {
					assert.NotContains(t, r.Description, sub)
				}
			}
		})
	}
}

// TestLatestVersionTables checks that every latest version belongs to the series that keys it, such as 80313 under "8.3".
func TestLatestVersionTables(t *testing.T) {
	for series, v := range scriptTable(t, "mongodb_version", `"version_check": {`) {
		assert.True(t, strings.HasPrefix(mongoVersion(tableInt(t, map[string]string{series: v}, series)), series+"."), "mongodb_version %s: %s", series, v)
	}

	for series, v := range scriptTable(t, "postgresql_version_check", "latest_versions = {") {
		num := tableInt(t, map[string]string{series: v}, series)
		got := strconv.Itoa(num / 10000)
		if num < 100000 {
			got = fmt.Sprintf("%d.%d", num/10000, num/100%100)
		}
		assert.Equal(t, series, got, "postgresql_version_check %s: %s", series, v)
	}
}

func TestRetiredChecks(t *testing.T) {
	for _, name := range []string{
		"mongodb_unsupported_version",
		"mongodb_cve_version",
		"postgresql_unsupported_check",
		"postgresql_cve_check",
		"mysql_version_eol_57",
		"mongodb_EOL",
		"mysql_unsupported_version_check",
		"postgresql_eol_check",
	} {
		_, err := os.Stat(filepath.Join("..", "..", "data", "checks", name+".yml"))
		assert.ErrorIs(t, err, os.ErrNotExist, name)
	}
}
