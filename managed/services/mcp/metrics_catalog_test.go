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

package mcp

import (
	"bufio"
	"compress/gzip"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exporterFamily is one metric of a captured exporter output.
type exporterFamily struct {
	typ    string
	labels map[string]bool
}

var sampleLine = regexp.MustCompile(`^([a-zA-Z_:][a-zA-Z0-9_:]*)(\{(.*)\})? `)

var labelName = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)="`)

// readExporterFixture reads testdata/exporters/<engine>.prom.gz into its metrics, by name.
func readExporterFixture(t *testing.T, engine string) map[string]*exporterFamily {
	t.Helper()

	f, err := os.Open(filepath.Join("testdata", "exporters", engine+".prom.gz"))
	require.NoError(t, err)
	defer f.Close()

	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gz.Close()

	families := map[string]*exporterFamily{}
	family := func(name string) *exporterFamily {
		if families[name] == nil {
			families[name] = &exporterFamily{typ: "untyped", labels: map[string]bool{}}
		}
		return families[name]
	}

	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if rest, ok := strings.CutPrefix(line, "# TYPE "); ok {
			fields := strings.Fields(rest)
			require.Len(t, fields, 2, line)
			family(fields[0]).typ = fields[1]
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		m := sampleLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		fam := family(m[1])
		for _, l := range labelName.FindAllStringSubmatch(m[3], -1) {
			fam.labels[l[1]] = true
		}
	}
	require.NoError(t, sc.Err())
	require.NotEmpty(t, families)
	return families
}

func TestMetricsCatalogMatchesExporters(t *testing.T) {
	t.Parallel()

	prefixes := map[string]string{engineMySQL: "mysql_", enginePostgreSQL: "pg_", engineMongoDB: "mongodb_", engineNode: "node_"}
	fixtures := map[string]map[string]*exporterFamily{}
	for _, engine := range metricsEngines {
		fixtures[engine] = readExporterFixture(t, engine)
	}

	perEngine := map[string]int{}
	seen := map[string]bool{}
	for _, m := range metricsCatalog {
		perEngine[m.Engine]++
		assert.False(t, seen[m.Name], "%s: listed twice", m.Name)
		seen[m.Name] = true

		fixture, ok := fixtures[m.Engine]
		require.True(t, ok, "%s: unknown engine '%s'", m.Name, m.Engine)
		assert.True(t, strings.HasPrefix(m.Name, prefixes[m.Engine]), "%s: not a %s metric", m.Name, m.Engine)
		assert.NotEmpty(t, m.Meaning, m.Name)
		assert.NotContains(t, m.Meaning, "\n", m.Name)
		assert.NotEmpty(t, m.Unit, m.Name)

		fam, ok := fixture[m.Name]
		if !assert.True(t, ok, "%s: not in the %s exporter output", m.Name, m.Engine) {
			continue
		}

		switch m.Type {
		case "counter":
			assert.Contains(t, []string{"counter", "untyped"}, fam.typ, "%s: a counter entry, typed %s by the exporter", m.Name, fam.typ)
		case "gauge":
			assert.Contains(t, []string{"gauge", "untyped"}, fam.typ, "%s: a gauge entry, typed %s by the exporter", m.Name, fam.typ)
			assert.False(t, strings.HasSuffix(m.Name, "_total"), "%s: a _total metric is a counter", m.Name)
		default:
			assert.Fail(t, "type must be counter or gauge", "%s: '%s'", m.Name, m.Type)
		}
		if strings.HasSuffix(m.Name, "_total") {
			assert.Equal(t, "counter", m.Type, m.Name)
		}

		got := make([]string, 0, len(fam.labels))
		for l := range fam.labels {
			got = append(got, l)
		}
		assert.ElementsMatch(t, got, m.Labels, "%s: labels differ from the exporter output", m.Name)
	}

	for _, engine := range metricsEngines {
		assert.NotZero(t, perEngine[engine], engine)
	}
}

func TestMetricsCatalogQueriesPass(t *testing.T) {
	t.Parallel()

	step := chooseStep(time.Hour, 0)
	for _, m := range metricsCatalog {
		scope := `service_name="<service>"`
		replacer := strings.NewReplacer("<service>", "shop-mysql", "<node>", "db-host-1")
		if m.Engine == engineNode {
			scope = `node_name="<node>"`
		}
		assert.Contains(t, m.Query, scope, m.Name)
		assert.Contains(t, m.Query, m.Name+"{", m.Name)

		if m.Type == "counter" {
			assert.Contains(t, m.Query, "rate(", "%s: a counter's query takes its rate", m.Name)
		} else {
			assert.NotContains(t, m.Query, "rate(", "%s: a gauge's query takes no rate", m.Name)
		}

		query := replacer.Replace(m.Query)
		assert.NotContains(t, query, "<", m.Name)
		_, err := checkExpr(query, time.Hour, step)
		assert.NoError(t, err, "%s: %s", m.Name, query)
	}
}

func TestMetricsCatalogUnitsMatchQueries(t *testing.T) {
	t.Parallel()

	for name, want := range map[string]struct{ unit, query string }{
		"node_vmstat_pgpgin":                  {"Bps", `rate(node_vmstat_pgpgin{node_name="<node>"}[5m]) * 1024`},
		"node_vmstat_pgpgout":                 {"Bps", `rate(node_vmstat_pgpgout{node_name="<node>"}[5m]) * 1024`},
		"pg_stat_user_tables_last_autovacuum": {"dateTimeAsSystem", `max(pg_stat_user_tables_last_autovacuum{service_name="<service>"}) * 1000`},
		"node_boot_time_seconds":              {"dateTimeAsSystem", `node_boot_time_seconds{node_name="<node>"} * 1000`},
		"node_time_seconds":                   {"dateTimeAsSystem", `node_time_seconds{node_name="<node>"} * 1000`},
	} {
		i, ok := metricInfoByName(name)
		require.True(t, ok, name)
		assert.Equal(t, want.unit, i.Unit, name)
		assert.Equal(t, want.query, i.Query, name)
	}
}

func TestMetricInfoByName(t *testing.T) {
	t.Parallel()

	i, ok := metricInfoByName("mysql_global_status_threads_connected")
	require.True(t, ok)
	assert.Equal(t, engineMySQL, i.Engine)
	assert.Equal(t, "gauge", i.Type)

	_, ok = metricInfoByName("mysql_global_status_qcache_hits")
	assert.False(t, ok)
}
