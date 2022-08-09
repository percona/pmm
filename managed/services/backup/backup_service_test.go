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

package backup

import (
	"context"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
	"github.com/percona/pmm/managed/utils/testdb"
)

func setup(t *testing.T, q *reform.Querier, serviceType models.ServiceType, serviceName string) *models.Agent {
	t.Helper()
	require.Contains(t, []models.ServiceType{models.MySQLServiceType, models.MongoDBServiceType}, serviceType)

	node, err := models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node-" + t.Name(),
	})
	require.NoError(t, err)

	pmmAgent, err := models.CreatePMMAgent(q, node.NodeID, nil)
	require.NoError(t, err)

	var service *models.Service
	service, err = models.AddNewService(q, serviceType, &models.AddDBMSServiceParams{
		ServiceName: serviceName,
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(60000),
	})
	require.NoError(t, err)

	agentType := models.MySQLdExporterType
	if serviceType == models.MongoDBServiceType {
		agentType = models.MongoDBExporterType
	}

	agent, err := models.CreateAgent(q, agentType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  service.ServiceID,
		Username:   "user",
		Password:   "password",
	})
	require.NoError(t, err)
	return agent
}

func TestPerformBackup(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	mockedJobsService := &mockJobsService{}
	mockedAgentsRegistry := &mockAgentsRegistry{}
	mockedVersioner := &mockVersioner{}
	backupService := NewService(db, mockedJobsService, mockedAgentsRegistry, mockedVersioner)

	locationRes, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
		Name:        "Test location",
		Description: "Test description",
		BackupLocationConfig: models.BackupLocationConfig{
			S3Config: &models.S3LocationConfig{
				Endpoint:     "https://s3.us-west-2.amazonaws.com/",
				AccessKey:    "access_key",
				SecretKey:    "secret_key",
				BucketName:   "example_bucket",
				BucketRegion: "us-east-2",
			},
		},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	t.Run("mysql", func(t *testing.T) {
		agent := setup(t, db.Querier, models.MySQLServiceType, "test-mysql-service")
		mockedJobsService.On("StartMySQLBackupJob", mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything).Return(nil)

		softwares := []agents.Software{
			&agents.Mysqld{},
			&agents.Xtrabackup{},
			&agents.Xbcloud{},
			&agents.Qpress{},
		}

		for _, tc := range []struct {
			versions      []agents.Version
			expectedError error
		}{
			{
				versions: []agents.Version{
					{Version: "8.0.25"},
					{Version: ""},
					{Version: ""},
					{Version: "1.1"},
				},
				expectedError: ErrXtrabackupNotInstalled,
			},
			{
				versions: []agents.Version{
					{Version: "8.0.25"},
					{Version: "8.0.24"},
					{Version: "8.0.25"},
					{Version: "1.1"},
				},
				expectedError: ErrInvalidXtrabackup,
			},
			{
				versions: []agents.Version{
					{Version: "8.0.25"},
					{Version: "8.0.24"},
					{Version: "8.0.24"},
					{Version: "1.1"},
				},
				expectedError: ErrIncompatibleXtrabackup,
			},
		} {
			t.Run(tc.expectedError.Error(), func(t *testing.T) {
				mockedVersioner.On("GetVersions", *agent.PMMAgentID, softwares).Return(tc.versions, nil).Once()
				artifactID, err := backupService.PerformBackup(ctx, PerformBackupParams{
					ServiceID:  pointer.GetString(agent.ServiceID),
					LocationID: locationRes.ID,
					Name:       "test_backup",
				})
				assert.True(t, errors.Is(err, tc.expectedError))
				assert.Empty(t, artifactID)
			})
		}

		t.Run("success", func(t *testing.T) {
			versions1 := []agents.Version{
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: "1.1"},
			}

			mockedVersioner.On("GetVersions", *agent.PMMAgentID, softwares).Return(versions1, nil).Once()
			artifactID, err := backupService.PerformBackup(ctx, PerformBackupParams{
				ServiceID:  pointer.GetString(agent.ServiceID),
				LocationID: locationRes.ID,
				Name:       "test_backup",
				DataModel:  models.PhysicalDataModel,
				Mode:       models.Snapshot,
			})
			require.NoError(t, err)

			var artifact models.Artifact
			err = db.SelectOneTo(&artifact, "WHERE id = $1", artifactID)
			require.NoError(t, err)
			assert.Equal(t, locationRes.ID, artifact.LocationID)
			assert.Equal(t, *agent.ServiceID, artifact.ServiceID)
			assert.EqualValues(t, models.MySQLServiceType, artifact.Vendor)
		})

		mock.AssertExpectationsForObjects(t, mockedJobsService, mockedVersioner, mockedAgentsRegistry)
	})

	t.Run("mongodb", func(t *testing.T) {
		agent := setup(t, db.Querier, models.MongoDBServiceType, "test-mongo-service")

		t.Run("PITR is incompatible with physical backups", func(t *testing.T) {
			mockedVersioner.On("GetVersions", *agent.PMMAgentID, mock.Anything).Return(nil, nil).Once()

			artifactID, err := backupService.PerformBackup(ctx, PerformBackupParams{
				ServiceID:  pointer.GetString(agent.ServiceID),
				LocationID: locationRes.ID,
				Name:       "test_backup",
				DataModel:  models.PhysicalDataModel,
				Mode:       models.PITR,
			})
			assert.ErrorIs(t, err, ErrIncompatibleDataModel)
			assert.Empty(t, artifactID)
		})
	})
}

