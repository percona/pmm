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
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api-tests/management"
	backupClient "github.com/percona/pmm/api/managementpb/backup/json/client"
	"github.com/percona/pmm/api/managementpb/backup/json/client/backups"
	"github.com/percona/pmm/api/managementpb/backup/json/client/locations"
	managementClient "github.com/percona/pmm/api/managementpb/json/client"
	mongodb "github.com/percona/pmm/api/managementpb/json/client/mongo_db"
	"github.com/percona/pmm/api/managementpb/json/client/node"
)

func TestMongoBackup(t *testing.T) {
	nodeName := pmmapitests.TestString(t, "node-for-basic-name")
	nodeID, pmmAgentID := management.RegisterGenericNode(t, node.RegisterNodeBody{
		NodeName: nodeName,
		NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
	})
	defer pmmapitests.RemoveNodes(t, nodeID)
	defer management.RemovePMMAgentWithSubAgents(t, pmmAgentID)
	serviceName := pmmapitests.TestString(t, "service-for-basic-name")

	params := &mongodb.AddMongoDBParams{
		Context: pmmapitests.Context,
		Body: mongodb.AddMongoDBBody{
			NodeID:      nodeID,
			PMMAgentID:  pmmAgentID,
			ServiceName: serviceName,
			Address:     "10.10.10.10",
			Port:        27017,
			Username:    "username",

			SkipConnectionCheck: true,
			DisableCollectors:   []string{"global_status", "perf_schema.tablelocks"},
		},
	}
	addMongoDBOK, err := managementClient.Default.MongoDB.AddMongoDB(params)
	require.NoError(t, err)
	serviceID := addMongoDBOK.Payload.Service.ServiceID
	defer pmmapitests.RemoveServices(t, serviceID)

	resp, err := backupClient.Default.Locations.AddLocation(&locations.AddLocationParams{
		Body: locations.AddLocationBody{
			Name:        gofakeit.Name(),
			Description: gofakeit.Question(),
			PMMClientConfig: &locations.AddLocationParamsBodyPMMClientConfig{
				Path: "/tmp",
			},
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	locationID := resp.Payload.LocationID
	defer deleteLocation(t, backupClient.Default.Locations, locationID)

	t.Run("normal", func(t *testing.T) {
		client := backupClient.Default.Backups
		backupRes, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
			Body: backups.ScheduleBackupBody{
				ServiceID:      serviceID,
				LocationID:     locationID,
				CronExpression: "0 1 1 1 1",
				Name:           "testing",
				Description:    "testing",
				Mode:           pointer.ToString(backups.ScheduleBackupBodyModeSNAPSHOT),
				Enabled:        false,
			},
			Context: pmmapitests.Context,
		})

		require.NoError(t, err)
		assert.NotEmpty(t, backupRes.Payload.ScheduledBackupID)

		body := backups.ChangeScheduledBackupBody{
			ScheduledBackupID: backupRes.Payload.ScheduledBackupID,
			Enabled:           true,
			CronExpression:    "0 2 2 2 2",
			Name:              "test2",
			Description:       "test2",
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
		client := backupClient.Default.Backups
		sb1, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
			Body: backups.ScheduleBackupBody{
				ServiceID:      serviceID,
				LocationID:     locationID,
				CronExpression: "0 1 1 1 1",
				Name:           "testing",
				Description:    "testing",
				Mode:           pointer.ToString(backups.ScheduleBackupBodyModeSNAPSHOT),
				Enabled:        true,
			},
			Context: pmmapitests.Context,
		})

		require.NoError(t, err)
		defer removeScheduledBackup(t, sb1.Payload.ScheduledBackupID)

		sb2, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
			Body: backups.ScheduleBackupBody{
				ServiceID:      serviceID,
				LocationID:     locationID,
				CronExpression: "0 1 1 1 1",
				Name:           "testing",
				Description:    "testing",
				Mode:           pointer.ToString(backups.ScheduleBackupBodyModeSNAPSHOT),
				Enabled:        true,
			},
			Context: pmmapitests.Context,
		})

		require.NoError(t, err)
		defer removeScheduledBackup(t, sb2.Payload.ScheduledBackupID)
	})

	t.Run("create PITR backup when other backups disabled", func(t *testing.T) {
		client := backupClient.Default.Backups

		sb1, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
			Body: backups.ScheduleBackupBody{
				ServiceID:      serviceID,
				LocationID:     locationID,
				CronExpression: "0 1 1 1 1",
				Name:           "testing",
				Description:    "testing",
				Mode:           pointer.ToString(backups.ScheduleBackupBodyModeSNAPSHOT),
				Enabled:        false,
			},
			Context: pmmapitests.Context,
		})

		require.NoError(t, err)
		defer removeScheduledBackup(t, sb1.Payload.ScheduledBackupID)

		pitrb1, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
			Body: backups.ScheduleBackupBody{
				ServiceID:      serviceID,
				LocationID:     locationID,
				CronExpression: "0 1 1 1 1",
				Name:           "testing",
				Description:    "testing",
				Mode:           pointer.ToString(backups.ScheduleBackupBodyModePITR),
				Enabled:        false,
			},
			Context: pmmapitests.Context,
		})

		require.NoError(t, err)
		defer removeScheduledBackup(t, pitrb1.Payload.ScheduledBackupID)

		pitrb2, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
			Body: backups.ScheduleBackupBody{
				ServiceID:      serviceID,
				LocationID:     locationID,
				CronExpression: "0 1 1 1 1",
				Name:           "testing",
				Description:    "testing",
				Mode:           pointer.ToString(backups.ScheduleBackupBodyModePITR),
				Enabled:        true,
			},
			Context: pmmapitests.Context,
		})

		require.NoError(t, err)
		defer removeScheduledBackup(t, pitrb2.Payload.ScheduledBackupID)
	})

	t.Run("only one enabled PITR backup allowed", func(t *testing.T) {
		client := backupClient.Default.Backups
		sb1, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
			Body: backups.ScheduleBackupBody{
				ServiceID:      serviceID,
				LocationID:     locationID,
				CronExpression: "0 1 1 1 1",
				Name:           "testing",
				Description:    "testing",
				Mode:           pointer.ToString(backups.ScheduleBackupBodyModePITR),
				Enabled:        true,
			},
			Context: pmmapitests.Context,
		})

		require.NoError(t, err)
		defer removeScheduledBackup(t, sb1.Payload.ScheduledBackupID)

		_, err = client.ScheduleBackup(&backups.ScheduleBackupParams{
			Body: backups.ScheduleBackupBody{
				ServiceID:      serviceID,
				LocationID:     locationID,
				CronExpression: "0 1 1 1 1",
				Name:           "testing",
				Description:    "testing",
				Mode:           pointer.ToString(backups.ScheduleBackupBodyModePITR),
				Enabled:        true,
			},
			Context: pmmapitests.Context,
		})

		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, "A scheduled PITR backup can be enabled only if there  no other scheduled backups.")
	})

	t.Run("prevent snapshot backups when PITR enabled", func(t *testing.T) {
		client := backupClient.Default.Backups
		pitrb1, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
			Body: backups.ScheduleBackupBody{
				ServiceID:      serviceID,
				LocationID:     locationID,
				CronExpression: "0 1 1 1 1",
				Name:           "testing",
				Description:    "testing",
				Mode:           pointer.ToString(backups.ScheduleBackupBodyModePITR),
				Enabled:        true,
			},
			Context: pmmapitests.Context,
		})

		require.NoError(t, err)
		defer removeScheduledBackup(t, pitrb1.Payload.ScheduledBackupID)

		_, err = client.StartBackup(&backups.StartBackupParams{
			Body: backups.StartBackupBody{
				ServiceID:   serviceID,
				LocationID:  locationID,
				Name:        "test-snapshot",
				Description: "Test snapshot.",
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, "Can't make a backup because service %s already has scheduled PITR backups. Please disable them if you want to make another backup.", serviceName)
	})

	t.Run("physical backups fail when PITR is enabled", func(t *testing.T) {
		client := backupClient.Default.Backups
		_, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
			Body: backups.ScheduleBackupBody{
				ServiceID:      serviceID,
				LocationID:     locationID,
				CronExpression: "0 1 1 1 1",
				Name:           t.Name(),
				Description:    "testing",
				Mode:           pointer.ToString(backups.ScheduleBackupBodyModePITR),
				Enabled:        true,
				DataModel:      pointer.ToString(backups.ScheduleBackupBodyDataModelPHYSICAL),
			},
			Context: pmmapitests.Context,
		})

		assert.Error(t, err)
	})

	t.Run("physical backup snapshots can be scheduled", func(t *testing.T) {
		client := backupClient.Default.Backups
		backupRes, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
			Body: backups.ScheduleBackupBody{
				ServiceID:      serviceID,
				LocationID:     locationID,
				CronExpression: "0 1 1 1 1",
				Name:           "testing",
				Description:    "testing",
				Mode:           pointer.ToString(backups.ScheduleBackupBodyModeSNAPSHOT),
				Enabled:        true,
				DataModel:      pointer.ToString(backups.ScheduleBackupBodyDataModelPHYSICAL),
			},
			Context: pmmapitests.Context,
		})

		require.NoError(t, err)
		assert.NotNil(t, backupRes.Payload)
		removeScheduledBackup(t, backupRes.Payload.ScheduledBackupID)
	})
}

func removeScheduledBackup(t *testing.T, id string) {
	_, err := backupClient.Default.Backups.RemoveScheduledBackup(&backups.RemoveScheduledBackupParams{
		Body: backups.RemoveScheduledBackupBody{
			ScheduledBackupID: id,
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
}
