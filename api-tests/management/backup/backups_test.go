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
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/brianvoe/gofakeit/v6"
	backupClient "github.com/percona/pmm/api/managementpb/backup/json/client"
	"github.com/percona/pmm/api/managementpb/backup/json/client/backups"
	"github.com/percona/pmm/api/managementpb/backup/json/client/locations"
	managementClient "github.com/percona/pmm/api/managementpb/json/client"
	mysql "github.com/percona/pmm/api/managementpb/json/client/my_sql"
	"github.com/percona/pmm/api/managementpb/json/client/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm-managed/api-tests"
	"github.com/percona/pmm-managed/api-tests/management"
)

func TestScheduleBackup(t *testing.T) {
	t.Parallel()

	nodeName := pmmapitests.TestString(t, "node-for-basic-name")
	nodeID, pmmAgentID := management.RegisterGenericNode(t, node.RegisterNodeBody{
		NodeName: nodeName,
		NodeType: pointer.ToString(node.RegisterNodeBodyNodeTypeGENERICNODE),
	})
	defer pmmapitests.RemoveNodes(t, nodeID)
	defer management.RemovePMMAgentWithSubAgents(t, pmmAgentID)
	serviceName := pmmapitests.TestString(t, "service-for-basic-name")

	params := &mysql.AddMySQLParams{
		Context: pmmapitests.Context,
		Body: mysql.AddMySQLBody{
			NodeID:      nodeID,
			PMMAgentID:  pmmAgentID,
			ServiceName: serviceName,
			Address:     "10.10.10.10",
			Port:        3306,
			Username:    "username",

			SkipConnectionCheck: true,
			DisableCollectors:   []string{"global_status", "perf_schema.tablelocks"},
		},
	}
	addMySQLOK, err := managementClient.Default.MySQL.AddMySQL(params)
	require.NoError(t, err)
	serviceID := addMySQLOK.Payload.Service.ServiceID
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
	defer deleteLocation(t, backupClient.Default.Locations, resp.Payload.LocationID)

	client := backupClient.Default.Backups
	backupRes, err := client.ScheduleBackup(&backups.ScheduleBackupParams{
		Body: backups.ScheduleBackupBody{
			ServiceID:      serviceID,
			LocationID:     resp.Payload.LocationID,
			CronExpression: "0 1 1 1 1",
			Name:           "testing",
			Description:    "testing",
			Enabled:        false,
		},
		Context: pmmapitests.Context,
	})

	assert.NoError(t, err)
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

	assert.NoError(t, err)
	assert.NotNil(t, changeRes)

	listRes, err := client.ListScheduledBackups(&backups.ListScheduledBackupsParams{
		Context: pmmapitests.Context,
	})

	assert.NoError(t, err)
	var backup *backups.ScheduledBackupsItems0
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
	assert.NoError(t, err)

	find := func(id string, backups []*backups.ScheduledBackupsItems0) *backups.ScheduledBackupsItems0 {
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
	assert.NoError(t, err)
	require.NotNil(t, listRes)

	deleted := find(backupRes.Payload.ScheduledBackupID, listRes.Payload.ScheduledBackups)
	assert.Nil(t, deleted, "scheduled backup %s is not deleted", backupRes.Payload.ScheduledBackupID)
}
