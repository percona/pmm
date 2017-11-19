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

package tests

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/services/consul"
	"github.com/percona/pmm-managed/services/prometheus"
	"github.com/percona/pmm-managed/utils/logger"
)

const prometheusTestdata = "../../testdata/prometheus/"

func SetupPrometheusTest(t *testing.T) (p *prometheus.Service, ctx context.Context, before []byte) {
	ctx, _ = logger.Set(context.Background(), t.Name())

	consulClient, err := consul.NewClient("127.0.0.1:8500")
	require.NoError(t, err)
	require.NoError(t, consulClient.DeleteKV(prometheus.ConsulKey))

	p, err = prometheus.NewService(filepath.Join(prometheusTestdata, "prometheus.yml"), "http://127.0.0.1:9090/", "promtool", consulClient)
	require.NoError(t, err)
	require.NoError(t, p.Check(ctx))

	before, err = ioutil.ReadFile(p.ConfigPath)
	require.NoError(t, err)

	return p, ctx, before
}

func TearDownPrometheusTest(t *testing.T, p *prometheus.Service, before []byte) {
	assert.NoError(t, ioutil.WriteFile(p.ConfigPath, before, 0666))
}
