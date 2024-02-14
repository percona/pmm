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

package backup

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api-tests/management"
	backupClient "github.com/percona/pmm/api/backup/v1/json/client"
	backups "github.com/percona/pmm/api/backup/v1/json/client/backups_service"
	locations "github.com/percona/pmm/api/backup/v1/json/client/locations_service"
	managementClient "github.com/percona/pmm/api/management/v1/json/client"
	mongodb "github.com/percona/pmm/api/management/v1/json/client/mongo_db_service"
	node "github.com/percona/pmm/api/management/v1/json/client/node_service"
)

func TestScheduleBackup(t *testing.T) {
	t.Run("mongo", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "node-for-basic-name")
		nodeID, pmmAgentID := management.RegisterGenericNode(t, node.RegisterNodeBody{
			NodeName: nodeName,
			NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})
		defer pmmapitests.RemoveNodes(t, nodeID)
		defer management.RemovePMMAgentWithSubAgents(t, pmmAgentID)
		mongo1Name := pmmapitests.TestString(t, "mongo")
		mongo2Name := pmmapitests.TestString(t, "mongo")

		mongo1Resp, err := managementClient.Default.MongoDBService.AddMongoDB(&mongodb.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: mongodb.AddMongoDBBody{
				NodeID:      nodeID,
				Cluster:     "test_cluster",
				PMMAgentID:  pmmAgentID,
				ServiceName: mongo1Name,
				Address:     "10.10.10.10",
				Port:        27017,
				Username:    "username",

				SkipConnectionCheck: true,
				DisableCollectors:   []string{"global_status", "perf_schema.tablelocks"},
			},
		})
		require.NoError(t, err)
		mongo1ID := mongo1Resp.Payload.Service.ServiceID
		defer pmmapitests.RemoveServices(t, mongo1ID)

		mongo2Resp, err := managementClient.Default.MongoDBService.AddMongoDB(&mongodb.AddMongoDBParams{
			Context: pmmapitests.Context,
			Body: mongodb.AddMongoDBBody{
				NodeID:      nodeID,
				Cluster:     "test_cluster",
				PMMAgentID:  pmmAgentID,
				ServiceName: mongo2Name,
				Address:     "10.10.10.11",
				Port:        27017,
				Username:    "username",

				SkipConnectionCheck: true,
				DisableCollectors:   []string{"global_status", "perf_schema.tablelocks"},
			},
		})
		require.NoError(t, err)
		mongo2ID := mongo2Resp.Payload.Service.ServiceID
		defer pmmapitests.RemoveServices(t, mongo2ID)

		resp, err := backupClient.Default.LocationsService.AddLocation(&locations.AddLocationParams{
			Body: locations.AddLocationBody{
				Name:        gofakeit.Name(),
				Description: gofakeit.Question(),
				FilesystemConfig: &locations.AddLocationParamsBodyFilesystemConfig{
					Path: "/tmp",
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		locationID := resp.Payload.LocationID
		defer deleteLocation(t, backupClient.Default.LocationsService, locationID)

		t.Run("schedule logical backup", func(t *testing.T) {
			client := backupClient.Default.BackupsService
			backupRes, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
				Body: backups.ScheduleBackupBody{
					ServiceID:      mongo1ID,
					LocationID:     locationID,
					CronExpression: "0 1 1 1 1",
					Name:           "testing",
					Description:    "testing",
					Mode:           pointer.ToString(backups.ScheduleBackupBodyModeBACKUPMODESNAPSHOT),
					Enabled:        false,
					DataModel:      pointer.ToString(backups.StartBackupBodyDataModelDATAMODELLOGICAL),
					Folder:         "backup_folder",
				},
				Context: pmmapitests.Context,
			})

			require.NoError(t, err)
			assert.NotEmpty(t, backupRes.Payload.ScheduledBackupID)

			body := backups.ChangeScheduledBackupBody{
				ScheduledBackupID: backupRes.Payload.ScheduledBackupID,
				Enabled:           pointer.ToBool(true),
				CronExpression:    pointer.ToString("0 2 2 2 2"),
				Name:              pointer.ToString("test2"),
				Description:       pointer.ToString("test2"),
			}
			changeRes, err := client.ChangeScheduledBackup(&backups.ChangeScheduledBackupParams{
				Body:    body,
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			assert.NotEmpty(t, changeRes)

			listRes, err := client.ListScheduledBackups(&backups.ListScheduledBackupsParams{
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			var backup *backups.ListScheduledBackupsOKBodyScheduledBackupsItems0
			for _, b := range listRes.Payload.ScheduledBackups {
				if b.ScheduledBackupID == backupRes.Payload.ScheduledBackupID {
					backup = b
					break
				}
			}
			require.NotNil(t, backup)

			// Assert change
			assert.Equal(t, body.Enabled, backup.Enabled)
			assert.Equal(t, body.Name, backup.Name)
			assert.Equal(t, body.Description, backup.Description)
			assert.Equal(t, body.CronExpression, backup.CronExpression)
			assert.Equal(t, "backup_folder", backup.Folder)

			_, err = client.RemoveScheduledBackup(&backups.RemoveScheduledBackupParams{
				Body: backups.RemoveScheduledBackupBody{
					ScheduledBackupID: backupRes.Payload.ScheduledBackupID,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			find := func(id string, backups []*backups.ListScheduledBackupsOKBodyScheduledBackupsItems0) *backups.ListScheduledBackupsOKBodyScheduledBackupsItems0 {
				for _, b := range backups {
					if b.ScheduledBackupID == id {
						return b
					}
				}
				return nil
			}
			listRes, err = client.ListScheduledBackups(&backups.ListScheduledBackupsParams{
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			assert.NotEmpty(t, listRes)

			deleted := find(backupRes.Payload.ScheduledBackupID, listRes.Payload.ScheduledBackups)
			assert.Nil(t, deleted, "scheduled backup %s is not deleted", backupRes.Payload.ScheduledBackupID)
		})

		t.Run("create multiple snapshot backups", func(t *testing.T) {
			client := backupClient.Default.BackupsService
			sb1, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
				Body: backups.ScheduleBackupBody{
					ServiceID:      mongo1ID,
					LocationID:     locationID,
					CronExpression: "0 1 1 1 1",
					Name:           "testing1",
					Description:    "testing",
					Mode:           pointer.ToString(backups.ScheduleBackupBodyModeBACKUPMODESNAPSHOT),
					Enabled:        true,
					DataModel:      pointer.ToString(backups.StartBackupBodyDataModelDATAMODELLOGICAL),
				},
				Context: pmmapitests.Context,
			})

			require.NoError(t, err)
			defer removeScheduledBackup(t, sb1.Payload.ScheduledBackupID)

			sb2, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
				Body: backups.ScheduleBackupBody{
					ServiceID:      mongo1ID,
					LocationID:     locationID,
					CronExpression: "0 1 1 1 1",
					Name:           "testing2",
					Description:    "testing",
					Mode:           pointer.ToString(backups.ScheduleBackupBodyModeBACKUPMODESNAPSHOT),
					Enabled:        true,
					DataModel:      pointer.ToString(backups.StartBackupBodyDataModelDATAMODELLOGICAL),
				},
				Context: pmmapitests.Context,
			})

			require.NoError(t, err)
			defer removeScheduledBackup(t, sb2.Payload.ScheduledBackupID)
		})

		t.Run("create PITR backup when other backups disabled", func(t *testing.T) {
			client := backupClient.Default.BackupsService

			sb1, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
				Body: backups.ScheduleBackupBody{
					ServiceID:      mongo1ID,
					LocationID:     locationID,
					CronExpression: "0 1 1 1 1",
					Name:           "testing1",
					Description:    "testing",
					Mode:           pointer.ToString(backups.ScheduleBackupBodyModeBACKUPMODESNAPSHOT),
					Enabled:        false,
					DataModel:      pointer.ToString(backups.StartBackupBodyDataModelDATAMODELLOGICAL),
				},
				Context: pmmapitests.Context,
			})

			require.NoError(t, err)
			defer removeScheduledBackup(t, sb1.Payload.ScheduledBackupID)

			pitrb1, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
				Body: backups.ScheduleBackupBody{
					ServiceID:      mongo1ID,
					LocationID:     locationID,
					CronExpression: "0 1 1 1 1",
					Name:           "testing2",
					Description:    "testing",
					Mode:           pointer.ToString(backups.ScheduleBackupBodyModeBACKUPMODEPITR),
					Enabled:        false,
					DataModel:      pointer.ToString(backups.StartBackupBodyDataModelDATAMODELLOGICAL),
				},
				Context: pmmapitests.Context,
			})

			require.NoError(t, err)
			defer removeScheduledBackup(t, pitrb1.Payload.ScheduledBackupID)

			pitrb2, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
				Body: backups.ScheduleBackupBody{
					ServiceID:      mongo1ID,
					LocationID:     locationID,
					CronExpression: "0 1 1 1 1",
					Name:           "testing3",
					Description:    "testing",
					Mode:           pointer.ToString(backups.ScheduleBackupBodyModeBACKUPMODEPITR),
					Enabled:        true,
					DataModel:      pointer.ToString(backups.StartBackupBodyDataModelDATAMODELLOGICAL),
				},
				Context: pmmapitests.Context,
			})

			require.NoError(t, err)
			defer removeScheduledBackup(t, pitrb2.Payload.ScheduledBackupID)
		})

		t.Run("only one enabled PITR backup allowed for the same cluster", func(t *testing.T) {
			client := backupClient.Default.BackupsService
			sb1, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
				Body: backups.ScheduleBackupBody{
					ServiceID:      mongo1ID,
					LocationID:     locationID,
					CronExpression: "0 1 1 1 1",
					Name:           "testing1",
					Description:    "testing",
					Mode:           pointer.ToString(backups.ScheduleBackupBodyModeBACKUPMODEPITR),
					Enabled:        true,
					DataModel:      pointer.ToString(backups.StartBackupBodyDataModelDATAMODELLOGICAL),
				},
				Context: pmmapitests.Context,
			})

			require.NoError(t, err)
			defer removeScheduledBackup(t, sb1.Payload.ScheduledBackupID)

			_, err = client.ScheduleBackup(&backups.ScheduleBackupParams{
				Body: backups.ScheduleBackupBody{
					ServiceID:      mongo2ID,
					LocationID:     locationID,
					CronExpression: "0 1 1 1 1",
					Name:           "testing2",
					Description:    "testing",
					Mode:           pointer.ToString(backups.ScheduleBackupBodyModeBACKUPMODEPITR),
					Enabled:        true,
					DataModel:      pointer.ToString(backups.StartBackupBodyDataModelDATAMODELLOGICAL),
				},
				Context: pmmapitests.Context,
			})

			pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, "A PITR backup for the cluster 'test_cluster' can be enabled only if there are no other scheduled backups for this cluster.")
		})

		t.Run("physical backups fail when PITR is enabled", func(t *testing.T) {
			client := backupClient.Default.BackupsService
			_, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
				Body: backups.ScheduleBackupBody{
					ServiceID:      mongo1ID,
					LocationID:     locationID,
					CronExpression: "0 1 1 1 1",
					Name:           "some_backup_name",
					Description:    "testing",
					Mode:           pointer.ToString(backups.ScheduleBackupBodyModeBACKUPMODEPITR),
					Enabled:        true,
					DataModel:      pointer.ToString(backups.ScheduleBackupBodyDataModelDATAMODELPHYSICAL),
				},
				Context: pmmapitests.Context,
			})

			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "the specified backup model is not compatible with other parameters")
		})

		t.Run("physical backup snapshots can be scheduled", func(t *testing.T) {
			client := backupClient.Default.BackupsService
			backupRes, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
				Body: backups.ScheduleBackupBody{
					ServiceID:      mongo1ID,
					LocationID:     locationID,
					CronExpression: "0 1 1 1 1",
					Name:           "testing",
					Description:    "testing",
					Mode:           pointer.ToString(backups.ScheduleBackupBodyModeBACKUPMODESNAPSHOT),
					Enabled:        true,
					DataModel:      pointer.ToString(backups.ScheduleBackupBodyDataModelDATAMODELPHYSICAL),
				},
				Context: pmmapitests.Context,
			})

			require.NoError(t, err)
			assert.NotNil(t, backupRes.Payload)
			removeScheduledBackup(t, backupRes.Payload.ScheduledBackupID)
		})
	})
}

func removeScheduledBackup(t *testing.T, id string) {
	t.Helper()
	_, err := backupClient.Default.BackupsService.RemoveScheduledBackup(&backups.RemoveScheduledBackupParams{
		Body: backups.RemoveScheduledBackupBody{
			ScheduledBackupID: id,
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
}
