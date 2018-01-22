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
	"path/filepath"
	"testing"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/services/consul"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/tests"
)

const testdata = "../../testdata/prometheus/"

// TODO merge with promtest.Setup without pulling "testing" package dependency and flags into binary
func setup(t *testing.T) (p *Service, ctx context.Context, before []byte) {
	ctx, _ = logger.Set(context.Background(), t.Name())

	consulClient, err := consul.NewClient("127.0.0.1:8500")
	require.NoError(t, err)
	require.NoError(t, consulClient.DeleteKV(ConsulKey))

	p, err = NewService(filepath.Join(testdata, "prometheus.yml"), "http://127.0.0.1:9090/", "promtool", consulClient)
	require.NoError(t, err)
	require.NoError(t, p.Check(ctx))

	before, err = ioutil.ReadFile(p.ConfigPath)
	require.NoError(t, err)

	return p, ctx, before
}

func teardown(t *testing.T, p *Service, before []byte) {
	assert.NoError(t, ioutil.WriteFile(p.ConfigPath, before, 0666))
}

func TestPrometheusConfig(t *testing.T) {
	p, ctx, before := setup(t)
	defer teardown(t, p, before)

	// check that we can write it exactly as it was
	c, err := p.loadConfig()
	assert.NoError(t, err)
	assert.NoError(t, p.saveConfigAndReload(ctx, c))
	after, err := ioutil.ReadFile(p.ConfigPath)
	require.NoError(t, err)
	b, a := string(before), string(after)
	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A: difflib.SplitLines(b),
		B: difflib.SplitLines(a),
	})
	require.Equal(t, b, a, "%s", diff)
	require.Len(t, c.ScrapeConfigs, 4)

	// specifically check that we can read secrets
	assert.Equal(t, "pmm", c.ScrapeConfigs[1].HTTPClientConfig.BasicAuth.Password)

	// check that invalid configuration is reverted
	c.ScrapeConfigs[0].ScrapeInterval = model.Duration(time.Second)
	err = p.saveConfigAndReload(ctx, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `scrape timeout greater than scrape interval`)
	after, err = ioutil.ReadFile(p.ConfigPath)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestPrometheusScrapeConfigs(t *testing.T) {
	p, ctx, before := setup(t)
	defer teardown(t, p, before)

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
	err = p.CreateScrapeConfig(ctx, cfg)
	require.NoError(t, err)

	err = p.CreateScrapeConfig(ctx, cfg)
	tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `scrape config with job name "ScrapeConfigs" already exist`), err)

	err = p.CreateScrapeConfig(ctx, &ScrapeConfig{JobName: "prometheus"})
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
	time.Sleep(time.Second)

	_, health, err = p.GetScrapeConfig(ctx, "ScrapeConfigs")
	require.NoError(t, err)
	expectedHealth = []ScrapeTargetHealth{
		{JobName: "ScrapeConfigs", Job: "ScrapeConfigs_relabeled", Target: "127.0.0.2:12345", Instance: "127.0.0.2:12345", Health: HealthDown},
		{JobName: "ScrapeConfigs", Job: "ScrapeConfigs_relabeled", Target: "127.0.0.3:12345", Instance: "127.0.0.3:12345", Health: HealthDown},
		{JobName: "ScrapeConfigs", Job: "ScrapeConfigs_relabeled", Target: "127.0.0.1:12345", Instance: "test_instance", Health: HealthDown},
	}
	assert.Equal(t, expectedHealth, health)
}

