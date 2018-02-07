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

// Package rds contains business logic of working with AWS RDS.
package rds

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
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
	"github.com/percona/pmm-managed/services/qan"
	"github.com/percona/pmm-managed/utils/ports"
	"github.com/percona/pmm-managed/utils/tests"
	"github.com/percona/pmm-managed/utils/tests/promtest"
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
	os.Setenv("PMM_QAN_API_URL", ts.URL)

	// We can't/shouldn't use /usr/local/percona/ (the default basedir), so use
	// a tmpdir instead with roughly the same, fake structure.
	rootDir, err := ioutil.TempDir("/tmp", "pmm-managed-test-rootdir-")
	assert.Nil(t, err)

	mySQLdExporterPath := filepath.Join(rootDir, "mysqld_exporter")
	rdsExporterPath := filepath.Join(rootDir, "rds_exporter")
	createFakeBin(t, mySQLdExporterPath)
	createFakeBin(t, rdsExporterPath)
	createFakeBin(t, filepath.Join(rootDir, "bin/percona-qan-agent"))
	createFakeBin(t, filepath.Join(rootDir, "bin/percona-qan-agent-installer"))
	os.MkdirAll(filepath.Join(rootDir, "config"), 0777)
	os.MkdirAll(filepath.Join(rootDir, "instance"), 0777)
	err = ioutil.WriteFile(filepath.Join(rootDir, "config/agent.conf"), []byte(`{"UUID":"42","ApiHostname":"somehostname","ApiPath":"/qan-api","ServerUser":"pmm"}`), 0666)
	assert.Nil(t, err)
	err = ioutil.WriteFile(filepath.Join(rootDir, "instance/13.json"), []byte(`{"UUID":"13"}`), 0666)
	assert.Nil(t, err)

	p, ctx, before := promtest.Setup(t)

	sqlDB := tests.OpenTestDB(t)
	db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf))
	portsRegistry := ports.NewRegistry(30000, 30999, nil)

	supervisor := &mocks.Supervisor{}
	qan, err := qan.NewService(ctx, rootDir, supervisor)
	require.NoError(t, err)
	svc, err := NewService(&ServiceConfig{
		MySQLdExporterPath: mySQLdExporterPath,
		RDSExporterPath:    rdsExporterPath,
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
	promtest.TearDown(t, svc.Prometheus, before)
	err := sqlDB.Close()
	require.NoError(t, err)
	if rootDir != "" {
		err := os.RemoveAll(rootDir)
		assert.Nil(t, err)
	}
	ts.Close()
	supervisor.AssertExpectations(t)
}

func TestDiscover(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		accessKey, secretKey := tests.GetAWSKeys(t)
		ctx, svc, sqlDB, before, rootDir, supervisor, ts := setup(t)
		defer teardown(t, svc, sqlDB, before, rootDir, supervisor, ts)

		actual, err := svc.Discover(ctx, accessKey, secretKey)
		require.NoError(t, err)
		expected := []Instance{{
			Node: models.RDSNode{
				Type:   "rds",
				Name:   "rds-aurora1",
				Region: "us-east-1",
			},
			Service: models.RDSService{
				Type:          "rds",
				Address:       pointer.ToString("rds-aurora1.cg8slbmxcsve.us-east-1.rds.amazonaws.com"),
				Port:          pointer.ToUint16(3306),
				Engine:        pointer.ToString("aurora"),
				EngineVersion: pointer.ToString("5.6.10a"),
			},
		}, {
			Node: models.RDSNode{
				Type:   "rds",
				Name:   "rds-aurora57",
				Region: "us-east-1",
			},
			Service: models.RDSService{
				Type:          "rds",
				Address:       pointer.ToString("rds-aurora57.cg8slbmxcsve.us-east-1.rds.amazonaws.com"),
				Port:          pointer.ToUint16(3306),
				Engine:        pointer.ToString("aurora-mysql"),
				EngineVersion: pointer.ToString("5.7.12"),
			},
		}, {
			Node: models.RDSNode{
				Type:   "rds",
				Name:   "rds-mysql56",
				Region: "us-east-1",
			},
			Service: models.RDSService{
				Type:          "rds",
				Address:       pointer.ToString("rds-mysql56.cg8slbmxcsve.us-east-1.rds.amazonaws.com"),
				Port:          pointer.ToUint16(3306),
				Engine:        pointer.ToString("mysql"),
				EngineVersion: pointer.ToString("5.6.37"),
			},
		}, {
			Node: models.RDSNode{
				Type:   "rds",
				Name:   "rds-mysql57",
				Region: "us-east-1",
			},
			Service: models.RDSService{
				Type:          "rds",
				Address:       pointer.ToString("rds-mysql57.cg8slbmxcsve.us-east-1.rds.amazonaws.com"),
				Port:          pointer.ToUint16(3306),
				Engine:        pointer.ToString("mysql"),
				EngineVersion: pointer.ToString("5.7.19"),
			},
		}}

		assert.Equal(t, expected, actual)
	})

	t.Run("WrongKeys", func(t *testing.T) {
		accessKey, secretKey := "AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
		ctx, svc, sqlDB, before, rootDir, supervisor, ts := setup(t)
		defer teardown(t, svc, sqlDB, before, rootDir, supervisor, ts)

		res, err := svc.Discover(ctx, accessKey, secretKey)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `The security token included in the request is invalid.`), err)
		assert.Empty(t, res)
	})
}

