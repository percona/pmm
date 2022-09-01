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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestCheckCompatibility(t *testing.T) {
	t.Parallel()
	mockVersioner := mockVersioner{}
	cSvc := NewCompatibilityService(nil, &mockVersioner)

	t.Run("returns empty if non-mysql service", func(t *testing.T) {
		dbVersion, err := cSvc.checkCompatibility(&models.Service{ServiceType: models.MongoDBServiceType}, &models.Agent{})
		assert.NoError(t, err)
		assert.Equal(t, "", dbVersion)
	})

	agentModel := models.Agent{AgentID: "test_agent_id"}
	software := []agents.Software{
		&agents.Mysqld{},
		&agents.Xtrabackup{},
		&agents.Xbcloud{},
		&agents.Qpress{},
	}

	for _, test := range []struct {
		name          string
		versions      []agents.Version
		expectedError error
	}{
		{
			name: "xtrabackup not installed",
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: ""},
				{Version: ""},
				{Version: "1.1"},
			},
			expectedError: ErrXtrabackupNotInstalled,
		},
		{
			name: "invalid xtrabackup",
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: "8.0.24"},
				{Version: "8.0.25"},
				{Version: "1.1"},
			},
			expectedError: ErrInvalidXtrabackup,
		},
		{
			name: "incompatible xtrabackup",
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: "8.0.24"},
				{Version: "8.0.24"},
				{Version: "1.1"},
			},
			expectedError: ErrIncompatibleXtrabackup,
		},
		{
			name: "qpress no installed",
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: ""},
			},
			expectedError: ErrIncompatibleService,
		},
		{
			name: "mysql no installed",
			versions: []agents.Version{
				{Version: ""},
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: "1.1"},
			},
			expectedError: ErrIncompatibleService,
		},
		{
			name: "error in software version",
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: "8.0.25", Error: "Some error"},
				{Version: "8.0.25"},
				{Version: "1.1"},
			},
			expectedError: ErrComparisonImpossible,
		},
		{
			name: "different version list len",
			versions: []agents.Version{
				{Version: "8.0.25"},
			},
			expectedError: ErrComparisonImpossible,
		},
		{
			name: "successful",
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: "1.1"},
			},
			expectedError: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			mockVersioner.On("GetVersions", agentModel.AgentID, software).Return(test.versions, nil).Once()
			dbVersion, err := cSvc.checkCompatibility(&models.Service{ServiceType: models.MySQLServiceType}, &agentModel)
			if test.expectedError != nil {
				assert.ErrorIs(t, err, test.expectedError)
				assert.Equal(t, "", dbVersion)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.versions[0].Version, dbVersion)
			}
		})
	}
	mock.AssertExpectationsForObjects(t, &mockVersioner)
}

