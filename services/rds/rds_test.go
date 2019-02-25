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

package rds

/*
import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
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

func setup(t *testing.T) (context.Context, *Service, *sql.DB, []byte, string, *mocks.Supervisor) {
	uuid.SetRand(new(tests.IDReader))

	// We can't/shouldn't use /usr/local/percona/ (the default basedir), so use
	// a tmpdir instead with roughly the same, fake structure.
	rootDir, err := ioutil.TempDir("/tmp", "pmm-managed-test-rootdir-")
	assert.Nil(t, err)

	mySQLdExporterPath, err := exec.LookPath("mysqld_exporter")
	require.NoError(t, err)
	rdsExporterPath, err := exec.LookPath("rds_exporter")
	require.NoError(t, err)
	rdsExporterConfigPath := filepath.Join(rootDir, "etc/percona-rds-exporter.yml")
	os.MkdirAll(filepath.Join(rootDir, "etc"), 0777)
	err = ioutil.WriteFile(rdsExporterConfigPath, []byte(`---`), 0666)
	require.Nil(t, err)
	ctx, p, before := prometheus.SetupTest(t)

	sqlDB := tests.OpenTestDB(t)
	db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf))
	portsRegistry := ports.NewRegistry(30000, 30999, nil)

	supervisor := &mocks.Supervisor{}
	svc, err := NewService(&ServiceConfig{
		MySQLdExporterPath:    mySQLdExporterPath,
		RDSExporterPath:       rdsExporterPath,
		RDSExporterConfigPath: rdsExporterConfigPath,
		Supervisor:            supervisor,

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

func TestDiscover(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		accessKey, secretKey := tests.GetAWSKeys(t)
		ctx, svc, sqlDB, before, rootDir, supervisor := setup(t)
		defer teardown(t, svc, sqlDB, before, rootDir, supervisor)

		actual, err := svc.Discover(ctx, accessKey, secretKey)
		require.NoError(t, err)
		expected := []Instance{{
			Node: models.AWSRDSNode{
				Type:   models.AmazonRDSRemoteNodeType,
				Name:   "rds-aurora1",
				Region: pointer.ToString("us-east-1"),
			},
			Service: models.AWSRDSService{
				Type:          models.AWSRDSServiceType,
				Address:       pointer.ToString("rds-aurora1.cg8slbmxcsve.us-east-1.rds.amazonaws.com"),
				Port:          pointer.ToUint16(3306),
				Engine:        pointer.ToString("aurora"),
				EngineVersion: pointer.ToString("5.6.10a"),
			},
		}, {
			Node: models.AWSRDSNode{
				Type:   models.AmazonRDSRemoteNodeType,
				Name:   "rds-aurora57",
				Region: pointer.ToString("us-east-1"),
			},
			Service: models.AWSRDSService{
				Type:          models.AWSRDSServiceType,
				Address:       pointer.ToString("rds-aurora57.cg8slbmxcsve.us-east-1.rds.amazonaws.com"),
				Port:          pointer.ToUint16(3306),
				Engine:        pointer.ToString("aurora-mysql"),
				EngineVersion: pointer.ToString("5.7.12"),
			},
		}, {
			Node: models.AWSRDSNode{
				Type:   models.AmazonRDSRemoteNodeType,
				Name:   "rds-mysql56",
				Region: pointer.ToString("us-east-1"),
			},
			Service: models.AWSRDSService{
				Type:          models.AWSRDSServiceType,
				Address:       pointer.ToString("rds-mysql56.cg8slbmxcsve.us-east-1.rds.amazonaws.com"),
				Port:          pointer.ToUint16(3306),
				Engine:        pointer.ToString("mysql"),
				EngineVersion: pointer.ToString("5.6.37"),
			},
		}, {
			Node: models.AWSRDSNode{
				Type:   models.AmazonRDSRemoteNodeType,
				Name:   "rds-mysql57",
				Region: pointer.ToString("us-east-1"),
			},
			Service: models.AWSRDSService{
				Type:          models.AWSRDSServiceType,
				Address:       pointer.ToString("rds-mysql57.cg8slbmxcsve.us-east-1.rds.amazonaws.com"),
				Port:          pointer.ToUint16(3306),
				Engine:        pointer.ToString("mysql"),
				EngineVersion: pointer.ToString("5.7.19"),
			},
		}}

		// If this test fails, see https://jira.percona.com/browse/PMM-1772 and linked issues.
		assert.Equal(t, expected, actual)
	})

	t.Run("WrongKeys", func(t *testing.T) {
		accessKey, secretKey := "AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
		ctx, svc, sqlDB, before, rootDir, supervisor := setup(t)
		defer teardown(t, svc, sqlDB, before, rootDir, supervisor)

		res, err := svc.Discover(ctx, accessKey, secretKey)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `The security token included in the request is invalid.`), err)
		assert.Empty(t, res)
	})
}

func TestAddListRemove(t *testing.T) {
	accessKey, secretKey := tests.GetAWSKeys(t)
	ctx, svc, sqlDB, before, rootDir, supervisor := setup(t)
	defer teardown(t, svc, sqlDB, before, rootDir, supervisor)

	actual, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, actual)

	err = svc.Add(ctx, accessKey, secretKey, &InstanceID{}, "username", "password")
	tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `RDS instance name is not given.`), err)

	err = svc.Add(ctx, accessKey, secretKey, &InstanceID{"us-east-1", "rds-mysql57"}, "wrong-username", "wrong-password")
	tests.AssertGRPCErrorRE(t, codes.Unauthenticated, `Access denied for user 'wrong\-username'@'.+' \(using password: YES\)`, err)

	username, password := os.Getenv("AWS_RDS_USERNAME"), os.Getenv("AWS_RDS_PASSWORD")
	supervisor.On("Start", mock.Anything, mock.Anything).Return(nil)
	supervisor.On("Stop", mock.Anything, mock.Anything).Return(nil)
	err = svc.Add(ctx, accessKey, secretKey, &InstanceID{"us-east-1", "rds-mysql57"}, username, password)
	assert.NoError(t, err)

	err = svc.Add(ctx, accessKey, secretKey, &InstanceID{"us-east-1", "rds-mysql57"}, username, password)
	tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `RDS instance "rds-mysql57" already exists in region "us-east-1".`), err)

	actual, err = svc.List(ctx)
	require.NoError(t, err)
	expected := []Instance{{
		Node: models.AWSRDSNode{
			ID:     "gen:00000000-0000-4000-8000-000000000004",
			Type:   models.AmazonRDSRemoteNodeType,
			Name:   "rds-mysql57",
			Region: pointer.ToString("us-east-1"),
		},
		Service: models.AWSRDSService{
			ID:            "gen:00000000-0000-4000-8000-000000000005",
			Type:          models.AWSRDSServiceType,
			NodeID:        "gen:00000000-0000-4000-8000-000000000004",
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

	supervisor.On("Stop", mock.Anything, mock.Anything).Return(nil)
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
	supervisor.On("Status", mock.Anything, mock.Anything).Return(fmt.Errorf("not running"))
	supervisor.On("Stop", mock.Anything, mock.Anything).Return(nil)
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
*/
