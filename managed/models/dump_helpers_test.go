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
	"sort"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestDumps(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	tx, err := db.Begin()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.Rollback())
	})

	t.Run("create", func(t *testing.T) {
		t.Run("normal", func(t *testing.T) {
			endTime := time.Now()
			startTime := endTime.Add(-10 * time.Minute)

			createDumpParams := models.CreateDumpParams{
				ServiceNames: []string{"foo", "bar"},
				StartTime:    &startTime,
				EndTime:      &endTime,
				ExportQAN:    false,
				IgnoreLoad:   true,
			}
			dump, err := models.CreateDump(tx.Querier, createDumpParams)
			require.NoError(t, err)
			assert.NotEmpty(t, dump.ID)
			assert.Equal(t, models.DumpStatusInProgress, dump.Status)
			assert.ElementsMatch(t, createDumpParams.ServiceNames, dump.ServiceNames)
			assert.Equal(t, createDumpParams.StartTime, dump.StartTime)
			assert.Equal(t, createDumpParams.EndTime, dump.EndTime)
			assert.Equal(t, createDumpParams.ExportQAN, dump.ExportQAN)
			assert.Equal(t, createDumpParams.IgnoreLoad, dump.IgnoreLoad)
		})

		t.Run("invalid start and end time", func(t *testing.T) {
			endTime := time.Now()
			startTime := endTime.Add(10 * time.Minute)

			createDumpParams := models.CreateDumpParams{
				ServiceNames: []string{"foo", "bar"},
				StartTime:    &startTime,
				EndTime:      &endTime,
				ExportQAN:    false,
				IgnoreLoad:   true,
			}
			_, err := models.CreateDump(tx.Querier, createDumpParams)
			require.EqualError(t, err, "invalid dump creation params: dump start time can't be greater than end time")
		})
	})

	t.Run("find", func(t *testing.T) {
		findTX, err := db.Begin()
		require.NoError(t, err)
		defer findTX.Rollback() //nolint:errcheck

		endTime := time.Now()
		startTime := endTime.Add(-10 * time.Minute)

		dump1, err := models.CreateDump(findTX.Querier, models.CreateDumpParams{
			ServiceNames: []string{"foo", "bar"},
			StartTime:    &startTime,
			EndTime:      &endTime,
			ExportQAN:    false,
			IgnoreLoad:   true,
		})
		require.NoError(t, err)

		dump2, err := models.CreateDump(findTX.Querier, models.CreateDumpParams{
			ServiceNames: []string{"foo", "bar"},
			StartTime:    &startTime,
			EndTime:      &endTime,
			ExportQAN:    false,
			IgnoreLoad:   true,
		})
		require.NoError(t, err)
		dump2.Status = models.DumpStatusSuccess
		err = models.UpdateDumpStatus(findTX.Querier, dump2.ID, dump2.Status)
		require.NoError(t, err)

		dump3, err := models.CreateDump(findTX.Querier, models.CreateDumpParams{
			ServiceNames: []string{"foo", "bar"},
			StartTime:    &startTime,
			EndTime:      &endTime,
			ExportQAN:    false,
			IgnoreLoad:   true,
		})
		require.NoError(t, err)
		dump3.Status = models.DumpStatusError
		err = findTX.Querier.Update(dump3)
		require.NoError(t, err)

		type testCase struct {
			Filters models.DumpFilters
			Expect  []string
		}

		testCases := []testCase{
			{
				Filters: models.DumpFilters{},
				Expect:  []string{dump1.ID, dump2.ID, dump3.ID},
			},
			{
				Filters: models.DumpFilters{
					Status: models.DumpStatusInProgress,
				},
				Expect: []string{dump1.ID},
			},
			{
				Filters: models.DumpFilters{
					Status: models.DumpStatusSuccess,
				},
				Expect: []string{dump2.ID},
			},
			{
				Filters: models.DumpFilters{
					Status: models.DumpStatusError,
				},
				Expect: []string{dump3.ID},
			},
		}

		for _, tc := range testCases {
			dumps, err := models.FindDumps(findTX.Querier, tc.Filters)
			require.NoError(t, err)
			ids := make([]string, len(dumps))
			for i := range dumps {
				ids[i] = dumps[i].ID
			}
			sort.Strings(tc.Expect)
			sort.Strings(ids)
			assert.Equal(t, tc.Expect, ids)
		}
	})
}

func TestDumpLogs(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	tx, err := db.Begin()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.Rollback())
		require.NoError(t, sqlDB.Close())
	})

	dump1, err := models.CreateDump(tx.Querier, models.CreateDumpParams{})
	require.NoError(t, err)

	dump2, err := models.CreateDump(tx.Querier, models.CreateDumpParams{})
	require.NoError(t, err)

	createRequests := []models.CreateDumpLogParams{
		{
			DumpID:  dump1.ID,
			ChunkID: 0,
			Data:    "some log",
		},
		{
			DumpID:  dump1.ID,
			ChunkID: 1,
			Data:    "another log",
		},
		{
			DumpID:  dump2.ID,
			ChunkID: 0,
			Data:    "some log",
		},
	}

	t.Run("create", func(t *testing.T) {
		for _, req := range createRequests {
			log, err := models.CreateDumpLog(tx.Querier, req)
			require.NoError(t, err)
			assert.Equal(t, req.DumpID, log.DumpID)
			assert.Equal(t, req.ChunkID, log.ChunkID)
			assert.Equal(t, req.Data, log.Data)
			assert.False(t, log.LastChunk)
		}
	})

	t.Run("find", func(t *testing.T) {
		type expectLog struct {
			DumpID  string
			ChunkID uint32
		}
		type testCase struct {
			Name    string
			Filters models.DumpLogsFilter
			Expect  []expectLog
		}
		testCases := []testCase{
			{
				Name: "dump filter",
				Filters: models.DumpLogsFilter{
					DumpID: dump1.ID,
				},
				Expect: []expectLog{
					{
						DumpID:  dump1.ID,
						ChunkID: 0,
					},
					{
						DumpID:  dump1.ID,
						ChunkID: 1,
					},
				},
			},
			{
				Name: "dump filter and limit",
				Filters: models.DumpLogsFilter{
					DumpID: dump1.ID,
					Limit:  pointer.ToInt(1),
				},
				Expect: []expectLog{
					{
						DumpID:  dump1.ID,
						ChunkID: 0,
					},
				},
			},
			{
				Name: "dump filter. limit and offset",
				Filters: models.DumpLogsFilter{
					DumpID: dump1.ID,
					Offset: 1,
					Limit:  pointer.ToInt(1),
				},
				Expect: []expectLog{
					{
						DumpID:  dump1.ID,
						ChunkID: 1,
					},
				},
			},
		}

		for _, tc := range testCases {
			tc := tc
			t.Run(tc.Name, func(t *testing.T) {
				logs, err := models.FindDumpLogs(tx.Querier, tc.Filters)
				require.NoError(t, err)
				require.Len(t, logs, len(tc.Expect))
				for i := range logs {
					assert.Equal(t, tc.Expect[i].DumpID, logs[i].DumpID)
					assert.Equal(t, tc.Expect[i].ChunkID, logs[i].ChunkID)
				}
			})
		}
	})
}
