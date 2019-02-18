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

package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/mocks"
	"github.com/percona/pmm-managed/services/prometheus"
	"github.com/percona/pmm-managed/utils/ports"
	"github.com/percona/pmm-managed/utils/tests"
)

func setup(t *testing.T) (context.Context, *Service, *sql.DB, []byte, string, *mocks.Supervisor) {
	uuid.SetRand(new(tests.IDReader))

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
	svc, err := NewService(&ServiceConfig{
		PostgresExporterPath: postgresExporterPath,
		Supervisor:           supervisor,

		DB:            db,
		Prometheus:    p,
		PortsRegistry: portsRegistry,
	})
	require.NoError(t, err)
	return ctx, svc, sqlDB, before, rootDir, supervisor
}

func teardown(t *testing.T, svc *Service, sqlDB *sql.DB, before []byte, rootDir string, supervisor *mocks.Supervisor) {
	prometheus.TearDownTest(t, svc.Prometheus, before)

	err := sqlDB.Close()
	require.NoError(t, err)
	if rootDir != "" {
		err := os.RemoveAll(rootDir)
		assert.Nil(t, err)
	}
	supervisor.AssertExpectations(t)
}

func TestAddListRemove(t *testing.T) {
	ctx, svc, sqlDB, before, rootDir, supervisor := setup(t)
	defer teardown(t, svc, sqlDB, before, rootDir, supervisor)

	actual, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, actual)

	_, err = svc.Add(ctx, "", "", 0, "pmm-managed", "pmm-managed")
	tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `PostgreSQL instance host is not given.`), err)

	_, err = svc.Add(ctx, "", " ", 0, "pmm-managed", "pmm-managed")
	tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `PostgreSQL instance host is not given.`), err)

	supervisor.On("Start", mock.Anything, mock.Anything).Return(nil)
	supervisor.On("Stop", mock.Anything, mock.Anything).Return(nil)
	id, err := svc.Add(ctx, "", "localhost", 0, "pmm-managed", "pmm-managed")
	assert.NoError(t, err)

	_, err = svc.Add(ctx, "", "localhost", 5432, "pmm-managed", "pmm-managed")
	tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `PostgreSQL instance "localhost" already exists.`), err)

	actual, err = svc.List(ctx)
	require.NoError(t, err)
	expected := []Instance{{
		Node: models.RemoteNode{
			ID:     "gen:00000000-0000-4000-8000-000000000001",
			Type:   "remote",
			Name:   "localhost",
			Region: pointer.ToString("remote"),
		},
		Service: models.PostgreSQLService{
			ID:            "gen:00000000-0000-4000-8000-000000000002",
			Type:          "postgresql",
			Name:          "localhost",
			NodeID:        "gen:00000000-0000-4000-8000-000000000001",
			Address:       pointer.ToString("localhost"),
			Port:          pointer.ToUint16(5432),
			Engine:        pointer.ToString("PostgreSQL"),
			EngineVersion: pointer.ToString("10.7"),
		},
	}}
	assert.Equal(t, expected, actual)

	supervisor.On("Stop", mock.Anything, mock.Anything).Return(nil)
	err = svc.Remove(ctx, id)
	assert.NoError(t, err)

	err = svc.Remove(ctx, id)
	tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf(`PostgreSQL instance with ID %q not found.`, id)), err)

	actual, err = svc.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, actual)
}

func TestRestore(t *testing.T) {
	ctx, svc, sqlDB, before, rootDir, supervisor := setup(t)
	defer teardown(t, svc, sqlDB, before, rootDir, supervisor)

	// Fill some hidden dependencies.
	actual, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, actual)

	// Restore shouldn't fail when there is nothing to restore.
	err = svc.DB.InTransaction(func(tx *reform.TX) error {
		return svc.Restore(ctx, tx)
	})
	require.NoError(t, err)

	// Add one instance.
	supervisor.On("Start", mock.Anything, mock.Anything).Return(nil)
	supervisor.On("Status", mock.Anything, mock.Anything).Return(nil)
	supervisor.On("Stop", mock.Anything, mock.Anything).Return(nil)
	_, err = svc.Add(ctx, "", "localhost", 5432, "pmm-managed", "pmm-managed")
	assert.NoError(t, err)

	// Restore should succeed.
	err = svc.DB.InTransaction(func(tx *reform.TX) error {
		return svc.Restore(ctx, tx)
	})
	require.NoError(t, err)
}

func TestExtractFromVersion(t *testing.T) {
	_, svc, sqlDB, before, rootDir, supervisor := setup(t)
	defer teardown(t, svc, sqlDB, before, rootDir, supervisor)

	version := "PostgreSQL 10.5 (Debian 10.5-1.pgdg90+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 6.3.0-18+deb9u1) 6.3.0 20170516, 64-bit"
	engine, engineVersion := svc.engineAndVersionFromPlainText(version)

	assert.Equal(t, "PostgreSQL", engine, "engine is not equal")
	assert.Equal(t, "10.5", engineVersion, "engineVersion is not equal")
}