func TestFindCompatibleServiceIDs(t *testing.T) {
	t.Parallel()
	cSvc := NewCompatibilityService(nil, nil)

	testSet := []*models.ServiceSoftwareVersions{
		{
			ServiceID: "1",
			SoftwareVersions: models.SoftwareVersions{
				{Name: models.MysqldSoftwareName, Version: ""},
				{Name: models.XtrabackupSoftwareName, Version: "8.0.25"},
				{Name: models.XbcloudSoftwareName, Version: "8.0.25"},
				{Name: models.QpressSoftwareName, Version: "1.1"},
			},
		},
		{
			ServiceID: "2",
			SoftwareVersions: models.SoftwareVersions{
				{Name: models.MysqldSoftwareName, Version: "8.0.25"},
				{Name: models.XtrabackupSoftwareName, Version: "8.0.24"},
				{Name: models.XbcloudSoftwareName, Version: "8.0.25"},
				{Name: models.QpressSoftwareName, Version: "1.1"},
			},
		},
		{
			ServiceID: "3",
			SoftwareVersions: models.SoftwareVersions{
				{Name: models.MysqldSoftwareName, Version: "8.0.25"},
				{Name: models.XtrabackupSoftwareName, Version: "8.0.25"},
				{Name: models.XbcloudSoftwareName, Version: "8.0.24"},
				{Name: models.QpressSoftwareName, Version: "1.1"},
			},
		},
		{
			ServiceID: "4",
			SoftwareVersions: models.SoftwareVersions{
				{Name: models.MysqldSoftwareName, Version: "8.0.25"},
				{Name: models.XtrabackupSoftwareName, Version: "8.0.25"},
				{Name: models.XbcloudSoftwareName, Version: "8.0.25"},
				{Name: models.QpressSoftwareName, Version: ""},
			},
		},
		{
			ServiceID: "5",
			SoftwareVersions: models.SoftwareVersions{
				{Name: models.MysqldSoftwareName, Version: "8.0.25"},
				{Name: models.XtrabackupSoftwareName, Version: "8.0.25"},
				{Name: models.XbcloudSoftwareName, Version: "8.0.25"},
				{Name: models.QpressSoftwareName, Version: "1.1"},
			},
		},
		{
			ServiceID: "6",
			SoftwareVersions: models.SoftwareVersions{
				{Name: models.MysqldSoftwareName, Version: "8.0.25"},
				{Name: models.XtrabackupSoftwareName, Version: ""},
				{Name: models.XbcloudSoftwareName, Version: "8.0.25"},
				{Name: models.QpressSoftwareName, Version: "1.1"},
			},
		},
		{
			ServiceID: "7",
			SoftwareVersions: models.SoftwareVersions{
				{Name: models.MysqldSoftwareName, Version: "8.0.24"},
				{Name: models.XtrabackupSoftwareName, Version: "8.0.25"},
				{Name: models.XbcloudSoftwareName, Version: "8.0.25"},
				{Name: models.QpressSoftwareName, Version: "1.1"},
			},
		},
		{
			ServiceID: "8",
			SoftwareVersions: models.SoftwareVersions{
				{Name: models.MysqldSoftwareName, Version: "8.0.25"},
				{Name: models.XtrabackupSoftwareName, Version: "8.0.26"},
				{Name: models.XbcloudSoftwareName, Version: "8.0.26"},
				{Name: models.QpressSoftwareName, Version: "1.1"},
			},
		},
	}

	t.Run("empty db version", func(t *testing.T) {
		res := cSvc.findCompatibleServiceIDs(&models.Artifact{DBVersion: ""}, testSet)
		assert.Equal(t, 0, len(res))
	})
	t.Run("matches several", func(t *testing.T) {
		res := cSvc.findCompatibleServiceIDs(&models.Artifact{DBVersion: "8.0.25"}, testSet)
		assert.ElementsMatch(t, []string{"5", "8"}, res)
	})
	t.Run("matches one", func(t *testing.T) {
		res := cSvc.findCompatibleServiceIDs(&models.Artifact{DBVersion: "8.0.24"}, testSet)
		assert.ElementsMatch(t, []string{"7"}, res)
	})
	t.Run("artifact version greater then existing services", func(t *testing.T) {
		res := cSvc.findCompatibleServiceIDs(&models.Artifact{DBVersion: "8.0.30"}, testSet)
		assert.Equal(t, 0, len(res))
	})
}

