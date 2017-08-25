// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package prometheus

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/utils/logger"
)

const testdata = "../../testdata/prometheus/"

func getPrometheus(t *testing.T, ctx context.Context) *Service {
	// TODO t.Helper()

	svc, err := NewService(filepath.Join(testdata, "prometheus.yml"), "http://127.0.0.1:9090/", "promtool")
	require.NoError(t, err)
	require.NoError(t, svc.Check(ctx))
	return svc
}

func TestPrometheusConfig(t *testing.T) {
	ctx, _ := logger.Set(context.Background(), "TestPrometheusConfig")
	p := getPrometheus(t, ctx)

	// always restore original after test
	before, err := ioutil.ReadFile(p.configPath)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, ioutil.WriteFile(p.configPath, before, 0666))
	}()

	// check that we can write it exactly as it was
	c, err := p.loadConfig()
	assert.NoError(t, err)
	assert.NoError(t, p.saveConfig(c))
	after, err := ioutil.ReadFile(p.configPath)
	require.NoError(t, err)
	b, a := string(before), string(after)
	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A: difflib.SplitLines(b),
		B: difflib.SplitLines(a),
	})
	assert.Equal(t, b, a, "%s", diff)

	// specifically check that we can read secrets
	assert.Equal(t, "pmm", c.ScrapeConfigs[2].HTTPClientConfig.BasicAuth.Password)
}

func TestPrometheusRules(t *testing.T) {
	t.Skip("TODO")

	ctx, _ := logger.Set(context.Background(), "TestPrometheusRules")
	p := getPrometheus(t, ctx)

	alerts, err := p.ListAlertRules(ctx)
	require.NoError(t, err)
	require.Len(t, alerts, 2)
	alerts[0].Text = "" // FIXME
	alerts[1].Text = "" // FIXME
	expected := []AlertRule{
		{"InstanceDown", filepath.Join(testdata, "alerts", "InstanceDown.rule"), "", false},
		{"Something", filepath.Join(testdata, "alerts", "Something.rule.disabled"), "", true},
	}
	assert.Equal(t, expected, alerts)

	defer func() {
		require.NoError(t, p.DeleteAlert(ctx, "TestPrometheus"))
		require.EqualError(t, p.DeleteAlert(ctx, "TestPrometheus"), os.ErrNotExist.Error())
	}()

	rule := &AlertRule{
		Name: "TestPrometheus",
		Text: "ALERT TestPrometheus IF up == 0",
	}
	require.NoError(t, p.PutAlert(ctx, rule))
	actual, err := p.GetAlert(ctx, "TestPrometheus")
	require.NoError(t, err)
	rule.FilePath = "../testdata/prometheus/alerts/TestPrometheus.rule"
	assert.Equal(t, rule, actual)
}

func TestPrometheusScrapeJobs(t *testing.T) {
	ctx, _ := logger.Set(context.Background(), "TestPrometheusScrapeJobs")
	p := getPrometheus(t, ctx)

	// always restore original after test
	before, err := ioutil.ReadFile(p.configPath)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, ioutil.WriteFile(p.configPath, before, 0666))
	}()

	jobs, err := p.ListScrapeJobs(ctx)
	require.NoError(t, err)
	require.Len(t, jobs, 3)
	expected := []ScrapeJob{
		{"prometheus", "1m", "30s", "/metrics", "http", []string{"127.0.0.1:9090"}},
		{"alertmanager", "10s", "5s", "/metrics", "http", []string{"127.0.0.1:9093"}},
		{"linux", "30s", "15s", "/metrics", "http", []string{"localhost:9100"}},
	}
	assert.Equal(t, expected, jobs)

	defer func() {
		require.NoError(t, p.DeleteScrapeJob(ctx, "test_job"))
		require.EqualError(t, p.DeleteScrapeJob(ctx, "test_job"), os.ErrNotExist.Error())
	}()

	// other fields are filled by defaults
	job := &ScrapeJob{
		Name:          "test_job",
		StatisTargets: []string{"127.0.0.1:12345", "127.0.0.2:12345"},
	}
	require.NoError(t, p.PutScrapeJob(ctx, job))
	actual, err := p.GetScrapeJob(ctx, "test_job")
	require.NoError(t, err)
	job = &ScrapeJob{"test_job", "30s", "15s", "/metrics", "http", []string{"127.0.0.1:12345", "127.0.0.2:12345"}}
	assert.Equal(t, job, actual)
}
