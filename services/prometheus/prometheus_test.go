// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package prometheus

import (
	"io/ioutil"
	"strings"
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
	ctx, p, before := SetupTest(t)
	defer TearDownTest(t, p, before)

	// check that we can write it exactly as it was
	c, err := p.loadConfig()
	assert.NoError(t, err)
	assert.NoError(t, p.saveConfigAndReload(ctx, c))
	after, err := ioutil.ReadFile(p.ConfigPath)
	require.NoError(t, err)
	beforeS, afterS := string(before), string(after)
	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(beforeS),
		FromFile: "Before",
		B:        difflib.SplitLines(afterS),
		ToFile:   "After",
		Context:  1,
	})
	require.NoError(t, err)
	require.Equal(t, strings.Split(beforeS, "\n"), strings.Split(afterS, "\n"), "%s", diff)
	require.Len(t, c.ScrapeConfigs, 14)

	// specifically check that we can read secrets
	require.NotNil(t, c.ScrapeConfigs[2].HTTPClientConfig.BasicAuth)
	assert.Equal(t, "pmm", c.ScrapeConfigs[2].HTTPClientConfig.BasicAuth.Password)

	// check that invalid configuration is reverted
	c.ScrapeConfigs[1].ScrapeInterval = model.Duration(time.Second)
	err = p.saveConfigAndReload(ctx, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `scrape timeout greater than scrape interval`)
	after, err = ioutil.ReadFile(p.ConfigPath)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestPrometheusScrapeConfigs(t *testing.T) {
	ctx, p, before := SetupTest(t)
	defer TearDownTest(t, p, before)

	cfgs, health, err := p.ListScrapeConfigs(ctx)
	require.NoError(t, err)
	require.Empty(t, cfgs)
	require.Empty(t, health)

	actual, health, err := p.GetScrapeConfig(ctx, "no_such_config")
	assert.Nil(t, actual)
	assert.Nil(t, health)
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "no_such_config" not found`), err)

	defer func() {
		err = p.DeleteScrapeConfig(ctx, "ScrapeConfigs")
		require.NoError(t, err)

		err = p.DeleteScrapeConfig(ctx, "ScrapeConfigs")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "ScrapeConfigs" not found`), err)
	}()

	cfg := &ScrapeConfig{
		JobName:        "ScrapeConfigs",
		ScrapeInterval: "1s",
		StaticConfigs: []StaticConfig{
			{[]string{"127.0.0.1:12345"}, []LabelPair{{"instance", "test_instance"}}},
			{[]string{"127.0.0.2:12345", "127.0.0.3:12345"}, nil},
		},
		RelabelConfigs: []RelabelConfig{{
			TargetLabel: "job",
			Replacement: "ScrapeConfigs_relabeled",
		}},
	}
	err = p.CreateScrapeConfig(ctx, cfg, false)
	require.NoError(t, err)

	err = p.CreateScrapeConfig(ctx, cfg, false)
	tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `scrape config with job name "ScrapeConfigs" already exist`), err)

	err = p.CreateScrapeConfig(ctx, &ScrapeConfig{JobName: "prometheus"}, false)
	tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, `scrape config with job name "prometheus" is built-in`), err)

	// other fields are filled by global values or defaults
	actual, health, err = p.GetScrapeConfig(ctx, "ScrapeConfigs")
	require.NoError(t, err)
	expected := &ScrapeConfig{
		JobName:        "ScrapeConfigs",
		ScrapeInterval: "1s",
		ScrapeTimeout:  "1s",
		MetricsPath:    "/metrics",
		Scheme:         "http",
		StaticConfigs: []StaticConfig{
			{[]string{"127.0.0.1:12345"}, []LabelPair{{"instance", "test_instance"}}},
			{[]string{"127.0.0.2:12345", "127.0.0.3:12345"}, nil},
		},
		RelabelConfigs: []RelabelConfig{{
			TargetLabel: "job",
			Replacement: "ScrapeConfigs_relabeled",
		}},
	}
	assert.Equal(t, expected, actual)
	expectedHealth := []ScrapeTargetHealth{
		{JobName: "ScrapeConfigs", Job: "ScrapeConfigs_relabeled", Target: "127.0.0.2:12345", Instance: "127.0.0.2:12345", Health: HealthUnknown},
		{JobName: "ScrapeConfigs", Job: "ScrapeConfigs_relabeled", Target: "127.0.0.3:12345", Instance: "127.0.0.3:12345", Health: HealthUnknown},
		{JobName: "ScrapeConfigs", Job: "ScrapeConfigs_relabeled", Target: "127.0.0.1:12345", Instance: "test_instance", Health: HealthUnknown},
	}
	assert.Equal(t, expectedHealth, health)

	// wait for Prometheus to scrape targets
	time.Sleep(6 * time.Second)

	_, health, err = p.GetScrapeConfig(ctx, "ScrapeConfigs")
	require.NoError(t, err)
	expectedHealth = []ScrapeTargetHealth{
		{JobName: "ScrapeConfigs", Job: "ScrapeConfigs_relabeled", Target: "127.0.0.2:12345", Instance: "127.0.0.2:12345", Health: HealthDown},
		{JobName: "ScrapeConfigs", Job: "ScrapeConfigs_relabeled", Target: "127.0.0.3:12345", Instance: "127.0.0.3:12345", Health: HealthDown},
		{JobName: "ScrapeConfigs", Job: "ScrapeConfigs_relabeled", Target: "127.0.0.1:12345", Instance: "test_instance", Health: HealthDown},
	}
	assert.Equal(t, expectedHealth, health)

	cfg.ScrapeInterval = "2s"
	err = p.UpdateScrapeConfig(ctx, cfg, false)
	require.NoError(t, err)

	actual, _, err = p.GetScrapeConfig(ctx, "ScrapeConfigs")
	require.NoError(t, err)
	expected.ScrapeInterval = "2s"
	expected.ScrapeTimeout = "2s" // config.Config.UnmarshalYAML sets timeout to interval if default timeout > interval
	assert.Equal(t, expected, actual)
}

