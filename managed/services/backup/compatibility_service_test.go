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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestCheckCompatibility(t *testing.T) {
	t.Parallel()

	agentModel := models.Agent{AgentID: "test_agent_id"}

	mysqlSoftware := agents.GetRequiredBackupSoftwareList(models.MySQLServiceType)
	expectedMysqlSoftware := []agents.Software{
		&agents.Mysqld{},
		&agents.Xtrabackup{},
		&agents.Xbcloud{},
		&agents.Qpress{},
	}
	require.Equal(t, expectedMysqlSoftware, mysqlSoftware)

	mongoSoftware := agents.GetRequiredBackupSoftwareList(models.MongoDBServiceType)
	expectedMongoSoftware := []agents.Software{
		&agents.MongoDB{},
		&agents.PBM{},
	}
	require.Equal(t, expectedMongoSoftware, mongoSoftware)

	for _, tc := range []struct {
		name          string
		serviceType   models.ServiceType
		versions      []agents.Version
		expectedError error
	}{
		// mysql cases
		{
			name:        "xtrabackup not installed",
			serviceType: models.MySQLServiceType,
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: ""},
				{Version: ""},
				{Version: "1.1"},
			},
			expectedError: ErrXtrabackupNotInstalled,
		},
		{
			name:        "invalid xtrabackup",
			serviceType: models.MySQLServiceType,
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: "8.0.24"},
				{Version: "8.0.25"},
				{Version: "1.1"},
			},
			expectedError: ErrInvalidXtrabackup,
		},
		{
			name:        "incompatible xtrabackup",
			serviceType: models.MySQLServiceType,
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: "8.0.24"},
				{Version: "8.0.24"},
				{Version: "1.1"},
			},
			expectedError: ErrIncompatibleXtrabackup,
		},
		{
			name:        "qpress no installed",
			serviceType: models.MySQLServiceType,
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: ""},
			},
			expectedError: ErrIncompatibleService,
		},
		{
			name:        "mysql no installed",
			serviceType: models.MySQLServiceType,
			versions: []agents.Version{
				{Version: ""},
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: "1.1"},
			},
			expectedError: ErrIncompatibleService,
		},
		{
			name:        "error in software version",
			serviceType: models.MySQLServiceType,
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: "8.0.25", Error: "Some error"},
				{Version: "8.0.25"},
				{Version: "1.1"},
			},
			expectedError: ErrComparisonImpossible,
		},
		{
			name:        "different version list len",
			serviceType: models.MySQLServiceType,
			versions: []agents.Version{
				{Version: "8.0.25"},
			},
			expectedError: ErrComparisonImpossible,
		},
		{
			name:        "successful",
			serviceType: models.MySQLServiceType,
			versions: []agents.Version{
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: "8.0.25"},
				{Version: "1.1"},
			},
			expectedError: nil,
		},
		// mongodb cases
		{
			name:        "successful",
			serviceType: models.MongoDBServiceType,
			versions: []agents.Version{
				{Version: "6.0.2-1"},
				{Version: "2.0.1"},
			},
			expectedError: nil,
		},
		{
			name:        "incompatible pbm version",
			serviceType: models.MongoDBServiceType,
			versions: []agents.Version{
				{Version: "6.0.2-1"},
				{Version: "2.0.0"},
			},
			expectedError: ErrIncompatiblePBM,
		},
		{
			name:        "mongo not installed",
			serviceType: models.MongoDBServiceType,
			versions: []agents.Version{
				{Version: ""},
				{Version: "2.0.1"},
			},
			expectedError: ErrIncompatibleService,
		},
		{
			name:        "pbm not installed",
			serviceType: models.MongoDBServiceType,
			versions: []agents.Version{
				{Version: "6.0.2-1"},
				{Version: ""},
			},
			expectedError: ErrIncompatibleService,
		},
	} {
		tc := tc
		t.Run(string(tc.serviceType)+"_"+tc.name, func(t *testing.T) {
			t.Parallel()
			var sw []agents.Software
			switch tc.serviceType {
			case models.MySQLServiceType:
				sw = mysqlSoftware
			case models.MongoDBServiceType:
				sw = mongoSoftware
			default: // just to satisfy linters
			}
			mockVersioner := mockVersioner{}
			mockVersioner.On("GetVersions", agentModel.AgentID, sw).Return(tc.versions, nil).Once()
			cSvc := NewCompatibilityService(nil, &mockVersioner)
			dbVersion, err := cSvc.checkCompatibility(&models.Service{ServiceType: tc.serviceType}, &agentModel)
			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
				assert.Equal(t, "", dbVersion)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.versions[0].Version, dbVersion)
			}
			mock.AssertExpectationsForObjects(t, &mockVersioner)
		})
	}
}