func TestPrometheusStaticTargets(t *testing.T) {
	p, ctx, before := setup(t)
	defer teardown(t, p, before)

	cfg := &ScrapeConfig{
		JobName:        "StaticTargets",
		ScrapeInterval: "1s",
		ScrapeTimeout:  "1s",
		MetricsPath:    "/external",
		Scheme:         "http",
		StaticConfigs: []StaticConfig{
			{[]string{"127.0.1.1:12345"}, nil},
		},
	}
	err := p.CreateScrapeConfig(ctx, cfg)
	require.NoError(t, err)

	// add the same targets twice: no error, no duplicate
	for i := 0; i < 2; i++ {
		err = p.AddStaticTargets(ctx, "StaticTargets", []string{"127.0.1.2:12345", "127.0.1.2:12345"})
		require.NoError(t, err)
	}

	actual, statuses, err := p.GetScrapeConfig(ctx, "StaticTargets")
	require.NoError(t, err)
	cfg.StaticConfigs = []StaticConfig{
		{[]string{"127.0.1.1:12345", "127.0.1.2:12345"}, nil},
	}
	assert.Equal(t, cfg, actual)
	expectedHealth := []ScrapeTargetHealth{
		{JobName: "StaticTargets", Job: "StaticTargets", Target: "127.0.1.1:12345", Instance: "127.0.1.1:12345", Health: HealthDown},
		{JobName: "StaticTargets", Job: "StaticTargets", Target: "127.0.1.2:12345", Instance: "127.0.1.2:12345", Health: HealthDown},
	}
	assert.Equal(t, expectedHealth, statuses)

	err = p.AddStaticTargets(ctx, "no_such_config", []string{"127.0.1.2:12345", "127.0.1.2:12345"})
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "no_such_config" not found`), err)

	err = p.AddStaticTargets(ctx, "prometheus", []string{"127.0.1.2:12345", "127.0.1.2:12345"})
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "prometheus" not found`), err)

	// remove the same target twice: no error
	for i := 0; i < 2; i++ {
		err = p.RemoveStaticTargets(ctx, "StaticTargets", []string{"127.0.1.1:12345"})
		require.NoError(t, err)
	}

	actual, statuses, err = p.GetScrapeConfig(ctx, "StaticTargets")
	require.NoError(t, err)
	cfg.StaticConfigs = []StaticConfig{
		{[]string{"127.0.1.2:12345"}, nil},
	}
	assert.Equal(t, cfg, actual)
	expectedHealth = []ScrapeTargetHealth{
		{JobName: "StaticTargets", Job: "StaticTargets", Target: "127.0.1.2:12345", Instance: "127.0.1.2:12345", Health: HealthDown},
	}
	assert.Equal(t, expectedHealth, statuses)

	err = p.RemoveStaticTargets(ctx, "StaticTargets", []string{"127.0.1.2:12345"})
	require.NoError(t, err)

	actual, statuses, err = p.GetScrapeConfig(ctx, "StaticTargets")
	require.NoError(t, err)
	cfg.StaticConfigs = nil
	assert.Equal(t, cfg, actual)
	assert.Nil(t, statuses)

	err = p.RemoveStaticTargets(ctx, "no_such_config", []string{"127.0.1.2:12345", "127.0.1.2:12345"})
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "no_such_config" not found`), err)

	err = p.RemoveStaticTargets(ctx, "prometheus", []string{"127.0.1.2:12345", "127.0.1.2:12345"})
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "prometheus" not found`), err)
}

// https://jira.percona.com/browse/PMM-1310?focusedCommentId=196688
func TestPrometheusBadScrapeConfig(t *testing.T) {
	p, ctx, before := setup(t)
	defer teardown(t, p, before)

	// https://jira.percona.com/browse/PMM-1636
	cfg := &ScrapeConfig{
		JobName: "10.10.11.50:9187",
	}
	err := p.CreateScrapeConfig(ctx, cfg)
	tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `job_name: invalid format. Job name must be 2 to 60 characters long, characters long, contain only letters, numbers, and symbols '-', '_', and start with a letter.`), err)

	cfg = &ScrapeConfig{
		JobName:        "BadScrapeConfig",
		ScrapeInterval: "1s",
		ScrapeTimeout:  "5s",
	}
	err = p.CreateScrapeConfig(ctx, cfg)
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
	p, ctx, before := setup(t)
	defer teardown(t, p, before)

	cfg := &ScrapeConfig{
		JobName: "ReadDefaults",
	}
	err := p.CreateScrapeConfig(ctx, cfg)
	assert.NoError(t, err)

	expected := &ScrapeConfig{
		JobName:        "ReadDefaults",
		ScrapeInterval: "30s",
		ScrapeTimeout:  "15s",
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
