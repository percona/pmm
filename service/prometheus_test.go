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

package service

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Percona-Lab/pmm-managed/utils/logger"
)

func TestPrometheus(t *testing.T) {
	p := &Prometheus{
		ConfigPath: "../testdata/prometheus/prometheus.yml",
		URL: &url.URL{
			Scheme: "http",
			Host:   "127.0.0.1:9090",
		},
		AlertRulesPath: "../testdata/prometheus/alerts/",
		PromtoolPath:   "promtool",
	}
	ctx, _ := logger.Set(context.Background())

	alerts, err := p.ListAlertRules(ctx)
	if err != nil {
		t.Fatal(err)
	}
	require.Len(t, alerts, 2)
	alerts[0].Text = "" // FIXME
	alerts[1].Text = "" // FIXME
	expected := []AlertRule{
		{"InstanceDown", "../testdata/prometheus/alerts/InstanceDown.rule", "", false},
		{"Something", "../testdata/prometheus/alerts/Something.rule.disabled", "", true},
	}
	assert.Equal(t, expected, alerts)

	rule := &AlertRule{
		Name: "TestPrometheus",
		Text: "ALERT TestPrometheus IF up == 0",
	}
	require.NoError(t, p.PutAlert(ctx, rule))

	defer func() {
		require.NoError(t, p.DeleteAlert(ctx, "TestPrometheus"))
	}()
}
