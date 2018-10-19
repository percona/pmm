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
	"context"
	"io/ioutil"

	"github.com/percona/pmm-managed/services/consul"
	"github.com/percona/pmm-managed/utils/logger"
)

// We can't use *testing.T (we don't want to import "testing" package which adds flags like "-test.run"),
// and we can't use assert.TestingT (it imports "net/http/httputil" packages which add "-httptest.serve" flag),
// so we use a custom interface.
type testingT interface {
	Name() string
	Fatal(args ...interface{})
}

// SetupTest returns a Prometheus service for testing.
// Returned parameters should be passed to TearDownTest after test.
func SetupTest(t testingT) (ctx context.Context, p *Service, before []byte) {
	ctx, _ = logger.Set(context.Background(), t.Name())

	consulClient, err := consul.NewClient("127.0.0.1:8500")
	if err != nil {
		t.Fatal(err)
	}
	if err = consulClient.DeleteKV(ConsulKey); err != nil {
		t.Fatal(err)
	}

	p, err = NewService("../../testdata/prometheus/prometheus.yml", "http://127.0.0.1:9090/", "promtool", consulClient)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Check(ctx); err != nil {
		t.Fatal(err)
	}

	if before, err = ioutil.ReadFile(p.ConfigPath); err != nil {
		t.Fatal(err)
	}

	return ctx, p, before
}

// TearDownTest tears down Prometheus service after testing.
func TearDownTest(t testingT, p *Service, before []byte) {
	if err := ioutil.WriteFile(p.ConfigPath, before, 0666); err != nil {
		t.Fatal(err)
	}
}
