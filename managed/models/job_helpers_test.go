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
	"strconv"
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

func TestJobs(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	tx, err := db.Begin()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.Rollback())
		require.NoError(t, sqlDB.Close())
	})

	t.Run("create", func(t *testing.T) {
		createJobParams := models.CreateJobParams{
			PMMAgentID: "agentid",
			Type:       models.MongoDBBackupJob,
			Data: &models.JobData{
				MongoDBBackup: &models.MongoDBBackupJobData{
					ServiceID:  "svc",
					ArtifactID: "artifactid",
				},
			},
			Timeout:  time.Second,
			Interval: time.Second,
			Retries:  3,
		}
		job, err := models.CreateJob(tx.Querier, createJobParams)
		require.NoError(t, err)
		assert.Equal(t, createJobParams.PMMAgentID, job.PMMAgentID)
		assert.Equal(t, createJobParams.Type, job.Type)
		assert.Equal(t, createJobParams.Timeout, job.Timeout)
		assert.Equal(t, createJobParams.Interval, job.Interval)
		assert.Equal(t, createJobParams.Retries, job.Retries)
		require.NotNil(t, job.Data.MongoDBBackup)
		assert.Equal(t, createJobParams.Data.MongoDBBackup.ServiceID, job.Data.MongoDBBackup.ServiceID)
		assert.Equal(t, createJobParams.Data.MongoDBBackup.ArtifactID, job.Data.MongoDBBackup.ArtifactID)

		_, err = models.CreateJob(tx.Querier, models.CreateJobParams{Type: "unknown"})
		assert.EqualError(t, err, "unknown job type: unknown")
	})

	t.Run("find", func(t *testing.T) {
		findTX, err := db.Begin()
		require.NoError(t, err)
		defer findTX.Rollback() //nolint:errcheck

		const jobsCount = 3
		jobs := make([]*models.Job, 0, jobsCount)
		for i := 0; i < jobsCount; i++ {
			id := strconv.Itoa(i)
			job, err := models.CreateJob(findTX.Querier, models.CreateJobParams{
				PMMAgentID: "agentid",
				Type:       models.MongoDBBackupJob,
				Data: &models.JobData{
					MongoDBBackup: &models.MongoDBBackupJobData{
						ServiceID:  "svc_" + id,
						ArtifactID: "artifact_" + id,
					},
				},
			})
			require.NoError(t, err)
			jobs = append(jobs, job)
		}

		type testCase struct {
			Filters models.JobsFilter
			Expect  []string
		}

		testCases := []testCase{
			{
				Filters: models.JobsFilter{},
				Expect:  []string{jobs[0].ID, jobs[1].ID, jobs[2].ID},
			},
			{
				Filters: models.JobsFilter{
					ArtifactID: jobs[0].Data.MongoDBBackup.ArtifactID,
				},
				Expect: []string{jobs[0].ID},
			},
		}

		for _, tc := range testCases {
			jobs, err := models.FindJobs(findTX.Querier, tc.Filters)
			require.NoError(t, err)
			ids := make([]string, len(jobs))
			for i := range jobs {
				ids[i] = jobs[i].ID
			}
			sort.Strings(tc.Expect)
			sort.Strings(ids)
			assert.Equal(t, tc.Expect, ids)
		}
	})
}

func TestJobLogs(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	tx, err := db.Begin()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.Rollback())
		require.NoError(t, sqlDB.Close())
	})

	job1, err := models.CreateJob(tx.Querier, models.CreateJobParams{
		PMMAgentID: "pmmagent",
		Type:       models.MongoDBBackupJob,
		Data:       &models.JobData{},
	})
	require.NoError(t, err)

	job2, err := models.CreateJob(tx.Querier, models.CreateJobParams{
		PMMAgentID: "pmmagent",
		Type:       models.MongoDBBackupJob,
		Data:       &models.JobData{},
	})
	require.NoError(t, err)

	createRequests := []models.CreateJobLogParams{
		{
			JobID:   job1.ID,
			ChunkID: 0,
			Data:    "some log",
		},
		{
			JobID:   job1.ID,
			ChunkID: 1,
			Data:    "another log",
		},
		{
			JobID:   job2.ID,
			ChunkID: 0,
			Data:    "some log",
		},
	}

	t.Run("create", func(t *testing.T) {
		for _, req := range createRequests {
			log, err := models.CreateJobLog(tx.Querier, req)
			require.NoError(t, err)
			assert.Equal(t, req.JobID, log.JobID)
			assert.Equal(t, req.ChunkID, log.ChunkID)
			assert.Equal(t, req.Data, log.Data)
			assert.False(t, log.LastChunk)
		}
	})

	t.Run("find", func(t *testing.T) {
		type expectLog struct {
			JobID   string
			ChunkID int
		}
		type testCase struct {
			Name    string
			Filters models.JobLogsFilter
			Expect  []expectLog
		}
		testCases := []testCase{
			{
				Name: "job filter",
				Filters: models.JobLogsFilter{
					JobID: job1.ID,
				},
				Expect: []expectLog{
					{
						JobID:   job1.ID,
						ChunkID: 0,
					},
					{
						JobID:   job1.ID,
						ChunkID: 1,
					},
				},
			},
			{
				Name: "job filter and limit",
				Filters: models.JobLogsFilter{
					JobID: job1.ID,
					Limit: pointer.ToInt(1),
				},
				Expect: []expectLog{
					{
						JobID:   job1.ID,
						ChunkID: 0,
					},
				},
			},
			{
				Name: "job filter. limit and offset",
				Filters: models.JobLogsFilter{
					JobID:  job1.ID,
					Offset: 1,
					Limit:  pointer.ToInt(1),
				},
				Expect: []expectLog{
					{
						JobID:   job1.ID,
						ChunkID: 1,
					},
				},
			},
		}

		for _, tc := range testCases {
			tc := tc
			t.Run(tc.Name, func(t *testing.T) {
				logs, err := models.FindJobLogs(tx.Querier, tc.Filters)
				require.NoError(t, err)
				require.Len(t, logs, len(tc.Expect))
				for i := range logs {
					assert.Equal(t, tc.Expect[i].JobID, logs[i].JobID)
					assert.Equal(t, tc.Expect[i].ChunkID, logs[i].ChunkID)
				}
			})
		}
	})

	t.Run("delete job", func(t *testing.T) {
		require.NoError(t, models.CleanupOldJobs(tx.Querier, time.Now()))
		for _, jobID := range []string{job1.ID, job2.ID} {
			logs, err := models.FindJobLogs(tx.Querier, models.JobLogsFilter{
				JobID: jobID,
			})
			assert.NoError(t, err)
			assert.Len(t, logs, 0)
		}
	})
}
