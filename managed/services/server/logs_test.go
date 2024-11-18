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

package server

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/logger"
)

var commonExpectedFiles = []string{
	"clickhouse-server.log",
	"grafana.log",
	"installed.json",
	"nginx.conf",
	"nginx.log",
	"pmm-agent.log",
	"pmm-agent.yaml",
	"pmm-managed.log",
	"pmm-ssl.conf",
	"pmm-update.log",
	"pmm-version.txt",
	"pmm.conf",
	"pmm.ini",
	"postgresql14.log",
	"qan-api2.ini",
	"qan-api2.log",
	"supervisorctl_status.log",
	"supervisord.conf",
	"supervisord.log",
	"victoriametrics-promscrape.yml",
	"victoriametrics.ini",
	"victoriametrics.log",
	"victoriametrics_targets.json",
	"vmalert.ini",
	"vmalert.log",
	"vmproxy.ini",
	"vmproxy.log",
}

func TestReadLog(t *testing.T) {
	f, err := os.CreateTemp("", "pmm-managed-supervisord-tests-")
	require.NoError(t, err)
	fNoNewLineEnding, err := os.CreateTemp("", "pmm-managed-supervisord-tests-")
	require.NoError(t, err)

	for i := range 10 { //nolint:typecheck
		fmt.Fprintf(f, "line #%03d\n", i)                // 10 bytes
		fmt.Fprintf(fNoNewLineEnding, "line #%03d\n", i) // 10 bytes
	}
	fmt.Fprintf(fNoNewLineEnding, "some string without new line")
	require.NoError(t, f.Close())
	require.NoError(t, fNoNewLineEnding.Close())

	defer os.Remove(f.Name())                //nolint:errcheck
	defer os.Remove(fNoNewLineEnding.Name()) //nolint:errcheck

	t.Run("LimitByLines", func(t *testing.T) {
		b, m, err := readLog(f.Name(), 5)
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now(), m, 5*time.Second)
		expected := []string{"line #005", "line #006", "line #007", "line #008", "line #009"}
		actual := strings.Split(strings.TrimSpace(string(b)), "\n")
		assert.Equal(t, expected, actual)
	})

	t.Run("LimitByLines - no new line ending", func(t *testing.T) {
		b, m, err := readLog(fNoNewLineEnding.Name(), 5)
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now(), m, 5*time.Second)
		expected := []string{"line #006", "line #007", "line #008", "line #009", "some string without new line"}
		actual := strings.Split(strings.TrimSpace(string(b)), "\n")
		assert.Equal(t, expected, actual)
	})
}

func TestReadLogUnlimited(t *testing.T) {
	f, err := os.CreateTemp("", "pmm-managed-supervisord-tests-")
	require.NoError(t, err)
	fNoNewLineEnding, err := os.CreateTemp("", "pmm-managed-supervisord-tests-")
	require.NoError(t, err)

	for i := range 10 { //nolint:typecheck
		fmt.Fprintf(f, "line #%03d\n", i)                // 10 bytes
		fmt.Fprintf(fNoNewLineEnding, "line #%03d\n", i) // 10 bytes
	}
	fmt.Fprintf(fNoNewLineEnding, "some string without new line")
	require.NoError(t, f.Close())
	require.NoError(t, fNoNewLineEnding.Close())

	defer os.Remove(f.Name())                //nolint:errcheck
	defer os.Remove(fNoNewLineEnding.Name()) //nolint:errcheck

	t.Run("UnlimitedLineCount", func(t *testing.T) {
		b, m, err := readLogUnlimited(f.Name())
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now(), m, 5*time.Second)
		expected := []string{"line #000", "line #001", "line #002", "line #003", "line #004", "line #005", "line #006", "line #007", "line #008", "line #009"}
		actual := strings.Split(strings.TrimSpace(string(b)), "\n")
		assert.Equal(t, expected, actual)
	})

	t.Run("UnlimitedLineCount - no new line ending", func(t *testing.T) {
		b, m, err := readLogUnlimited(fNoNewLineEnding.Name())
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now(), m, 5*time.Second)
		expected := []string{"line #000", "line #001", "line #002", "line #003", "line #004", "line #005", "line #006", "line #007", "line #008", "line #009", "some string without new line"}
		actual := strings.Split(strings.TrimSpace(string(b)), "\n")
		assert.Equal(t, expected, actual)
	})
}

func TestAddAdminSummary(t *testing.T) {
	t.Skip("FIXME")

	zipfile, err := os.CreateTemp("", "*-test.zip")
	assert.NoError(t, err)

	zw := zip.NewWriter(zipfile)
	err = addAdminSummary(context.Background(), zw)
	assert.NoError(t, err)

	assert.NoError(t, zw.Close())

	reader, err := zip.OpenReader(zipfile.Name())
	assert.NoError(t, err)

	hasClientDir := false
	for _, file := range reader.File {
		if filepath.Dir(file.Name) == "client" {
			hasClientDir = true
			break
		}
	}
	assert.True(t, hasClientDir)
}

func TestFiles(t *testing.T) {
	updater := &Updater{}
	params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
	require.NoError(t, err)
	l := NewLogs("2.4.5", updater, params)
	ctx := logger.Set(context.Background(), t.Name())

	files := l.files(ctx, nil, maxLogReadLines)
	actual := make([]string, 0, len(files))
	for _, f := range files {
		if f.Name == "prometheus.base.yml" {
			assert.EqualError(t, f.Err, "open /srv/prometheus/prometheus.base.yml: no such file or directory")
			continue
		}

		if f.Name == "supervisorctl_status.log" {
			assert.EqualError(t, f.Err, "exit status 3")
			// NOTE: this fails in supervisorctl v4+ if there are stopped services; it is not critical because the call succeeds
			actual = append(actual, f.Name)
			continue
		}

		assert.NoError(t, f.Err, "name = %q", f.Name)

		actual = append(actual, f.Name)
	}

	sort.Strings(actual)
	assert.Equal(t, commonExpectedFiles, actual)
}

func TestZip(t *testing.T) {
	t.Skip("FIXME")

	updater := &Updater{}
	params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
	require.NoError(t, err)
	l := NewLogs("2.4.5", updater, params)
	ctx := logger.Set(context.Background(), t.Name())

	var buf bytes.Buffer
	require.NoError(t, l.Zip(ctx, &buf, nil, -1))
	reader := bytes.NewReader(buf.Bytes())
	r, err := zip.NewReader(reader, reader.Size())
	require.NoError(t, err)

	additionalFiles := []string{
		"client/list.txt",
		"client/pmm-admin-version.txt",
		"client/pmm-agent-config.yaml",
		"client/pmm-agent-version.txt",
		"client/status.json",
		"client/pmm-agent/pmm-agent.log",
		"prometheus.base.yml",
	}
	// zip file includes client files
	expected := append(commonExpectedFiles, additionalFiles...) //nolint:gocritic

	actual := make([]string, 0, len(r.File))
	for _, f := range r.File {
		// skip with dynamic IDs now
		// TODO: use regex to match ~ "client/pmm-agent/NODE_EXPORTER 297b465c-a767-4bc5-809d-d394a83c7086.log"
		if strings.Contains(f.Name, "client/pmm-agent/") && f.Name != "client/pmm-agent/pmm-agent.log" {
			continue
		}

		assert.NotZero(t, f.Modified)

		actual = append(actual, f.Name)
	}

	sort.Strings(actual)
	sort.Strings(expected)
	assert.Equal(t, expected, actual)
}
