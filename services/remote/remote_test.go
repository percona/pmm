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

// Package rds contains business logic of working with AWS RDS.
package remote

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/mocks"
	"github.com/percona/pmm-managed/services/postgresql"
	"github.com/percona/pmm-managed/services/prometheus"
	"github.com/percona/pmm-managed/utils/ports"
	"github.com/percona/pmm-managed/utils/tests"
)

func setup(t *testing.T) (context.Context, *postgresql.Service, *Service, *sql.DB, []byte, string, *mocks.Supervisor) {
	// We can't/shouldn't use /usr/local/percona/ (the default basedir), so use
	// a tmpdir instead with roughly the same, fake structure.
	rootDir, err := ioutil.TempDir("/tmp", "pmm-managed-test-rootdir-")
	assert.Nil(t, err)

	postgresExporterPath, err := exec.LookPath("postgres_exporter")
	require.NoError(t, err)

	ctx, p, before := prometheus.SetupTest(t)

	sqlDB := tests.OpenTestDB(t)
	db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf))
	portsRegistry := ports.NewRegistry(30000, 30999, nil)

	supervisor := &mocks.Supervisor{}
	postgreSQLService, err := postgresql.NewService(&postgresql.ServiceConfig{
		PostgresExporterPath: postgresExporterPath,
		Supervisor:           supervisor,

		DB:            db,
		Prometheus:    p,
		PortsRegistry: portsRegistry,
	})
	require.NoError(t, err)

	svc, err := NewService(&ServiceConfig{
		DB: db,
	})
	require.NoError(t, err)

	return ctx, postgreSQLService, svc, sqlDB, before, rootDir, supervisor
}

func teardown(t *testing.T, postgreSQLService *postgresql.Service, sqlDB *sql.DB, before []byte, rootDir string, supervisor *mocks.Supervisor) {
	prometheus.TearDownTest(t, postgreSQLService.Prometheus, before)

	err := sqlDB.Close()
	require.NoError(t, err)
	if rootDir != "" {
		err := os.RemoveAll(rootDir)
		assert.Nil(t, err)
	}
	supervisor.AssertExpectations(t)
}

func TestList(t *testing.T) {
	ctx, postgreSQLService, svc, sqlDB, before, rootDir, supervisor := setup(t)
	defer teardown(t, postgreSQLService, sqlDB, before, rootDir, supervisor)

	actual, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, actual)

	supervisor.On("Start", mock.Anything, mock.Anything).Return(nil)
	supervisor.On("Stop", mock.Anything, mock.Anything).Return(nil)
	id, err := postgreSQLService.Add(ctx, "", "localhost", 5432, "username", "password")
	assert.NoError(t, err)

	actual, err = svc.List(ctx)
	require.NoError(t, err)
	expected := []Instance{{
		Node: models.RemoteNode{
			ID:   2,
			Type: "remote",
			Name: "localhost:5432",
		},
		Service: models.RemoteService{
			ID:            1000,
			Type:          "postgresql",
			NodeID:        2,
			Address:       pointer.ToString("localhost"),
			Port:          pointer.ToUint16(5432),
			Engine:        pointer.ToString("PostgreSQL"),
			EngineVersion: pointer.ToString("10.5"),
		},
	}}
	assert.Equal(t, expected, actual)

	supervisor.On("Stop", mock.Anything, mock.Anything).Return(nil)
	err = postgreSQLService.Remove(ctx, id)
	assert.NoError(t, err)

	actual, err = svc.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, actual)
}
