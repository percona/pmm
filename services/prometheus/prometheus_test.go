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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/utils/tests"
)

func TestPrometheusConfig(t *testing.T) {
	p, ctx, before := SetupTest(t)
	defer TearDownTest(t, p, before)

	// check that we can write it exactly as it was
	c, err := p.loadConfig()
	assert.NoError(t, err)
	assert.NoError(t, p.saveConfigAndReload(ctx, c))
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

	// check that invalid configuration is reverted
	c.ScrapeConfigs[0].ScrapeInterval = model.Duration(time.Second)
	err = p.saveConfigAndReload(ctx, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `scrape timeout greater than scrape interval`)
	after, err = ioutil.ReadFile(p.configPath)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestPrometheusRules(t *testing.T) {
	t.Skip("TODO")

	p, ctx, before := SetupTest(t)
	defer TearDownTest(t, p, before)

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

func TestPrometheusScrapeConfigs(t *testing.T) {
	p, ctx, before := SetupTest(t)
	defer TearDownTest(t, p, before)

	cfgs, err := p.ListScrapeConfigs(ctx)
	require.NoError(t, err)
	require.Empty(t, cfgs)

	actual, err := p.GetScrapeConfig(ctx, "no_such_config")
	assert.Nil(t, actual)
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "no_such_config" not found`), err)

	defer func() {
		err = p.DeleteScrapeConfig(ctx, "test_config")
		require.NoError(t, err)

		err = p.DeleteScrapeConfig(ctx, "test_config")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "test_config" not found`), err)
	}()

	cfg := &ScrapeConfig{
		JobName: "test_config",
		StaticConfigs: []StaticConfig{
			{[]string{"127.0.0.1:12345", "127.0.0.2:12345"}, nil},
		},
	}
	err = p.CreateScrapeConfig(ctx, cfg)
	require.NoError(t, err)

	err = p.CreateScrapeConfig(ctx, cfg)
	tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `scrape config with job name "test_config" already exist`), err)

	err = p.CreateScrapeConfig(ctx, &ScrapeConfig{JobName: "prometheus"})
	tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, `scrape config with job name "prometheus" is built-in`), err)

	// other fields are filled by global values or defaults
	actual, err = p.GetScrapeConfig(ctx, "test_config")
	require.NoError(t, err)
	expected := &ScrapeConfig{
		JobName:        "test_config",
		ScrapeInterval: "30s",
		ScrapeTimeout:  "15s",
		MetricsPath:    "/metrics",
		Scheme:         "http",
		StaticConfigs: []StaticConfig{
			{[]string{"127.0.0.1:12345", "127.0.0.2:12345"}, nil},
		},
	}
	assert.Equal(t, expected, actual)

	cfg.HonorLabels = true
	cfg.RelabelConfigs = []RelabelConfig{{
		TargetLabel: "job",
		Replacement: "test_config_relabeled",
	}}
	err = p.SetScrapeConfigs(ctx, false, cfg)
	require.NoError(t, err)

	actual, err = p.GetScrapeConfig(ctx, "test_config")
	require.NoError(t, err)
	expected = &ScrapeConfig{
		JobName:        "test_config",
		ScrapeInterval: "30s",
		ScrapeTimeout:  "15s",
		MetricsPath:    "/metrics",
		HonorLabels:    true,
		Scheme:         "http",
		StaticConfigs: []StaticConfig{
			{[]string{"127.0.0.1:12345", "127.0.0.2:12345"}, nil},
		},
		RelabelConfigs: []RelabelConfig{
			{"job", "test_config_relabeled"},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestPrometheusStaticTargets(t *testing.T) {
	p, ctx, before := SetupTest(t)
	defer TearDownTest(t, p, before)

	cfg := &ScrapeConfig{
		JobName:        "test_config",
		ScrapeInterval: "2s",
		ScrapeTimeout:  "1s",
		MetricsPath:    "/external",
		Scheme:         "http",
		StaticConfigs: []StaticConfig{
			{[]string{"127.0.0.1:12345"}, nil},
		},
	}
	err := p.CreateScrapeConfig(ctx, cfg)
	require.NoError(t, err)

	// add the same targets twice: no error, no duplicate
	for i := 0; i < 2; i++ {
		err = p.AddStaticTargets(ctx, "test_config", []string{"127.0.0.2:12345", "127.0.0.2:12345"})
		require.NoError(t, err)
	}

	actual, err := p.GetScrapeConfig(ctx, "test_config")
	require.NoError(t, err)
	cfg.StaticConfigs = []StaticConfig{
		{[]string{"127.0.0.1:12345", "127.0.0.2:12345"}, nil},
	}
	assert.Equal(t, cfg, actual)

	err = p.AddStaticTargets(ctx, "no_such_config", []string{"127.0.0.2:12345", "127.0.0.2:12345"})
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "no_such_config" not found`), err)

	err = p.AddStaticTargets(ctx, "prometheus", []string{"127.0.0.2:12345", "127.0.0.2:12345"})
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "prometheus" not found`), err)

	// remove the same target twice: no error
	for i := 0; i < 2; i++ {
		err = p.RemoveStaticTargets(ctx, "test_config", []string{"127.0.0.1:12345"})
		require.NoError(t, err)
	}

	actual, err = p.GetScrapeConfig(ctx, "test_config")
	require.NoError(t, err)
	cfg.StaticConfigs = []StaticConfig{
		{[]string{"127.0.0.2:12345"}, nil},
	}
	assert.Equal(t, cfg, actual)

	err = p.RemoveStaticTargets(ctx, "test_config", []string{"127.0.0.2:12345"})
	require.NoError(t, err)

	actual, err = p.GetScrapeConfig(ctx, "test_config")
	require.NoError(t, err)
	cfg.StaticConfigs = nil
	assert.Equal(t, cfg, actual)

	err = p.RemoveStaticTargets(ctx, "no_such_config", []string{"127.0.0.2:12345", "127.0.0.2:12345"})
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "no_such_config" not found`), err)

	err = p.RemoveStaticTargets(ctx, "prometheus", []string{"127.0.0.2:12345", "127.0.0.2:12345"})
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "prometheus" not found`), err)
}