func TestFindArtifactCompatibleServices(t *testing.T) { //nolint:maintidx
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	cSvc := NewCompatibilityService(db, nil)

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	addRecord := func(records ...reform.Record) {
		// Order matters
		for _, record := range records {
			err := db.Insert(record)
			assert.NoError(t, err)
		}
	}
	dropRecords := func(records ...reform.Record) {
		// Order matters
		for _, record := range records {
			err := db.Delete(record)
			assert.NoError(t, err)
		}
	}

	t.Run("artifact not found", func(t *testing.T) {
		serviceModel, nodeModel, locationModel := setupSoftwareTest(t, db)
		t.Cleanup(func() {
			dropRecords(serviceModel, nodeModel, locationModel)
		})

		artifactModel := models.Artifact{
			ID:         "test_artifact_id",
			Name:       " ",
			Vendor:     "mysql",
			DBVersion:  "8.0.25",
			LocationID: "test_location_id",
			ServiceID:  "test_service_id",
			DataModel:  " ",
			Mode:       " ",
			Status:     " ",
			Type:       " ",
		}
		addRecord(&artifactModel)
		t.Cleanup(func() {
			dropRecords(&artifactModel)
		})

		res, err := cSvc.FindArtifactCompatibleServices(context.Background(), "some_id")
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("empty db version", func(t *testing.T) {
		serviceModel, nodeModel, locationModel := setupSoftwareTest(t, db)
		t.Cleanup(func() {
			dropRecords(serviceModel, nodeModel, locationModel)
		})

		artifactModel := models.Artifact{
			ID:         "test_artifact_id",
			Name:       " ",
			Vendor:     "mysql",
			DBVersion:  "",
			LocationID: "test_location_id",
			ServiceID:  "test_service_id",
			DataModel:  " ",
			Mode:       " ",
			Status:     " ",
			Type:       " ",
		}
		addRecord(&artifactModel)
		t.Cleanup(func() {
			dropRecords(&artifactModel)
		})

		res, err := cSvc.FindArtifactCompatibleServices(context.Background(), "test_artifact_id")
		assert.NoError(t, err)
		assert.ElementsMatch(t, []*models.Service{serviceModel}, res)
	})

	t.Run("non-mysql db vendor", func(t *testing.T) {
		serviceModel, nodeModel, locationModel := setupSoftwareTest(t, db)
		t.Cleanup(func() {
			dropRecords(serviceModel, nodeModel, locationModel)
		})

		artifactModel := models.Artifact{
			ID:         "test_artifact_id",
			Name:       " ",
			Vendor:     "mongodb",
			DBVersion:  "8.0.25",
			LocationID: "test_location_id",
			ServiceID:  "test_service_id",
			DataModel:  " ",
			Mode:       " ",
			Status:     " ",
			Type:       " ",
		}
		addRecord(&artifactModel)
		t.Cleanup(func() {
			dropRecords(&artifactModel)
		})

		res, err := cSvc.FindArtifactCompatibleServices(context.Background(), "test_artifact_id")
		assert.NoError(t, err)
		assert.ElementsMatch(t, []*models.Service{serviceModel}, res)
	})

	t.Run("no software versions data", func(t *testing.T) {
		serviceModel, nodeModel, locationModel := setupSoftwareTest(t, db)
		t.Cleanup(func() {
			dropRecords(serviceModel, nodeModel, locationModel)
		})

		artifactModel := models.Artifact{
			ID:         "test_artifact_id",
			Name:       " ",
			Vendor:     "mysql",
			DBVersion:  "8.0.25",
			LocationID: "test_location_id",
			ServiceID:  "test_service_id",
			DataModel:  " ",
			Mode:       " ",
			Status:     " ",
			Type:       " ",
		}
		addRecord(&artifactModel)
		t.Cleanup(func() {
			dropRecords(&artifactModel)
		})

		res, err := cSvc.FindArtifactCompatibleServices(context.Background(), "test_artifact_id")
		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("find several services", func(t *testing.T) {
		serviceModel, nodeModel, locationModel := setupSoftwareTest(t, db)
		t.Cleanup(func() {
			dropRecords(serviceModel, nodeModel, locationModel)
		})

		artifactModel := models.Artifact{
			ID:         "test_artifact_id",
			Name:       " ",
			Vendor:     "mysql",
			DBVersion:  "8.0.25",
			LocationID: "test_location_id",
			ServiceID:  "test_service_id",
			DataModel:  " ",
			Mode:       " ",
			Status:     " ",
			Type:       " ",
		}

		// Versions data for existing service.
		ssvModel := models.ServiceSoftwareVersions{
			ServiceID:   "test_service_id",
			ServiceType: "mysql",
			SoftwareVersions: models.SoftwareVersions{
				{Name: "mysqld", Version: "8.0.25"},
				{Name: "xtrabackup", Version: "8.0.25"},
				{Name: "xbcloud", Version: "8.0.25"},
				{Name: "qpress", Version: "1.1"},
			},
		}

		type svsData struct {
			service models.Service
			ssv     models.ServiceSoftwareVersions
		}

		svsData2 := svsData{
			service: models.Service{
				ServiceID:   "test_service_id_2",
				ServiceType: " ",
				ServiceName: "test_service_name_2",
				NodeID:      "test_node_id",
			},

			ssv: models.ServiceSoftwareVersions{
				ServiceID:   "test_service_id_2",
				ServiceType: "mysql",
				SoftwareVersions: models.SoftwareVersions{
					{Name: "mysqld", Version: "8.0.25"},
					{Name: "xtrabackup", Version: "8.0.24"},
					{Name: "xbcloud", Version: "8.0.24"},
					{Name: "qpress", Version: "1.1"},
				},
			},
		}
		svsData3 := svsData{
			service: models.Service{
				ServiceID:   "test_service_id_3",
				ServiceType: " ",
				ServiceName: "test_service_name_3",
				NodeID:      "test_node_id",
			},

			ssv: models.ServiceSoftwareVersions{
				ServiceID:   "test_service_id_3",
				ServiceType: "mysql",
				SoftwareVersions: models.SoftwareVersions{
					{Name: "mysqld", Version: "8.0.25"},
					{Name: "xtrabackup", Version: "8.0.25"},
					{Name: "qpress", Version: "1.1"},
				},
			},
		}
		svsData4 := svsData{
			service: models.Service{
				ServiceID:   "test_service_id_4",
				ServiceType: " ",
				ServiceName: "test_service_name_4",
				NodeID:      "test_node_id",
			},

			ssv: models.ServiceSoftwareVersions{
				ServiceID:   "test_service_id_4",
				ServiceType: "mysql",
				SoftwareVersions: models.SoftwareVersions{
					{Name: "mysqld", Version: "8.0.25"},
					{Name: "xtrabackup", Version: "8.0.26"},
					{Name: "xbcloud", Version: "8.0.26"},
					{Name: "qpress", Version: "1.1"},
				},
			},
		}
		svsData5 := svsData{
			service: models.Service{
				ServiceID:   "test_service_id_5",
				ServiceType: " ",
				ServiceName: "test_service_name_5",
				NodeID:      "test_node_id",
			},

			ssv: models.ServiceSoftwareVersions{
				ServiceID:   "test_service_id_5",
				ServiceType: "mongodb",
				SoftwareVersions: models.SoftwareVersions{
					{Name: "mysqld", Version: "8.0.25"},
					{Name: "xtrabackup", Version: "8.0.25"},
					{Name: "xbcloud", Version: "8.0.25"},
					{Name: "qpress", Version: "1.1"},
				},
			},
		}

		addRecord(&artifactModel, &ssvModel)
		addRecord(&svsData2.service, &svsData2.ssv)
		addRecord(&svsData3.service, &svsData3.ssv)
		addRecord(&svsData4.service, &svsData4.ssv)
		addRecord(&svsData5.service, &svsData5.ssv)

		t.Cleanup(func() {
			dropRecords(&svsData2.ssv, &svsData2.service)
			dropRecords(&svsData3.ssv, &svsData3.service)
			dropRecords(&svsData4.ssv, &svsData4.service)
			dropRecords(&svsData5.ssv, &svsData5.service)
			dropRecords(&ssvModel, &artifactModel)
		})

		res, err := cSvc.FindArtifactCompatibleServices(context.Background(), "test_artifact_id")
		assert.NoError(t, err)
		assert.ElementsMatch(t, []*models.Service{serviceModel, &svsData4.service}, res)
	})
}

func setupSoftwareTest(t *testing.T, db *reform.DB) (*models.Service, *models.Node, *models.BackupLocation) {
	locationModel := models.BackupLocation{
		ID:   "test_location_id",
		Name: " ",
		Type: " ",
	}
	nodeModel := models.Node{
		NodeID:   "test_node_id",
		NodeName: " ",
		NodeType: " ",
	}
	serviceModel := models.Service{
		ServiceID:   "test_service_id",
		ServiceType: " ",
		ServiceName: " ",
		NodeID:      "test_node_id",
	}
	err := db.Insert(&locationModel)
	assert.NoError(t, err)
	err = db.Insert(&nodeModel)
	assert.NoError(t, err)
	err = db.Insert(&serviceModel)
	assert.NoError(t, err)

	// Order matters
	return &serviceModel, &nodeModel, &locationModel
}
