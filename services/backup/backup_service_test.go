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

package backup

import (
	"context"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/agents"
	"github.com/percona/pmm-managed/utils/testdb"
)

func setup(t *testing.T, q *reform.Querier, serviceName string) *models.Agent {
	t.Helper()
	node, err := models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node",
	})
	require.NoError(t, err)

	pmmAgent, err := models.CreatePMMAgent(q, node.NodeID, nil)
	require.NoError(t, err)
	require.NoError(t, q.Update(pmmAgent))

	mysql, err := models.AddNewService(q, models.MySQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: serviceName,
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(3306),
	})
	require.NoError(t, err)

	agent, err := models.CreateAgent(q, models.MySQLdExporterType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  mysql.ServiceID,
		Username:   "user",
		Password:   "password",
	})
	require.NoError(t, err)
	return agent
}

func TestBackupAndRestore(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	mockedJobsService := &mockJobsService{}
	mockedJobsService.On("StartMySQLBackupJob", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockedAgentsRegistry := &mockAgentsRegistry{}
	mockedAgentsRegistry.On("StartPBMSwitchPITRActions").Return(nil)
	mockedVersioner := &mockVersioner{}
	backupService := NewService(db, mockedJobsService, mockedAgentsRegistry, mockedVersioner)

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	agent := setup(t, db.Querier, "test-service")
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

	softwares := []agents.Software{
		&agents.Mysqld{},
		&agents.Xtrabackup{},
		&agents.Xbcloud{},
		&agents.Qpress{},
	}
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

	mockedVersioner.On("GetVersions", *agent.PMMAgentID, softwares).Return(versions1, nil).Once()
	restoreID, err := backupService.RestoreBackup(ctx, pointer.GetString(agent.ServiceID), artifactID)
	require.Errorf(t, err, "artifact %q status is not successful, status: \"pending\"", artifactID)
	assert.Empty(t, restoreID)

	// imitate successful backup
	updatedArtifact, err := models.UpdateArtifact(db.Querier, artifactID, models.UpdateArtifactParams{
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

	compatibleServices, err := backupService.FindArtifactCompatibleServices(ctx, artifactID)
	require.NoError(t, err)
	require.Len(t, compatibleServices, 1)
	require.Equal(t, pointer.GetString(agent.ServiceID), compatibleServices[0].ServiceID)

	mockedVersioner.On("GetVersions", *agent.PMMAgentID, softwares).Return(versions1, nil).Once()
	mockedJobsService.On("StartMySQLRestoreBackupJob", mock.Anything, pointer.GetString(agent.PMMAgentID),
		pointer.GetString(agent.ServiceID), mock.Anything, artifact.Name, mock.Anything).Return(nil).Once()
	restoreID, err = backupService.RestoreBackup(ctx, pointer.GetString(agent.ServiceID), artifactID)
	require.NoError(t, err)
	assert.NotEmpty(t, restoreID)

	mock.AssertExpectationsForObjects(t, mockedJobsService, mockedVersioner)
}
