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

package models_test

import (
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestSoftwareVersions(t *testing.T) {
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	nodeID1 := "node_id_1"
	serviceID1, serviceID2 := "service_id_1", "service_id_2"

	prepareService := func(q *reform.Querier) {
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
			&models.Service{
				ServiceID:   serviceID2,
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service 2",
				NodeID:      nodeID1,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(777),
			},
		} {
			require.NoError(t, q.Insert(str))
		}
	}

	t.Run("create", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		prepareService(q)

		params := models.CreateServiceSoftwareVersionsParams{
			ServiceID:   serviceID1,
			ServiceType: models.MySQLServiceType,
			SoftwareVersions: []models.SoftwareVersion{
				{
					Name:    models.MysqldSoftwareName,
					Version: "8.0.0",
				},
				{
					Name:    models.XtrabackupSoftwareName,
					Version: "8.0.0",
				},
			},
			NextCheckAt: time.Now().UTC().Truncate(time.Second),
		}

		ssv, err := models.CreateServiceSoftwareVersions(q, params)
		require.NoError(t, err)

		assert.Equal(t, params.ServiceID, ssv.ServiceID)
		assert.Equal(t, params.ServiceType, ssv.ServiceType)
		assert.Equal(t, models.SoftwareVersions(params.SoftwareVersions), ssv.SoftwareVersions)
		assert.Equal(t, params.NextCheckAt, ssv.NextCheckAt)
		assert.Less(t, time.Now().UTC().Unix()-ssv.CreatedAt.Unix(), int64(5))
		assert.Less(t, time.Now().UTC().Unix()-ssv.UpdatedAt.Unix(), int64(5))
	})

	t.Run("list and remove", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		prepareService(q)

		params1 := models.CreateServiceSoftwareVersionsParams{
			ServiceID:   serviceID1,
			ServiceType: models.MySQLServiceType,
			SoftwareVersions: []models.SoftwareVersion{
				{
					Name:    models.MysqldSoftwareName,
					Version: "8.0.0",
				},
				{
					Name:    models.XtrabackupSoftwareName,
					Version: "8.0.0",
				},
			},
			NextCheckAt: time.Now().UTC().Truncate(time.Second).Add(30 * time.Second), // for different ordering
		}
		params2 := models.CreateServiceSoftwareVersionsParams{
			ServiceID:   serviceID2,
			ServiceType: models.MySQLServiceType,
			SoftwareVersions: []models.SoftwareVersion{
				{
					Name:    models.MysqldSoftwareName,
					Version: "5.0.0",
				},
				{
					Name:    models.XtrabackupSoftwareName,
					Version: "5.0.0",
				},
			},
			NextCheckAt: time.Now().UTC().Truncate(time.Second),
		}

		ssv1, err := models.CreateServiceSoftwareVersions(q, params1)
		require.NoError(t, err)
		require.NotNil(t, ssv1)
		ssv2, err := models.CreateServiceSoftwareVersions(q, params2)
		require.NoError(t, err)
		require.NotNil(t, ssv2)

		// TODO Add tests for non-empty FindServicesSoftwareVersionsFilter
		actual, err := models.FindServicesSoftwareVersions(q, models.FindServicesSoftwareVersionsFilter{}, models.SoftwareVersionsOrderByNextCheckAt)
		require.NoError(t, err)
		require.Len(t, actual, 2)

		assertEqual := func(expected models.CreateServiceSoftwareVersionsParams, actual *models.ServiceSoftwareVersions) {
			assert.Equal(t, expected.ServiceID, actual.ServiceID)
			assert.Equal(t, expected.ServiceType, actual.ServiceType)
			assert.Equal(t, models.SoftwareVersions(expected.SoftwareVersions), actual.SoftwareVersions)
			assert.Equal(t, expected.NextCheckAt, actual.NextCheckAt)
			assert.Less(t, time.Now().UTC().Unix()-actual.CreatedAt.Unix(), int64(5))
			assert.Less(t, time.Now().UTC().Unix()-actual.UpdatedAt.Unix(), int64(5))
		}

		assertEqual(params1, actual[1])
		assertEqual(params2, actual[0])

		require.NoError(t, models.RemoveService(q, serviceID1, models.RemoveRestrict))
		actual, err = models.FindServicesSoftwareVersions(q, models.FindServicesSoftwareVersionsFilter{}, models.SoftwareVersionsOrderByNextCheckAt)
		require.NoError(t, err)
		require.Len(t, actual, 1)
		assertEqual(params2, actual[0])

		require.NoError(t, models.DeleteServiceSoftwareVersions(q, serviceID2))
		actual, err = models.FindServicesSoftwareVersions(q, models.FindServicesSoftwareVersionsFilter{}, models.SoftwareVersionsOrderByNextCheckAt)
		require.NoError(t, err)
		require.Len(t, actual, 0)
	})

	t.Run("update", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		prepareService(q)

		createParams := models.CreateServiceSoftwareVersionsParams{
			ServiceID:   serviceID1,
			ServiceType: models.MySQLServiceType,
			SoftwareVersions: []models.SoftwareVersion{
				{
					Name:    models.MysqldSoftwareName,
					Version: "8.0.0",
				},
				{
					Name:    models.XtrabackupSoftwareName,
					Version: "8.0.0",
				},
				{
					Name:    models.XbcloudSoftwareName,
					Version: "8.0.0",
				},
			},
		}

		nextCheck := time.Now().UTC().Truncate(time.Second)
		updateParams := models.UpdateServiceSoftwareVersionsParams{
			SoftwareVersions: []models.SoftwareVersion{
				{
					Name:    models.MysqldSoftwareName,
					Version: "5.0.0",
				},
				{
					Name:    models.XtrabackupSoftwareName,
					Version: "5.0.0",
				},
			},
			NextCheckAt: &nextCheck,
		}

		ssv1, err := models.CreateServiceSoftwareVersions(q, createParams)
		require.NoError(t, err)
		require.NotNil(t, ssv1)
		ssv2, err := models.UpdateServiceSoftwareVersions(q, serviceID1, updateParams)
		require.NoError(t, err)
		require.NotNil(t, ssv2)

		actual, err := models.FindServicesSoftwareVersions(q, models.FindServicesSoftwareVersionsFilter{}, models.SoftwareVersionsOrderByNextCheckAt)
		require.NoError(t, err)
		require.Len(t, actual, 1)

		assert.Equal(t, serviceID1, actual[0].ServiceID)
		assert.Equal(t, models.MySQLServiceType, actual[0].ServiceType)
		assert.Equal(t, nextCheck, actual[0].NextCheckAt)
		assert.ElementsMatch(t, updateParams.SoftwareVersions, actual[0].SoftwareVersions)
	})
}

