// Copyright (C) 2023 Percona LLC
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

package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestCheckMongoDBBackupPreconditions(t *testing.T) {
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	schedule1, err := models.CreateScheduledTask(db.Querier, models.CreateScheduledTaskParams{
		CronExpression: "* * * * *",
		Type:           models.ScheduledMongoDBBackupTask,
		Data: &models.ScheduledTaskData{
			MongoDBBackupTask: &models.MongoBackupTaskData{
				CommonBackupTaskData: models.CommonBackupTaskData{
					ServiceID:   "service1",
					Name:        "mongo1",
					ClusterName: "cluster1",
					LocationID:  "loc",
					Mode:        models.PITR,
				},
			},
		},
		Disabled: false,
	})
	require.NoError(t, err)

	_, err = models.CreateScheduledTask(db.Querier, models.CreateScheduledTaskParams{
		CronExpression: "* * * * *",
		Type:           models.ScheduledMongoDBBackupTask,
		Data: &models.ScheduledTaskData{
			MongoDBBackupTask: &models.MongoBackupTaskData{
				CommonBackupTaskData: models.CommonBackupTaskData{
					ServiceID:   "service2",
					Name:        "mongo2",
					ClusterName: "cluster2",
					LocationID:  "loc",
					Mode:        models.Snapshot,
				},
			},
		},
		Disabled: false,
	})
	require.NoError(t, err)

	t.Run("unable to create snapshot backup for cluster with enabled PITR backup", func(t *testing.T) {
		err := db.InTransactionContext(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.Snapshot, "cluster1", "", "")
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "A snapshot backup for cluster 'cluster1' can be performed only if there is no enabled PITR backup for this cluster."), err)
	})

	t.Run("unable to create second PITR backup for cluster", func(t *testing.T) {
		err := db.InTransactionContext(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.PITR, "cluster1", "", "")
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "A PITR backup for the cluster 'cluster1' can be enabled only if there are no other scheduled backups for this cluster."), err)
	})

	t.Run("able to update existing PITR backup for cluster", func(t *testing.T) {
		err := db.InTransactionContext(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.PITR, "cluster1", "", schedule1.ID)
		})
		require.NoError(t, err)
	})

	t.Run("unable to create second PITR backup for service", func(t *testing.T) {
		err := db.InTransactionContext(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.Snapshot, "", "service1", "")
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "A snapshot backup for service 'service1' can be performed only if there are no other scheduled backups for this service."), err)
	})

	t.Run("able to update existing PITR backup for service", func(t *testing.T) {
		err := db.InTransactionContext(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.PITR, "", "service1", schedule1.ID)
		})
		require.NoError(t, err)
	})

	t.Run("unable to create PITR backup for cluster with scheduled snapshot backup", func(t *testing.T) {
		err := db.InTransactionContext(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.PITR, "cluster2", "", "")
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "A PITR backup for the cluster 'cluster2' can be enabled only if there are no other scheduled backups for this cluster."), err)
	})

	t.Run("able to create second snapshot backup for cluster", func(t *testing.T) {
		err := db.InTransactionContext(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.Snapshot, "cluster2", "", "")
		})
		require.NoError(t, err)
	})

	t.Run("unable to create PITR backup for service with scheduled snapshot backup", func(t *testing.T) {
		err := db.InTransactionContext(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.PITR, "", "service2", "")
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "A PITR backup for the service with ID 'service2' can be enabled only if there are no other scheduled backups for this service."), err)
	})

	t.Run("able to create second snapshot backup for service", func(t *testing.T) {
		err := db.InTransactionContext(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.Snapshot, "", "service2", "")
		})
		require.NoError(t, err)
	})

	t.Run("incremental backups are not supported", func(t *testing.T) {
		err := db.InTransactionContext(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.Incremental, "cluster1", "", "")
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Incremental backups unsupported for MongoDB"), err)
	})
}

func TestCheckArtifactOverlapping(t *testing.T) {
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	folder1, folder2 := "folder1", "folder2"

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node",
	})
	require.NoError(t, err)

	mongoSvc1, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mongodb1",
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(60000),
		Cluster:     "cluster1",
	})
	require.NoError(t, err)

	mongoSvc2, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mongodb2",
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(60000),
		Cluster:     "cluster1",
	})
	require.NoError(t, err)

	mongoSvc3, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mongodb3",
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(60000),
		Cluster:     "cluster2",
	})
	require.NoError(t, err)

	mysqlSvc1, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mysql1",
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(60000),
		Cluster:     "mysql_cluster_1",
	})
	require.NoError(t, err)

	mysqlSvc2, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mysql2",
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(60000),
		Cluster:     "mysql_cluster_2",
	})
	require.NoError(t, err)

	location, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
		Name: "test_location",
		BackupLocationConfig: models.BackupLocationConfig{
			FilesystemConfig: &models.FilesystemLocationConfig{
				Path: "/tmp",
			},
		},
	})
	require.NoError(t, err)

	_, err = models.CreateScheduledTask(db.Querier, models.CreateScheduledTaskParams{
		CronExpression: "* * * * *",
		StartAt:        time.Now().Truncate(time.Second).UTC(),
		Type:           models.ScheduledMongoDBBackupTask,
		Data: &models.ScheduledTaskData{
			MongoDBBackupTask: &models.MongoBackupTaskData{
				CommonBackupTaskData: models.CommonBackupTaskData{
					ServiceID:     mongoSvc1.ServiceID,
					LocationID:    location.ID,
					Name:          "test",
					Description:   "test backup task",
					DataModel:     models.LogicalDataModel,
					Mode:          models.Snapshot,
					Retention:     7,
					Retries:       3,
					RetryInterval: 5 * time.Second,
					ClusterName:   "cluster1",
					Folder:        folder1,
				},
			},
		},
	})
	require.NoError(t, err)

	_, err = models.CreateArtifact(db.Querier, models.CreateArtifactParams{
		Name:       "test_artifact",
		Vendor:     "mysql",
		LocationID: location.ID,
		ServiceID:  mysqlSvc1.ServiceID,
		DataModel:  models.LogicalDataModel,
		Mode:       models.Snapshot,
		Status:     models.SuccessBackupStatus,
		Folder:     folder2,
	})
	require.NoError(t, err)

	err = CheckArtifactOverlapping(db.Querier, mongoSvc2.ServiceID, location.ID, folder1)
	assert.NoError(t, err)

	err = CheckArtifactOverlapping(db.Querier, mongoSvc3.ServiceID, location.ID, folder1)
	assert.ErrorIs(t, err, ErrLocationFolderPairAlreadyUsed)

	err = CheckArtifactOverlapping(db.Querier, mysqlSvc1.ServiceID, location.ID, folder1)
	assert.ErrorIs(t, err, ErrLocationFolderPairAlreadyUsed)

	err = CheckArtifactOverlapping(db.Querier, mysqlSvc2.ServiceID, location.ID, folder2)
	assert.NoError(t, err)

	err = CheckArtifactOverlapping(db.Querier, mongoSvc1.ServiceID, location.ID, folder2)
	assert.ErrorIs(t, err, ErrLocationFolderPairAlreadyUsed)
}
