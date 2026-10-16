// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package models_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestLogParserPresetsServerLogs(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	q := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf)).Querier

	t.Run("postgres starts an entry only at a top-level line", func(t *testing.T) {
		pg := mustPreset(t, q, "postgres")
		re := regexp.MustCompile(pg.LineStartPattern)
		assert.True(t, re.MatchString(`2026-10-16 12:38:54.405 UTC [233190] ERROR:  deadlock detected`))
		assert.True(t, re.MatchString(`2026-10-16 12:38:54.405 UTC [233190] LOG:  checkpoint starting: wal`))
		for _, l := range []string{"DETAIL", "HINT", "STATEMENT", "CONTEXT", "QUERY", "LOCATION"} {
			assert.False(t, re.MatchString(`2026-10-16 12:38:54.405 UTC [233190] `+l+`:  x`), l)
		}
	})

	t.Run("pmm_agent parses times with and without milliseconds", func(t *testing.T) {
		agent := mustPreset(t, q, "pmm_agent")
		assert.Contains(t, agent.OperatorYAML, `layout: '2006-01-02T15:04:05Z07:00'`)
		for _, ts := range []string{"2026-10-16T19:56:51Z", "2026-10-16T19:56:49.437+00:00"} {
			_, err := time.Parse("2006-01-02T15:04:05Z07:00", ts)
			require.NoError(t, err, ts)
		}
	})

	t.Run("victoriametrics parses tab-separated lines", func(t *testing.T) {
		vm := mustPreset(t, q, "victoriametrics")
		assert.True(t, vm.BuiltIn)
		require.NoError(t, models.ValidateLogParserOperatorYAML(vm.OperatorYAML))
		vmRe := regexp.MustCompile(regexParserRegex(t, vm.OperatorYAML))
		m := vmRe.FindStringSubmatch("2026-10-16T13:20:06.550Z\twarn\t/app/vmalert/config/log/logger.go:44\tno groups found in /srv/prometheus/rules/*.yml")
		require.NotNil(t, m)
		assert.Equal(t, "warn", m[vmRe.SubexpIndex("level")])
		assert.Equal(t, "no groups found in /srv/prometheus/rules/*.yml", m[vmRe.SubexpIndex("message")])
	})
}

func mustPreset(t *testing.T, q *reform.Querier, name string) *models.LogParserPreset {
	t.Helper()
	p, err := models.FindLogParserPresetByName(q, name)
	require.NoError(t, err)
	require.NotNil(t, p, name)
	return p
}

// regexParserRegex returns the regex of the regex_parser operator in a preset's operators.
func regexParserRegex(t *testing.T, operatorYAML string) string {
	t.Helper()
	var ops []map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(operatorYAML), &ops))
	for _, op := range ops {
		if op["type"] == "regex_parser" {
			re, ok := op["regex"].(string)
			require.True(t, ok)
			return re
		}
	}
	require.FailNow(t, "no regex_parser operator")
	return ""
}
