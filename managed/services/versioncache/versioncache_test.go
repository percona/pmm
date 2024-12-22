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

package versioncache

import (
	"context"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestVersionCache(t *testing.T) {
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Run("success", func(t *testing.T) {
		versionerMock := &MockVersioner{}
		versionerMock.Test(t)

		nodeID1 := "node_id_1"
		serviceID1 := "service_id_1"
		agentID1, agentID2 := "agent_id_1", "agent_id_2"
		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:   nodeID1,
				NodeType: models.GenericNodeType,
				NodeName: "Node 1",
			},
			&models.Service{
				ServiceID:   serviceID1,
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service 1",
				NodeID:      nodeID1,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(777),
			},
			&models.ServiceSoftwareVersions{
				ServiceID:        serviceID1,
				ServiceType:      models.MySQLServiceType,
				SoftwareVersions: nil,
				NextCheckAt:      time.Now(),
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
			&models.Agent{
				AgentID:      agentID1,
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: &nodeID1,
			},
			&models.Agent{
				AgentID:      agentID2,
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   &agentID1,
				RunsOnNodeID: nil,
				ServiceID:    &serviceID1,
			},
		} {
			require.NoError(t, db.Insert(str))
		}
		t.Cleanup(func() {
			assert.NoError(t, db.Delete(&models.Agent{AgentID: agentID2}))
			assert.NoError(t, db.Delete(&models.Agent{AgentID: agentID1}))
			assert.NoError(t, db.Delete(&models.Service{ServiceID: serviceID1}))
			assert.NoError(t, db.Delete(&models.Node{NodeID: nodeID1}))
		})

		softwares := agents.GetRequiredBackupSoftwareList(models.MySQLServiceType)
		versions1 := []agents.Version{
			{Version: "8.0.23"},
			{Version: "8.0.23"},
			{Version: "8.0.23"},
			{Version: "1.1"},
		}
		versionerMock.On("GetVersions", agentID1, softwares).Return(versions1, nil).Once()

		done := make(chan struct{}, 1)
		mockGetVersions := func(oldVersions, newVersions []agents.Version, finish bool) {
			versionerMock.On("GetVersions", agentID1, softwares).Return(newVersions, nil).Run(func(args mock.Arguments) {
				v, err := models.FindServiceSoftwareVersionsByServiceID(db.Querier, serviceID1)
				require.NoError(t, err)
				require.NotNil(t, v)

				require.Equal(t, serviceID1, v.ServiceID)
				require.Equal(t, models.MySQLServiceType, v.ServiceType)
				softwareVersions := models.SoftwareVersions{
					{
						Name:    models.MysqldSoftwareName,
						Version: oldVersions[0].Version,
					},
					{
						Name:    models.XtrabackupSoftwareName,
						Version: oldVersions[1].Version,
					},
					{
						Name:    models.XbcloudSoftwareName,
						Version: oldVersions[2].Version,
					},
					{
						Name:    models.QpressSoftwareName,
						Version: oldVersions[3].Version,
					},
				}
				require.Equal(t, softwareVersions, v.SoftwareVersions)

				if finish {
					done <- struct{}{}
				}
			}).Once()
		}

		versions2 := []agents.Version{
			{Version: "8.0.24"},
			{Version: "5.0.25"},
			{Version: "5.0.25"},
			{Version: "0.1"},
		}
		mockGetVersions(versions1, versions2, false)
		mockGetVersions(versions2, versions2, true)

		// the test is finished, but make a universal mock for all the other version updates.
		versionerMock.On("GetVersions", agentID1, softwares).Return(versions2, nil)

		ctx, cancel := context.WithCancel(context.Background())

		serviceCheckInterval = time.Second
		minCheckInterval = 0
		startupDelay = 0

		cache := New(db, versionerMock)
		go func() {
			cache.Run(ctx)
		}()

		select {
		case <-time.After(5 * time.Second):
			t.FailNow()
		case <-done:
		}

		cancel()
		versionerMock.AssertExpectations(t)
		// Sleep here so cache.Run() has time to finish its run.
		// Otherwise, the tests fail here due to t.Log() being called after
		// the test has finished.
		<-time.After(1200 * time.Millisecond)
	})

	t.Run("no version request if no backup software declared", func(t *testing.T) {
		nodeID1 := "node_id_1"
		serviceID1 := "service_id_1"
		agentID1, agentID2 := "agent_id_1", "agent_id_2"
		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:   nodeID1,
				NodeType: models.GenericNodeType,
				NodeName: "Node 1",
			},
			&models.Service{
				ServiceID:   serviceID1,
				ServiceType: models.PostgreSQLServiceType,
				ServiceName: "Service 1",
				NodeID:      nodeID1,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(777),
			},
			&models.ServiceSoftwareVersions{
				ServiceID:        serviceID1,
				ServiceType:      models.PostgreSQLServiceType,
				SoftwareVersions: nil,
				NextCheckAt:      time.Now(),
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
			&models.Agent{
				AgentID:      agentID1,
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: &nodeID1,
			},
			&models.Agent{
				AgentID:      agentID2,
				AgentType:    models.PostgresExporterType,
				PMMAgentID:   &agentID1,
				RunsOnNodeID: nil,
				ServiceID:    &serviceID1,
			},
		} {
			require.NoError(t, db.Insert(str))
		}
		t.Cleanup(func() {
			assert.NoError(t, db.Delete(&models.Agent{AgentID: agentID2}))
			assert.NoError(t, db.Delete(&models.Agent{AgentID: agentID1}))
			assert.NoError(t, db.Delete(&models.Service{ServiceID: serviceID1}))
			assert.NoError(t, db.Delete(&models.Node{NodeID: nodeID1}))
		})

		versionerMock := &MockVersioner{}
		cache := New(db, versionerMock)
		nextCheck, err := cache.updateVersionsForNextService()
		assert.ErrorIs(t, err, ErrInvalidArgument)
		assert.Equal(t, minCheckInterval, nextCheck)

		versionerMock.AssertNotCalled(t, "GetVersions")
	})
}
