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

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/proto"
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
	"github.com/percona/pmm-managed/services/qan"
	"github.com/percona/pmm-managed/utils/ports"
	"github.com/percona/pmm-managed/utils/tests"
)

func setup(t *testing.T) (context.Context, *Service, *sql.DB, []byte, string, *mocks.Supervisor, *httptest.Server) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/instances/42":
			switch r.Method {
			case "GET":
				var in *proto.Instance
				w.WriteHeader(http.StatusOK)
				t := r.URL.Query().Get("type")
				switch t {
				case "agent":
					in = &proto.Instance{
						Subsystem:  "agent",
						UUID:       "42",
						ParentUUID: "17",
					}
				case "mysql":
					in = &proto.Instance{
						Subsystem:  "mysql",
						UUID:       "13",
						ParentUUID: "17",
					}
				}
				data, _ := json.Marshal(in)
				w.Write(data)
			default:
				w.WriteHeader(600)
			}
		case "/instances":
			switch r.Method {
			case "POST":
				w.Header().Set("Location", "13")
				w.WriteHeader(http.StatusCreated)
			default:
				w.WriteHeader(600)
			}
		case "/instances/13":
			switch r.Method {
			case "DELETE":
				w.WriteHeader(http.StatusNoContent)
			default:
				w.WriteHeader(600)
			}
		case "/agents/42/cmd":
			switch r.Method {
			case "PUT":
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(600)
			}
		default:
			panic("unsupported path: " + r.URL.Path)
		}
	}))

	require.NoError(t, os.Setenv("PMM_QAN_API_URL", ts.URL))

	// We can't/shouldn't use /usr/local/percona/ (the default basedir), so use
	// a tmpdir instead with roughly the same, fake structure.
	rootDir, err := ioutil.TempDir("/tmp", "pmm-managed-test-rootdir-")
	assert.Nil(t, err)

	mySQLdExporterPath, err := exec.LookPath("mysqld_exporter")
	require.NoError(t, err)
	createFakeBin(t, filepath.Join(rootDir, "bin/percona-qan-agent"))
	createFakeBin(t, filepath.Join(rootDir, "bin/percona-qan-agent-installer"))
	os.MkdirAll(filepath.Join(rootDir, "config"), 0777)
	os.MkdirAll(filepath.Join(rootDir, "instance"), 0777)
	err = ioutil.WriteFile(filepath.Join(rootDir, "config/agent.conf"), []byte(`{"UUID":"42","ApiHostname":"somehostname","ApiPath":"/qan-api","ServerUser":"pmm"}`), 0666)
	require.Nil(t, err)
	err = ioutil.WriteFile(filepath.Join(rootDir, "instance/13.json"), []byte(`{"UUID":"13"}`), 0666)
	require.Nil(t, err)

	ctx, p, before := prometheus.SetupTest(t)

	sqlDB := tests.OpenTestDB(t)
	db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf))
	portsRegistry := ports.NewRegistry(30000, 30999, nil)

	supervisor := &mocks.Supervisor{}
	qan, err := qan.NewService(ctx, rootDir, supervisor)
	require.NoError(t, err)
	svc, err := NewService(&ServiceConfig{
		MySQLdExporterPath: mySQLdExporterPath,
		QAN:                qan,
		Supervisor:         supervisor,

		DB:            db,
		Prometheus:    p,
		PortsRegistry: portsRegistry,
	})
	require.NoError(t, err)
	return ctx, svc, sqlDB, before, rootDir, supervisor, ts
}

func teardown(t *testing.T, svc *Service, sqlDB *sql.DB, before []byte, rootDir string, supervisor *mocks.Supervisor, ts *httptest.Server) {
	prometheus.TearDownTest(t, svc.Prometheus, before)

	require.NoError(t, os.Unsetenv("PMM_QAN_API_URL"))

	err := sqlDB.Close()
	require.NoError(t, err)
	if rootDir != "" {
		err := os.RemoveAll(rootDir)
		assert.Nil(t, err)
	}
	ts.Close()
	supervisor.AssertExpectations(t)
}

func TestAddListRemove(t *testing.T) {
	ctx, svc, sqlDB, before, rootDir, supervisor, ts := setup(t)
	defer teardown(t, svc, sqlDB, before, rootDir, supervisor, ts)

	actual, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, actual)

	_, err = svc.Add(ctx, "", "", 0, "username", "password")
	tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `MySQL instance host is not given.`), err)

	_, err = svc.Add(ctx, "", " ", 0, "username", "password")
	tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `MySQL instance host is not given.`), err)

	supervisor.On("Start", mock.Anything, mock.Anything).Return(nil)
	supervisor.On("Status", mock.Anything, mock.Anything).Return(fmt.Errorf("not running"))
	supervisor.On("Stop", mock.Anything, mock.Anything).Return(nil)
	id, err := svc.Add(ctx, "", "localhost", 0, "pmm-managed", "pmm-managed")
	assert.NoError(t, err)

	_, err = svc.Add(ctx, "", "localhost", 3306, "pmm-managed", "pmm-managed")
	tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `MySQL instance "localhost" already exists.`), err)

	actual, err = svc.List(ctx)
	require.NoError(t, err)
	expected := []Instance{{
		Node: models.RemoteNode{
			ID:     2,
			Type:   "remote",
			Name:   "localhost",
			Region: "remote",
		},
		Service: models.MySQLService{
			ID:            1000,
			Type:          "mysql",
			NodeID:        2,
			Address:       pointer.ToString("localhost"),
			Port:          pointer.ToUint16(3306),
			Engine:        pointer.ToString("Percona Server"),
			EngineVersion: pointer.ToString("5.7.23"),
		},
	}}
	assert.Equal(t, expected, actual)

	supervisor.On("Stop", mock.Anything, mock.Anything).Return(nil)
	err = svc.Remove(ctx, id)
	assert.NoError(t, err)

	err = svc.Remove(ctx, id)
	tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf(`MySQL instance with ID %d not found.`, id)), err)

	actual, err = svc.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, actual)
}

func TestRestore(t *testing.T) {
	ctx, svc, sqlDB, before, rootDir, supervisor, ts := setup(t)
	defer teardown(t, svc, sqlDB, before, rootDir, supervisor, ts)

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
