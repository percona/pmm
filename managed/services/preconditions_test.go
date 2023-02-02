// Copyright (C) 2022 Percona LLC
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

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestCheckMongoDBBackupPreconditions(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
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