func TestPrometheusScrapeConfigsReachability(t *testing.T) {
	ctx, p, before := SetupTest(t)
	defer TearDownTest(t, p, before)

	cfg := &ScrapeConfig{
		JobName:        "ScrapeConfigsReachability",
		ScrapeInterval: "1s",
		StaticConfigs: []StaticConfig{
			{[]string{"127.0.0.1:12345"}, nil},
		},
	}
	err := p.CreateScrapeConfig(ctx, cfg, true)
	tests.AssertGRPCErrorRE(t, codes.FailedPrecondition, `127.0.0.1:12345: Get http://127.0.0.1:12345/metrics: dial tcp 127.0.0.1:12345: \w+: connection refused`, err)

	actual, health, err := p.GetScrapeConfig(ctx, "ScrapeConfigsReachability")
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "ScrapeConfigsReachability" not found`), err)
	assert.Nil(t, actual)
	assert.Nil(t, health)
}

// https://jira.percona.com/browse/PMM-1310?focusedCommentId=196688
func TestPrometheusBadScrapeConfig(t *testing.T) {
	ctx, p, before := SetupTest(t)
	defer TearDownTest(t, p, before)

	// https://jira.percona.com/browse/PMM-1636
	cfg := &ScrapeConfig{
		JobName: "10.10.11.50:9187",
	}
	err := p.CreateScrapeConfig(ctx, cfg, false)
	tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `job_name: invalid format. Job name must be 2 to 60 characters long, characters long, contain only letters, numbers, and symbols '-', '_', and start with a letter.`), err)

	cfg = &ScrapeConfig{
		JobName:        "BadScrapeConfig",
		ScrapeInterval: "1s",
		ScrapeTimeout:  "5s",
	}
	err = p.CreateScrapeConfig(ctx, cfg, false)
	tests.AssertGRPCError(t, status.New(codes.Aborted, `scrape timeout greater than scrape interval for scrape config with job name "BadScrapeConfig"`), err)

	cfgs, statuses, err := p.ListScrapeConfigs(ctx)
	require.NoError(t, err)
	assert.Empty(t, cfgs)
	assert.Empty(t, statuses)

	actual, statuses, err := p.GetScrapeConfig(ctx, "BadScrapeConfig")
	assert.Nil(t, actual)
	assert.Nil(t, statuses)
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "BadScrapeConfig" not found`), err)

	err = p.DeleteScrapeConfig(ctx, "BadScrapeConfig")
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "BadScrapeConfig" not found`), err)
}

// https://jira.percona.com/browse/PMM-1310?focusedCommentId=196689
func TestPrometheusReadDefaults(t *testing.T) {
	ctx, p, before := SetupTest(t)
	defer TearDownTest(t, p, before)

	cfg := &ScrapeConfig{
		JobName: "ReadDefaults",
	}
	err := p.CreateScrapeConfig(ctx, cfg, false)
	assert.NoError(t, err)

	expected := &ScrapeConfig{
		JobName:        "ReadDefaults",
		ScrapeInterval: "1m",
		ScrapeTimeout:  "10s",
		MetricsPath:    "/metrics",
		Scheme:         "http",
	}

	cfgs, statuses, err := p.ListScrapeConfigs(ctx)
	require.NoError(t, err)
	assert.Equal(t, []ScrapeConfig{*expected}, cfgs)
	assert.Empty(t, statuses)

	actual, statuses, err := p.GetScrapeConfig(ctx, "ReadDefaults")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
	assert.Nil(t, statuses)
}