// https://jira.percona.com/browse/PMM-1310?focusedCommentId=196688
func TestPrometheusBadScrapeConfig(t *testing.T) {
	p, ctx, before := SetupTest(t)
	defer TearDownTest(t, p, before)

	// https://jira.percona.com/browse/PMM-1636
	cfg := &ScrapeConfig{
		JobName: "10.10.11.50:9187",
	}
	err := p.CreateScrapeConfig(ctx, cfg)
	tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `job_name: invalid format. Job name must be 2 to 60 characters long, characters long, contain only letters, numbers, and symbols '-', '_', and start with a letter.`), err)

	cfg = &ScrapeConfig{
		JobName:        "test_config",
		ScrapeInterval: "1s",
		ScrapeTimeout:  "5s",
	}
	err = p.CreateScrapeConfig(ctx, cfg)
	tests.AssertGRPCError(t, status.New(codes.Aborted, `scrape timeout greater than scrape interval for scrape config with job name "test_config"`), err)

	cfgs, err := p.ListScrapeConfigs(ctx)
	require.NoError(t, err)
	assert.Empty(t, cfgs)

	actual, err := p.GetScrapeConfig(ctx, "test_config")
	assert.Nil(t, actual)
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "test_config" not found`), err)

	err = p.DeleteScrapeConfig(ctx, "test_config")
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "test_config" not found`), err)
}

// https://jira.percona.com/browse/PMM-1310?focusedCommentId=196689
func TestPrometheusReadDefaults(t *testing.T) {
	p, ctx, before := SetupTest(t)
	defer TearDownTest(t, p, before)

	cfg := &ScrapeConfig{
		JobName: "test_config",
	}
	err := p.CreateScrapeConfig(ctx, cfg)
	assert.NoError(t, err)

	expected := &ScrapeConfig{
		JobName:        "test_config",
		ScrapeInterval: "30s",
		ScrapeTimeout:  "15s",
		MetricsPath:    "/metrics",
		Scheme:         "http",
	}

	cfgs, err := p.ListScrapeConfigs(ctx)
	require.NoError(t, err)
	assert.Equal(t, []ScrapeConfig{*expected}, cfgs)

	actual, err := p.GetScrapeConfig(ctx, "test_config")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}