func TestFindCompatibleServiceIDs(t *testing.T) {
	t.Parallel()
	cSvc := NewCompatibilityService(nil, nil)

	t.Run("mysql", func(t *testing.T) {
		t.Parallel()

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
			t.Parallel()
			res := cSvc.findCompatibleServiceIDs(&models.Artifact{Vendor: "mysql", DBVersion: ""}, testSet)
			assert.Empty(t, res)
		})
		t.Run("matches several", func(t *testing.T) {
			t.Parallel()
			res := cSvc.findCompatibleServiceIDs(&models.Artifact{Vendor: "mysql", DBVersion: "8.0.25"}, testSet)
			assert.ElementsMatch(t, []string{"5", "8"}, res)
		})
		t.Run("matches one", func(t *testing.T) {
			t.Parallel()
			res := cSvc.findCompatibleServiceIDs(&models.Artifact{Vendor: "mysql", DBVersion: "8.0.24"}, testSet)
			assert.ElementsMatch(t, []string{"7"}, res)
		})
		t.Run("artifact version greater then existing services", func(t *testing.T) {
			t.Parallel()
			res := cSvc.findCompatibleServiceIDs(&models.Artifact{Vendor: "mysql", DBVersion: "8.0.30"}, testSet)
			assert.Empty(t, res)
		})
	})

	t.Run("mongo", func(t *testing.T) {
		t.Parallel()

		testSet := []*models.ServiceSoftwareVersions{
			{
				ServiceID: "1",
				SoftwareVersions: models.SoftwareVersions{
					{Name: models.MongoDBSoftwareName, Version: ""},
					{Name: models.PBMSoftwareName, Version: "2.0.1"},
				},
			},
			{
				ServiceID: "2",
				SoftwareVersions: models.SoftwareVersions{
					{Name: models.MongoDBSoftwareName, Version: "6.0.5"},
					{Name: models.PBMSoftwareName, Version: "2.0.0"},
				},
			},
			{
				ServiceID: "3",
				SoftwareVersions: models.SoftwareVersions{
					{Name: models.MongoDBSoftwareName, Version: "6.0.5"},
					{Name: models.PBMSoftwareName, Version: ""},
				},
			},
			{
				ServiceID: "4",
				SoftwareVersions: models.SoftwareVersions{
					{Name: models.MongoDBSoftwareName, Version: "6.0.5"},
					{Name: models.PBMSoftwareName, Version: "2.0.1"},
				},
			},
			{
				ServiceID: "5",
				SoftwareVersions: models.SoftwareVersions{
					{Name: models.MongoDBSoftwareName, Version: "5.0.5"},
					{Name: models.PBMSoftwareName, Version: "2.0.5"},
				},
			},
			{
				ServiceID: "6",
				SoftwareVersions: models.SoftwareVersions{
					{Name: models.MongoDBSoftwareName, Version: "5.0.5"},
					{Name: models.PBMSoftwareName, Version: "2.0.5"},
				},
			},
		}

		t.Run("empty db version", func(t *testing.T) {
			t.Parallel()
			res := cSvc.findCompatibleServiceIDs(&models.Artifact{Vendor: "mongodb", DBVersion: ""}, testSet)
			assert.Empty(t, res)
		})
		t.Run("matches several", func(t *testing.T) {
			t.Parallel()
			res := cSvc.findCompatibleServiceIDs(&models.Artifact{Vendor: "mongodb", DBVersion: "5.0.5"}, testSet)
			assert.ElementsMatch(t, []string{"5", "6"}, res)
		})
		t.Run("matches one", func(t *testing.T) {
			t.Parallel()
			res := cSvc.findCompatibleServiceIDs(&models.Artifact{Vendor: "mongodb", DBVersion: "6.0.5"}, testSet)
			assert.ElementsMatch(t, []string{"4"}, res)
		})
	})
}