func TestAddListRemove(t *testing.T) {
	accessKey, secretKey := tests.GetAWSKeys(t)
	ctx, svc, sqlDB, before, rootDir, supervisor, ts := setup(t)
	defer teardown(t, svc, sqlDB, before, rootDir, supervisor, ts)

	actual, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, actual)

	err = svc.Add(ctx, accessKey, secretKey, &InstanceID{}, "username", "password")
	tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `RDS instance name is not given.`), err)

	err = svc.Add(ctx, accessKey, secretKey, &InstanceID{"us-east-1", "rds-mysql57"}, "wrong-username", "wrong-password")
	tests.AssertGRPCErrorRE(t, codes.Unauthenticated, `Access denied for user 'wrong\-username'@'.+' \(using password: YES\)`, err)

	username, password := os.Getenv("AWS_RDS_USERNAME"), os.Getenv("AWS_RDS_PASSWORD")
	supervisor.On("Start", mock.Anything, mock.Anything).Return(nil)
	supervisor.On("Status", mock.Anything, mock.Anything).Return(fmt.Errorf("not running"))
	supervisor.On("Stop", mock.Anything, "pmm-qan-agent-9000").Return(nil) // todo why we stop it if it was not running?
	err = svc.Add(ctx, accessKey, secretKey, &InstanceID{"us-east-1", "rds-mysql57"}, username, password)
	assert.NoError(t, err)

	err = svc.Add(ctx, accessKey, secretKey, &InstanceID{"us-east-1", "rds-mysql57"}, username, password)
	tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `RDS instance "rds-mysql57" already exists in region "us-east-1".`), err)

	actual, err = svc.List(ctx)
	require.NoError(t, err)
	expected := []Instance{{
		Node: models.RDSNode{
			ID:     3,
			Type:   "rds",
			Name:   "rds-mysql57",
			Region: "us-east-1",
		},
		Service: models.RDSService{
			ID:            1001,
			Type:          "rds",
			NodeID:        3,
			AWSAccessKey:  &accessKey,
			AWSSecretKey:  &secretKey,
			Address:       pointer.ToString("rds-mysql57.cg8slbmxcsve.us-east-1.rds.amazonaws.com"),
			Port:          pointer.ToUint16(3306),
			Engine:        pointer.ToString("mysql"),
			EngineVersion: pointer.ToString("5.7.19"),
		},
	}}
	assert.Equal(t, expected, actual)

	err = svc.Remove(ctx, &InstanceID{})
	tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `RDS instance name is not given.`), err)

	supervisor.On("Stop", mock.Anything, "pmm-mysqld_exporter-30001").Return(nil)
	err = svc.Remove(ctx, &InstanceID{"us-east-1", "rds-mysql57"})
	assert.NoError(t, err)

	err = svc.Remove(ctx, &InstanceID{"us-east-1", "rds-mysql57"})
	tests.AssertGRPCError(t, status.New(codes.NotFound, `RDS instance "rds-mysql57" not found in region "us-east-1".`), err)

	actual, err = svc.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, actual)
}

func TestRestore(t *testing.T) {
	accessKey, secretKey := tests.GetAWSKeys(t)
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

	// Add one instance.
	// todo: mock AWS service
	supervisor.On("Start", mock.Anything, mock.Anything).Return(nil)
	supervisor.On("Status", mock.Anything, mock.Anything).Return(fmt.Errorf("not running"))
	supervisor.On("Stop", mock.Anything, "pmm-qan-agent-9000").Return(nil) // todo why we stop it if it was not running?
	username, password := os.Getenv("AWS_RDS_USERNAME"), os.Getenv("AWS_RDS_PASSWORD")
	err = svc.Add(ctx, accessKey, secretKey, &InstanceID{"us-east-1", "rds-mysql57"}, username, password)
	assert.NoError(t, err)

	// Restore should succeed.
	err = svc.DB.InTransaction(func(tx *reform.TX) error {
		return svc.Restore(ctx, tx)
	})

	require.NoError(t, err)
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
