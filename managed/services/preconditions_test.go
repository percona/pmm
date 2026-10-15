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
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
				ServiceID:   "service1",
				Name:        "mongo1",
				ClusterName: "cluster1",
				LocationID:  "loc",
				Mode:        models.PITR,
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
				ServiceID:   "service2",
				Name:        "mongo2",
				ClusterName: "cluster2",
				LocationID:  "loc",
				Mode:        models.Snapshot,
			},
		},
		Disabled: false,
	})
	require.NoError(t, err)

	t.Run("unable to create snapshot backup for cluster with enabled PITR backup", func(t *testing.T) {
		err := db.InTransactionContext(t.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(_ *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.Snapshot, "cluster1", "", "")
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "A snapshot backup for cluster 'cluster1' can be performed only if there is no enabled PITR backup for this cluster."), err)
	})

	t.Run("unable to create second PITR backup for cluster", func(t *testing.T) {
		err := db.InTransactionContext(t.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(_ *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.PITR, "cluster1", "", "")
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "A PITR backup for the cluster 'cluster1' can be enabled only if there are no other scheduled backups for this cluster."), err)
	})

	t.Run("able to update existing PITR backup for cluster", func(t *testing.T) {
		err := db.InTransactionContext(t.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(_ *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.PITR, "cluster1", "", schedule1.ID)
		})
		require.NoError(t, err)
	})

	t.Run("unable to create second PITR backup for service", func(t *testing.T) {
		err := db.InTransactionContext(t.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(_ *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.Snapshot, "", "service1", "")
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "A snapshot backup for service 'service1' can be performed only if there are no other scheduled backups for this service."), err)
	})

	t.Run("able to update existing PITR backup for service", func(t *testing.T) {
		err := db.InTransactionContext(t.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(_ *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.PITR, "", "service1", schedule1.ID)
		})
		require.NoError(t, err)
	})

	t.Run("unable to create PITR backup for cluster with scheduled snapshot backup", func(t *testing.T) {
		err := db.InTransactionContext(t.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(_ *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.PITR, "cluster2", "", "")
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "A PITR backup for the cluster 'cluster2' can be enabled only if there are no other scheduled backups for this cluster."), err)
	})

	t.Run("able to create second snapshot backup for cluster", func(t *testing.T) {
		err := db.InTransactionContext(t.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(_ *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.Snapshot, "cluster2", "", "")
		})
		require.NoError(t, err)
	})

	t.Run("unable to create PITR backup for service with scheduled snapshot backup", func(t *testing.T) {
		err := db.InTransactionContext(t.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(_ *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.PITR, "", "service2", "")
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "A PITR backup for the service with ID 'service2' can be enabled only if there are no other scheduled backups for this service."), err)
	})

	t.Run("able to create second snapshot backup for service", func(t *testing.T) {
		err := db.InTransactionContext(t.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(_ *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.Snapshot, "", "service2", "")
		})
		require.NoError(t, err)
	})

	t.Run("incremental backups are not supported", func(t *testing.T) {
		err := db.InTransactionContext(t.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(_ *reform.TX) error {
			return CheckMongoDBBackupPreconditions(db.Querier, models.Incremental, "cluster1", "", "")
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Incremental backups unsupported for MongoDB"), err)
	})
}

func TestCheckArtifactOverlapping(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
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
		Address:     new("127.0.0.1"),
		Port:        new(uint16(60000)),
		Cluster:     "cluster1",
	})
	require.NoError(t, err)

	mongoSvc2, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mongodb2",
		NodeID:      node.NodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(60000)),
		Cluster:     "cluster1",
	})
	require.NoError(t, err)

	mongoSvc3, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mongodb3",
		NodeID:      node.NodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(60000)),
		Cluster:     "cluster2",
	})
	require.NoError(t, err)

	mysqlSvc1, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mysql1",
		NodeID:      node.NodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(60000)),
		Cluster:     "mysql_cluster_1",
	})
	require.NoError(t, err)

	mysqlSvc2, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mysql2",
		NodeID:      node.NodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(60000)),
		Cluster:     "mysql_cluster_2",
	})
	require.NoError(t, err)

	location, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
		Name: "test_location",
		FilesystemConfig: &models.FilesystemLocationConfig{
			Path: "/tmp",
		},
	})
	require.NoError(t, err)

	_, err = models.CreateScheduledTask(db.Querier, models.CreateScheduledTaskParams{
		CronExpression: "* * * * *",
		StartAt:        time.Now().Truncate(time.Second).UTC(),
		Type:           models.ScheduledMongoDBBackupTask,
		Data: &models.ScheduledTaskData{
			MongoDBBackupTask: &models.MongoBackupTaskData{
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
	require.NoError(t, err)

	err = CheckArtifactOverlapping(db.Querier, mongoSvc3.ServiceID, location.ID, folder1)
	require.ErrorIs(t, err, ErrLocationFolderPairAlreadyUsed)

	err = CheckArtifactOverlapping(db.Querier, mysqlSvc1.ServiceID, location.ID, folder1)
	require.ErrorIs(t, err, ErrLocationFolderPairAlreadyUsed)

	err = CheckArtifactOverlapping(db.Querier, mysqlSvc2.ServiceID, location.ID, folder2)
	require.NoError(t, err)

	err = CheckArtifactOverlapping(db.Querier, mongoSvc1.ServiceID, location.ID, folder2)
	require.ErrorIs(t, err, ErrLocationFolderPairAlreadyUsed)
}

// connectedFunc adapts a function to AgentConnectionChecker.
type connectedFunc func(pmmAgentID string) bool

func (f connectedFunc) IsConnected(pmmAgentID string) bool {
	return f(pmmAgentID)
}

func TestCheckNodeRemovable(t *testing.T) {
	// clientNodePrefix mimics the Nodes the PMM HA Helm chart registers for its PMM Client pods:
	// they are named "<namespace>-<pod name>".
	const clientNodePrefix = "pmm-pmm-ha-client-"

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	newNode := func(t *testing.T, name string) (*models.Node, *models.Agent) {
		t.Helper()

		node, err := models.CreateNode(db.Querier, models.ContainerNodeType, &models.CreateNodeParams{NodeName: name, Address: name})
		require.NoError(t, err)
		pmmAgent, err := models.CreatePMMAgent(db.Querier, node.NodeID, nil)
		require.NoError(t, err)

		return node, pmmAgent
	}

	clientNode, clientAgent := newNode(t, clientNodePrefix+"0")
	otherNode, _ := newNode(t, "pmm-client-0")
	prefixes := []string{"pmm-pmm-ha-pg-db-", clientNodePrefix}

	connected := connectedFunc(func(string) bool { return true })
	disconnected := connectedFunc(func(string) bool { return false })

	t.Run("a protected Node with a connected pmm-agent is rejected", func(t *testing.T) {
		expected := status.New(codes.FailedPrecondition, "Node '"+clientNode.NodeName+"' is managed by this PMM deployment "+
			"and cannot be removed while its pmm-agent is connected. Scale the deployment down to remove it.")
		asked := connectedFunc(func(pmmAgentID string) bool {
			assert.Equal(t, clientAgent.AgentID, pmmAgentID)
			return true
		})
		tests.AssertGRPCError(t, expected, CheckNodeRemovable(db.Querier, asked, clientNode, prefixes))
	})

	// What a scale-down leaves behind: nothing runs the pod any more, so nothing keeps the Node.
	t.Run("a protected Node with a disconnected pmm-agent is removable", func(t *testing.T) {
		assert.NoError(t, CheckNodeRemovable(db.Querier, disconnected, clientNode, prefixes))
	})

	t.Run("a protected Node without pmm-agent is removable", func(t *testing.T) {
		node, err := models.CreateNode(db.Querier, models.ContainerNodeType, &models.CreateNodeParams{
			NodeName: clientNodePrefix + "1",
			Address:  clientNodePrefix + "1",
		})
		require.NoError(t, err)
		assert.NoError(t, CheckNodeRemovable(db.Querier, connected, node, prefixes))
	})

	t.Run("a Node the prefixes do not name is removable", func(t *testing.T) {
		asked := connectedFunc(func(pmmAgentID string) bool {
			assert.Fail(t, "a Node which is not protected needs no connection check", pmmAgentID)
			return true
		})
		assert.NoError(t, CheckNodeRemovable(db.Querier, asked, otherNode, prefixes))
	})

	t.Run("no prefixes configured", func(t *testing.T) {
		assert.NoError(t, CheckNodeRemovable(db.Querier, connected, clientNode, nil))
	})
}

func TestCheckPMMAgentRemovable(t *testing.T) {
	const clientNodePrefix = "pmm-pmm-ha-client-"

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	node, err := models.CreateNode(db.Querier, models.ContainerNodeType, &models.CreateNodeParams{
		NodeName: clientNodePrefix + "0",
		Address:  clientNodePrefix + "0",
	})
	require.NoError(t, err)
	pmmAgent, err := models.CreatePMMAgent(db.Querier, node.NodeID, nil)
	require.NoError(t, err)
	nodeExporter, err := models.CreateNodeExporter(db.Querier, pmmAgent.AgentID, nil, true, false, nil, nil, "")
	require.NoError(t, err)

	prefixes := []string{clientNodePrefix}
	connected := connectedFunc(func(string) bool { return true })
	disconnected := connectedFunc(func(string) bool { return false })

	t.Run("the pmm-agent of a protected Node is rejected while connected", func(t *testing.T) {
		expected := status.New(codes.FailedPrecondition, "pmm-agent runs on Node '"+node.NodeName+"', which is managed by "+
			"this PMM deployment, and cannot be removed while it is connected. Scale the deployment down to remove it.")
		tests.AssertGRPCError(t, expected, CheckPMMAgentRemovable(db.Querier, connected, pmmAgent, prefixes))
	})

	t.Run("the pmm-agent of a protected Node is removable once disconnected", func(t *testing.T) {
		assert.NoError(t, CheckPMMAgentRemovable(db.Querier, disconnected, pmmAgent, prefixes))
	})

	// The exporters run by that pmm-agent can go: the pod keeps its identity without them.
	t.Run("other Agents of a protected Node are removable", func(t *testing.T) {
		assert.NoError(t, CheckPMMAgentRemovable(db.Querier, connected, nodeExporter, prefixes))
	})

	t.Run("no prefixes configured", func(t *testing.T) {
		assert.NoError(t, CheckPMMAgentRemovable(db.Querier, connected, pmmAgent, nil))
	})
}
