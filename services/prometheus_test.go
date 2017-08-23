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

package services

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/utils/logger"
)

var testdata = filepath.FromSlash("../testdata/prometheus/")

func TestPrometheus(t *testing.T) {
	ctx, _ := logger.Set(context.Background(), "TestPrometheus")
	p := &Prometheus{
		ConfigPath: filepath.Join(testdata, "prometheus.yml"),
		URL: &url.URL{
			Scheme: "http",
			Host:   "127.0.0.1:9090",
		},
		PromtoolPath:   "promtool",
		AlertRulesPath: filepath.Join(testdata, "alerts"),
	}
	require.NoError(t, p.Check(ctx))

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
