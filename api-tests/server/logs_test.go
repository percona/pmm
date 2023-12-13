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
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	serverClient "github.com/percona/pmm/api/serverpb/json/client"
	"github.com/percona/pmm/api/serverpb/json/client/server"
)

func TestDownloadLogs(t *testing.T) {
	var buf bytes.Buffer
	res, err := serverClient.Default.Server.Logs(&server.LogsParams{
		Context: pmmapitests.Context,
	}, &buf)
	require.NoError(t, err)
	require.NotNil(t, res)

	r := bytes.NewReader(buf.Bytes())
	zipR, err := zip.NewReader(r, r.Size())
	assert.NoError(t, err)

	expected := []string{
		"alertmanager.ini",
		"alertmanager.log",
		"clickhouse-server.log",
		"client/list.txt",
		"client/pmm-admin-version.txt",
		"client/pmm-agent-config.yaml",
		"client/pmm-agent-version.txt",
		"client/pmm-agent/pmm-agent.log",
		"client/status.json",
		"grafana.log",
		"installed.json",
		"nginx.conf",
		"nginx.log",
		"pmm-agent.log",
		"pmm-agent.yaml",
		"pmm-managed.log",
		"pmm-ssl.conf",
		"pmm-update-perform-init.log",
		"pmm-version.txt",
		"pmm.conf",
		"pmm.ini",
		"postgresql14.log",
		"prometheus.base.yml",
		"qan-api2.ini",
		"qan-api2.log",
		"supervisorctl_status.log",
		"supervisord.conf",
		"supervisord.log",
		"systemctl_status.log",
		"victoriametrics-promscrape.yml",
		"victoriametrics.ini",
		"victoriametrics.log",
		"victoriametrics_targets.json",
		"vmalert.ini",
		"vmalert.log",
		"vmproxy.log",
	}

	if os.Getenv("PERCONA_TEST_DBAAS") == "1" {
		expected = append(expected, "dbaas-controller.log")
		sort.Strings(expected)
	}

	actual := make([]string, 0, len(zipR.File))
	for _, file := range zipR.File {
		// skip with dynamic IDs now @TODO use regex to match ~ "client/pmm-agent/NODE_EXPORTER 297b465c-a767-4bc5-809d-d394a83c7086.log"
		if strings.Contains(file.Name, "client/pmm-agent/") && file.Name != "client/pmm-agent/pmm-agent.log" {
			continue
		}

		actual = append(actual, file.Name)
	}

	sort.Strings(actual)
	assert.Equal(t, expected, actual)
}