func TestRestoreBackup(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	mockedJobsService := &mockJobsService{}
	mockedAgentsRegistry := &mockAgentsRegistry{}
	mockedVersioner := &mockVersioner{}
	backupService := NewService(db, mockedJobsService, mockedAgentsRegistry, mockedVersioner)

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	agent := setup(t, db.Querier, models.MySQLServiceType, "test-service")

	locationRes, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
		Name:        "Test location",
		Description: "Test description",
		BackupLocationConfig: models.BackupLocationConfig{
			S3Config: &models.S3LocationConfig{
				Endpoint:     "https://s3.us-west-2.amazonaws.com/",
				AccessKey:    "access_key",
				SecretKey:    "secret_key",
				BucketName:   "example_bucket",
				BucketRegion: "us-east-2",
			},
		},
	})
	require.NoError(t, err)

	artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
		Name:       "artifact-name",
		Vendor:     string(models.MySQLServiceType),
		DBVersion:  "8.0.26",
		LocationID: locationRes.ID,
		ServiceID:  *agent.ServiceID,
		DataModel:  models.PhysicalDataModel,
		Mode:       models.Snapshot,
		Status:     models.SuccessBackupStatus,
	})
	require.NoError(t, err)

	softwares := []agents.Software{
		&agents.Mysqld{},
		&agents.Xtrabackup{},
		&agents.Xbcloud{},
		&agents.Qpress{},
	}

	for _, tc := range []struct {
		testName      string
		versions      []agents.Version
		expectedError error
	}{
		{
			testName: "xtrabackup not installed",
			versions: []agents.Version{
				{Version: "8.0.26"},
				{Version: ""},
				{Version: ""},
				{Version: "1.1"},
			},
			expectedError: ErrXtrabackupNotInstalled,
		},
		{
			testName: "invalid xtrabackup",
			versions: []agents.Version{
				{Version: "8.0.26"},
				{Version: "8.0.25"},
				{Version: "8.0.26"},
				{Version: "1.1"},
			},
			expectedError: ErrInvalidXtrabackup,
		},
		{
			testName: "incompatible xtrabackup",
			versions: []agents.Version{
				{Version: "8.0.26"},
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: "1.1"},
			},
			expectedError: ErrIncompatibleXtrabackup,
		},
		{
			testName: "incompatible target MySQL",
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: "1.1"},
			},
			expectedError: ErrIncompatibleTargetMySQL,
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			mockedVersioner.On("GetVersions", *agent.PMMAgentID, softwares).Return(tc.versions, nil).Once()
			restoreID, err := backupService.RestoreBackup(ctx, pointer.GetString(agent.ServiceID), artifact.ID)
			assert.True(t, errors.Is(err, tc.expectedError), err)
			assert.Empty(t, restoreID)
		})
	}

	t.Run("success", func(t *testing.T) {
		versions1 := []agents.Version{
			{Version: "8.0.26"},
			{Version: "8.0.26"},
			{Version: "8.0.26"},
			{Version: "1.1"},
		}

		updatedArtifact, err := models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{
			Status: models.BackupStatusPointer(models.PendingBackupStatus),
		})
		require.NoError(t, err)
		require.NotNil(t, updatedArtifact)

		mockedVersioner.On("GetVersions", *agent.PMMAgentID, softwares).Return(versions1, nil).Once()
		restoreID, err := backupService.RestoreBackup(ctx, pointer.GetString(agent.ServiceID), artifact.ID)
		require.Errorf(t, err, "artifact %q status is not successful, status: \"pending\"", artifact.ID)
		assert.Empty(t, restoreID)

		// imitate successful backup
		updatedArtifact, err = models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{
			Status: models.BackupStatusPointer(models.SuccessBackupStatus),
		})
		require.NoError(t, err)
		require.NotNil(t, updatedArtifact)

		// imitate successful update of the service software versions
		_, err = models.UpdateServiceSoftwareVersions(db.Querier, pointer.GetString(agent.ServiceID),
			models.UpdateServiceSoftwareVersionsParams{
				SoftwareVersions: []models.SoftwareVersion{
					{
						Name:    models.MysqldSoftwareName,
						Version: versions1[0].Version,
					},
					{
						Name:    models.XtrabackupSoftwareName,
						Version: versions1[1].Version,
					},
					{
						Name:    models.XbcloudSoftwareName,
						Version: versions1[2].Version,
					},
					{
						Name:    models.QpressSoftwareName,
						Version: versions1[3].Version,
					},
				},
			})
		require.NoError(t, err)

		compatibleServices, err := backupService.FindArtifactCompatibleServices(ctx, artifact.ID)
		require.NoError(t, err)
		require.Len(t, compatibleServices, 1)
		require.Equal(t, pointer.GetString(agent.ServiceID), compatibleServices[0].ServiceID)

		mockedVersioner.On("GetVersions", *agent.PMMAgentID, softwares).Return(versions1, nil).Once()
		mockedJobsService.On("StartMySQLRestoreBackupJob", mock.Anything, pointer.GetString(agent.PMMAgentID),
			pointer.GetString(agent.ServiceID), mock.Anything, artifact.Name, mock.Anything).Return(nil).Once()
		restoreID, err = backupService.RestoreBackup(ctx, pointer.GetString(agent.ServiceID), artifact.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, restoreID)
	})

	mock.AssertExpectationsForObjects(t, mockedJobsService, mockedVersioner, mockedAgentsRegistry)
}
