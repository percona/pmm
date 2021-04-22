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
	backupv1beta1 "github.com/percona/pmm/api/managementpb/backup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
)

func setup(t *testing.T, db *reform.DB, serviceName string) *models.Agent {
	t.Helper()
	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node",
	})
	require.NoError(t, err)

	pmmAgent, err := models.CreatePMMAgent(db.Querier, node.NodeID, nil)
	require.NoError(t, err)
	require.NoError(t, db.Update(pmmAgent))

	mysql, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: serviceName,
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(3306),
	})
	require.NoError(t, err)

	agent, err := models.CreateAgent(db.Querier, models.MySQLdExporterType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  mysql.ServiceID,
		Username:   "user",
		Password:   "password",
	})
	require.NoError(t, err)
	return agent
}

func TestStartBackup(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	mockedJobsService := &mockJobsService{}
	mockedJobsService.On("StartMySQLBackupJob", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything).Return(nil)
	backupSvc := NewBackupsService(db, mockedJobsService)

	agent := setup(t, db, "test-service")
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

	backupRes, err := backupSvc.StartBackup(ctx, &backupv1beta1.StartBackupRequest{
		ServiceId:  pointer.GetString(agent.ServiceID),
		LocationId: locationRes.ID,
		Name:       "Test backup",
	})

	assert.NoError(t, err)
	var artifact models.Artifact
	err = db.SelectOneTo(&artifact, "WHERE id = $1", backupRes.ArtifactId)
	assert.NoError(t, err)
	assert.Equal(t, locationRes.ID, artifact.LocationID)
	assert.Equal(t, *agent.ServiceID, artifact.ServiceID)
	assert.EqualValues(t, models.MySQLServiceType, artifact.Vendor)
}
