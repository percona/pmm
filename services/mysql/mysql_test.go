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

package mysql

/*
import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func setup(t *testing.T) (context.Context, *Service, *sql.DB, []byte, *mocks.Supervisor) {
	uuid.SetRand(new(tests.IDReader))

	mySQLdExporterPath, err := exec.LookPath("mysqld_exporter")
	require.NoError(t, err)
	ctx, p, before := prometheus.SetupTest(t)

	sqlDB := tests.OpenTestDB(t)
	db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf))
	portsRegistry := ports.NewRegistry(30000, 30999, nil)

	supervisor := &mocks.Supervisor{}
	svc, err := NewService(&ServiceConfig{
		MySQLdExporterPath: mySQLdExporterPath,
		Supervisor:         supervisor,

		DB:            db,
		Prometheus:    p,
		PortsRegistry: portsRegistry,
	})
	require.NoError(t, err)
	return ctx, svc, sqlDB, before, supervisor
}

func teardown(t *testing.T, svc *Service, sqlDB *sql.DB, before []byte, supervisor *mocks.Supervisor) {
	prometheus.TearDownTest(t, svc.Prometheus, before)

	err := sqlDB.Close()
	require.NoError(t, err)
	supervisor.AssertExpectations(t)
}

func TestAddListRemove(t *testing.T) {
	ctx, svc, sqlDB, before, supervisor := setup(t)
	defer teardown(t, svc, sqlDB, before, supervisor)

	actual, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, actual)

	_, err = svc.Add(ctx, "", "", 0, "username", "password")
	tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `MySQL instance host is not given.`), err)

	_, err = svc.Add(ctx, "", " ", 0, "username", "password")
	tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `MySQL instance host is not given.`), err)

	supervisor.On("Start", mock.Anything, mock.Anything).Return(nil)
	supervisor.On("Stop", mock.Anything, mock.Anything).Return(nil)
	id, err := svc.Add(ctx, "", "localhost", 0, "pmm-managed", "pmm-managed")
	assert.NoError(t, err)

	_, err = svc.Add(ctx, "", "localhost", 3306, "pmm-managed", "pmm-managed")
	tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `MySQL instance "localhost" already exists.`), err)

	actual, err = svc.List(ctx)
	require.NoError(t, err)
	expected := []Instance{{
		Node: models.RemoteNode{
			ID:     "gen:00000000-0000-4000-8000-000000000001",
			Type:   "remote",
			Name:   "localhost",
			Region: pointer.ToString("remote"),
		},
		Service: models.MySQLService{
			ID:            "gen:00000000-0000-4000-8000-000000000002",
			Type:          "mysql",
			Name:          "localhost",
			NodeID:        "gen:00000000-0000-4000-8000-000000000001",
			Address:       pointer.ToString("localhost"),
			Port:          pointer.ToUint16(3306),
			Engine:        pointer.ToString("Percona Server"),
			EngineVersion: pointer.ToString("5.7.24"),
		},
	}}
	assert.Equal(t, expected, actual)

	supervisor.On("Stop", mock.Anything, mock.Anything).Return(nil)
	err = svc.Remove(ctx, id)
	assert.NoError(t, err)

	err = svc.Remove(ctx, id)
	tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf(`MySQL instance with ID %q not found.`, id)), err)

	actual, err = svc.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, actual)
}

func TestRestore(t *testing.T) {
	ctx, svc, sqlDB, before, supervisor := setup(t)
	defer teardown(t, svc, sqlDB, before, supervisor)

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
	_, err = svc.Add(ctx, "", "localhost", 3306, "pmm-managed", "pmm-managed")
	assert.NoError(t, err)

	// Restore should succeed.
	err = svc.DB.InTransaction(func(tx *reform.TX) error {
		return svc.Restore(ctx, tx)
	})
	require.NoError(t, err)
}

func TestNormalizeEngineAndEngineVersion(t *testing.T) {
	parameters := []struct {
		versionComment  string
		version         string
		expectedEngine  string
		expectedVersion string
	}{
		{version: "5.7.23-23", versionComment: "Percona Server (GPL), Release '23', Revision '500fcf5'", expectedEngine: "Percona Server", expectedVersion: "5.7.23"},
		{version: "10.3.10-MariaDB-1:10.3.10+maria~bionic", versionComment: "mariadb.org binary distribution", expectedEngine: "MariaDB", expectedVersion: "10.3.10"},
		{version: "5.7.24-0ubuntu0.18.04.1", versionComment: "(Ubuntu)", expectedEngine: "MySQL", expectedVersion: "5.7.24"},
		{version: "8.0.13", versionComment: "MySQL Community Server - GPL", expectedEngine: "MySQL", expectedVersion: "8.0.13"},
		{version: "5.6.42", versionComment: "MySQL Community Server - GPL", expectedEngine: "MySQL", expectedVersion: "5.6.42"},
	}

	for _, params := range parameters {
		engine, engineVersion, err := normalizeEngineAndEngineVersion(params.versionComment, params.version)
		assert.NoError(t, err)

		assert.Equal(t, params.expectedEngine, engine, "engine is not equal")
		assert.Equal(t, params.expectedVersion, engineVersion, "engineVersion is not equal")
	}
}

func createFakeBin(t *testing.T, name string) {
	var err error

	dir := filepath.Dir(name)
	err = os.MkdirAll(dir, 0777)
	assert.NoError(t, err)

	f, err := os.Create(name)
	assert.NoError(t, err)

	_, err = f.WriteString("#!/bin/sh\n")
	assert.NoError(t, err)

	_, err = f.WriteString("echo 'it works'")
	assert.NoError(t, err)

	err = f.Close()
	assert.NoError(t, err)

	err = os.Chmod(name, 0777)
	assert.NoError(t, err)
}
*/
