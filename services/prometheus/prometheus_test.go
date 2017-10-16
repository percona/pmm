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
	"time"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/services/consul"
	"github.com/percona/pmm-managed/utils/logger"
)

const testdata = "../../testdata/prometheus/"

func getPrometheus(t testing.TB, ctx context.Context) *Service {
	// t.Helper() TODO enable when we switch to 1.9+

	consulClient, err := consul.NewClient("127.0.0.1:8500")
	require.NoError(t, err)
	require.NoError(t, consulClient.DeleteKV(consulKey))

	svc, err := NewService(filepath.Join(testdata, "prometheus.yml"), "http://127.0.0.1:9090/", "promtool", consulClient)
	require.NoError(t, err)
	require.NoError(t, svc.Check(ctx))
	return svc
}

func assertGRPCError(t testing.TB, expected *status.Status, actual error) {
	// t.Helper() TODO enable when we switch to 1.9+

	s, ok := status.FromError(actual)
	if !assert.True(t, ok) {
		return
	}
	assert.Equal(t, expected.Code(), s.Code())
	assert.Equal(t, expected.Message(), s.Message())
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

func TestPrometheusScrapeConfigs(t *testing.T) {
	ctx, _ := logger.Set(context.Background(), "TestPrometheusScrapeConfigs")
	p := getPrometheus(t, ctx)

	// always restore original after test
	before, err := ioutil.ReadFile(p.configPath)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, ioutil.WriteFile(p.configPath, before, 0666))
	}()

	cfgs, err := p.ListScrapeConfigs(ctx)
	require.NoError(t, err)
	require.Empty(t, cfgs)

	actual, err := p.GetScrapeConfig(ctx, "no_such_config")
	assert.Nil(t, actual)
	assertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "no_such_config" not found`), err)

	defer func() {
		err = p.DeleteScrapeConfig(ctx, "test_config")
		require.NoError(t, err)

		err = p.DeleteScrapeConfig(ctx, "test_config")
		assertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "test_config" not found`), err)
	}()

	// other fields are filled by defaults
	scs := []StaticConfig{
		{[]string{"127.0.0.1:12345", "127.0.0.2:12345"}, nil},
	}
	cfg := &ScrapeConfig{
		JobName:       "test_config",
		StaticConfigs: scs,
	}
	err = p.CreateScrapeConfig(ctx, cfg)
	require.NoError(t, err)

	err = p.CreateScrapeConfig(ctx, cfg)
	assertGRPCError(t, status.New(codes.AlreadyExists, `scrape config with job name "test_config" already exist`), err)

	actual, err = p.GetScrapeConfig(ctx, "test_config")
	require.NoError(t, err)
	assert.Equal(t, cfg, actual)
}

func TestPrometheusStaticTargets(t *testing.T) {
	ctx, _ := logger.Set(context.Background(), "TestPrometheusStaticTargets")
	p := getPrometheus(t, ctx)

	// always restore original after test
	before, err := ioutil.ReadFile(p.configPath)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, ioutil.WriteFile(p.configPath, before, 0666))
	}()

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
	err = p.CreateScrapeConfig(ctx, cfg)
	require.NoError(t, err)

	err = p.AddStaticTargets(ctx, "test_config", []string{"127.0.0.2:12345"})
	require.NoError(t, err)

	actual, err := p.GetScrapeConfig(ctx, "test_config")
	require.NoError(t, err)
	cfg.StaticConfigs = []StaticConfig{
		{[]string{"127.0.0.1:12345", "127.0.0.2:12345"}, nil},
	}
	assert.Equal(t, cfg, actual)

	err = p.RemoveStaticTargets(ctx, "test_config", []string{"127.0.0.1:12345"})
	require.NoError(t, err)

	actual, err = p.GetScrapeConfig(ctx, "test_config")
	require.NoError(t, err)
	cfg.StaticConfigs = []StaticConfig{
		{[]string{"127.0.0.2:12345"}, nil},
	}
	assert.Equal(t, cfg, actual)
}