func TestSoftwareVersionsParamsValidation(t *testing.T) {
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	testCases := []struct {
		name     string
		params   models.CreateServiceSoftwareVersionsParams
		errorMsg string
	}{
		{
			name: "service id is missing",
			params: models.CreateServiceSoftwareVersionsParams{
				ServiceType:      models.MySQLServiceType,
				SoftwareVersions: []models.SoftwareVersion{},
				NextCheckAt:      time.Now().UTC().Truncate(time.Second),
			},
			errorMsg: "invalid argument: service_id shouldn't be empty",
		},
		{
			name: "invalid service type",
			params: models.CreateServiceSoftwareVersionsParams{
				ServiceID:        "service_id",
				ServiceType:      "invalid",
				SoftwareVersions: []models.SoftwareVersion{},
				NextCheckAt:      time.Now().UTC().Truncate(time.Second),
			},
			errorMsg: "invalid argument: invalid service type \"invalid\"",
		},
		{
			name: "invalid software name",
			params: models.CreateServiceSoftwareVersionsParams{
				ServiceID:        "service_id",
				ServiceType:      models.MySQLServiceType,
				SoftwareVersions: []models.SoftwareVersion{{Name: "invalid", Version: "8.0.0"}},
				NextCheckAt:      time.Now().UTC().Truncate(time.Second),
			},
			errorMsg: "invalid argument: invalid software name \"invalid\"",
		},
		{
			name: "empty software version",
			params: models.CreateServiceSoftwareVersionsParams{
				ServiceID:        "service_id",
				ServiceType:      models.MySQLServiceType,
				SoftwareVersions: []models.SoftwareVersion{{Name: models.MysqldSoftwareName}},
				NextCheckAt:      time.Now().UTC().Truncate(time.Second),
			},
			errorMsg: "invalid argument: empty version for software name \"mysqld\"",
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			tx, err := db.Begin()
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, tx.Rollback())
			})

			q := tx.Querier

			c, err := models.CreateServiceSoftwareVersions(q, test.params)
			if test.errorMsg != "" {
				assert.EqualError(t, err, test.errorMsg)
				assert.Nil(t, c)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, c)
		})
	}
}

func TestUpdateServiceSoftwareVersionsParamsValidation(t *testing.T) {
	for _, test := range []struct {
		name     string
		params   models.UpdateServiceSoftwareVersionsParams
		errorMsg string
	}{
		{
			name: "invalid software name",
			params: models.UpdateServiceSoftwareVersionsParams{
				SoftwareVersions: []models.SoftwareVersion{{Name: "invalid", Version: "8.0.0"}},
				NextCheckAt:      pointer.ToTime(time.Now().UTC().Truncate(time.Second)),
			},
			errorMsg: "invalid argument: invalid software name \"invalid\"",
		},
		{
			name: "empty software version",
			params: models.UpdateServiceSoftwareVersionsParams{
				SoftwareVersions: []models.SoftwareVersion{{Name: models.MysqldSoftwareName}},
				NextCheckAt:      pointer.ToTime(time.Now().UTC().Truncate(time.Second)),
			},
			errorMsg: "invalid argument: empty version for software name \"mysqld\"",
		},
		{
			name: "next check time can be nil",
			params: models.UpdateServiceSoftwareVersionsParams{
				SoftwareVersions: []models.SoftwareVersion{{Name: models.MysqldSoftwareName, Version: "8.0.0"}},
				NextCheckAt:      nil,
			},
			errorMsg: "",
		},
		{
			name: "all parameters are filled up",
			params: models.UpdateServiceSoftwareVersionsParams{
				SoftwareVersions: []models.SoftwareVersion{{Name: models.MysqldSoftwareName, Version: "8.0.0"}},
				NextCheckAt:      pointer.ToTime(time.Now().UTC().Truncate(time.Second)),
			},
			errorMsg: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.params.Validate()
			if test.errorMsg != "" {
				assert.EqualError(t, err, test.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