func TestFindArtifactCompatibleServices(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	cSvc := NewCompatibilityService(db, nil)

	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	addRecord := func(records ...reform.Record) {
		// Order matters
		for _, record := range records {
			err := db.Insert(record)
			require.NoError(t, err)
		}
	}
	dropRecords := func(records ...reform.Record) {
		// Order matters
		for _, record := range records {
			err := db.Delete(record)
			require.NoError(t, err)
		}
	}

	for _, test := range []struct {
		name               string
		artifactIDToSearch string
		artifact           models.Artifact
		errString          string
		expectEmptyResult  bool
	}{
		{
			name:               "artifact not found",
			artifactIDToSearch: "some_id",
			artifact: models.Artifact{
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
			},
			errString:         "not found",
			expectEmptyResult: true,
		},
		{
			name:               "empty db version",
			artifactIDToSearch: "test_artifact_id",
			artifact: models.Artifact{
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
			},
			errString:         "",
			expectEmptyResult: false,
		},
		{
			name:               "no software versions data for mysql",
			artifactIDToSearch: "test_artifact_id",
			artifact: models.Artifact{
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
			},
			errString:         "",
			expectEmptyResult: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			serviceModel, nodeModel, locationModel := setupSoftwareTest(t, db)
			t.Cleanup(func() {
				dropRecords(serviceModel, nodeModel, locationModel)
			})
			artifact := test.artifact

			addRecord(&artifact)
			t.Cleanup(func() {
				dropRecords(&artifact)
			})

			res, err := cSvc.FindArtifactCompatibleServices(context.Background(), test.artifactIDToSearch)

			if test.errString != "" {
				assert.ErrorContains(t, err, test.errString)
			} else {
				assert.NoError(t, err)
			}

			if test.expectEmptyResult {
				assert.Empty(t, res)
			} else {
				assert.ElementsMatch(t, []*models.Service{serviceModel}, res)
			}
		})
	}

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

func TestArtifactCompatibility(t *testing.T) {
	t.Parallel()
	mockVersioner := mockVersioner{}
	cSvc := NewCompatibilityService(nil, &mockVersioner)

	artifactModelMySQL := &models.Artifact{DBVersion: "8.0.25"}
	artifactModelMongo := &models.Artifact{DBVersion: "6.0.2"}

	serviceModelMySQL := &models.Service{ServiceType: models.MySQLServiceType}
	serviceModelMongo := &models.Service{ServiceType: models.MongoDBServiceType}

	tests := []struct {
		name            string
		artifact        *models.Artifact
		service         *models.Service
		targetDBVersion string
		expectedErr     error
	}{
		// mysql cases
		{
			name:            "mysql successful",
			artifact:        artifactModelMySQL,
			service:         serviceModelMySQL,
			targetDBVersion: "8.0.25",
			expectedErr:     nil,
		},
		{
			name:            "mysql empty artifact version successful",
			artifact:        &models.Artifact{},
			service:         serviceModelMySQL,
			targetDBVersion: "8.0.25",
			expectedErr:     nil,
		},
		{
			name:            "mysql incompatible version",
			artifact:        artifactModelMySQL,
			service:         serviceModelMySQL,
			targetDBVersion: "8.0.24",
			expectedErr:     ErrIncompatibleTargetMySQL,
		},
		{
			name:            "mysql empty target db version",
			artifact:        artifactModelMySQL,
			service:         serviceModelMySQL,
			targetDBVersion: "",
			expectedErr:     ErrIncompatibleTargetMySQL,
		},

		// mongo cases
		{
			name:            "mongo successful",
			artifact:        artifactModelMongo,
			service:         serviceModelMongo,
			targetDBVersion: "6.0.2",
			expectedErr:     nil,
		},
		{
			name:            "mongo incompatible version",
			artifact:        artifactModelMongo,
			service:         serviceModelMongo,
			targetDBVersion: "6.0.1",
			expectedErr:     ErrIncompatibleTargetMongoDB,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := cSvc.artifactCompatibility(tc.artifact, tc.service, tc.targetDBVersion)

			if tc.expectedErr == nil {
				require.NoError(t, err)
				return
			}

			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

func setupSoftwareTest(t *testing.T, db *reform.DB) (*models.Service, *models.Node, *models.BackupLocation) {
	t.Helper()
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
	require.NoError(t, err)
	err = db.Insert(&nodeModel)
	require.NoError(t, err)
	err = db.Insert(&serviceModel)
	require.NoError(t, err)

	// Order matters
	return &serviceModel, &nodeModel, &locationModel
}
