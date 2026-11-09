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

package agents

import (
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	backuppb "github.com/percona/pmm/api/backup/v1"
	"github.com/percona/pmm/managed/models"
)

func rowValues(t *testing.T, r reform.Record) []driver.Value {
	t.Helper()
	values := make([]driver.Value, 0, len(r.Values()))
	for _, v := range r.Values() {
		dv, err := driver.DefaultParameterConverter.ConvertValue(v)
		require.NoError(t, err)
		values = append(values, dv)
	}
	return values
}

func TestHandleJobResultMongoDBBackupFolderUpdateError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = mock.ExpectClose()
		assert.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	job := &models.Job{
		ID:   "job-id",
		Type: models.MongoDBBackupJob,
		Data: &models.JobData{MongoDBBackup: &models.MongoDBBackupJobData{ArtifactID: "artifact-id"}},
	}
	artifact := &models.Artifact{
		ID:   "artifact-id",
		Name: "artifact-name",
		Mode: models.PITR,
		Type: models.OnDemandArtifactType,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT .+ FROM "jobs" WHERE .+ LIMIT 1`).
		WithArgs(job.ID).
		WillReturnRows(sqlmock.NewRows(models.JobTable.Columns()).AddRow(rowValues(t, job)...))
	mock.ExpectQuery(`SELECT .+ FROM "artifacts" WHERE .+ LIMIT 1`).
		WithArgs(artifact.ID).
		WillReturnRows(sqlmock.NewRows(models.ArtifactTable.Columns()).AddRow(rowValues(t, artifact)...))
	mock.ExpectExec(`UPDATE "artifacts"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .+ FROM "artifacts" WHERE .+ LIMIT 1`).
		WithArgs(artifact.ID).
		WillReturnError(errors.New("connection reset"))
	mock.ExpectRollback()

	s := NewJobsService(db, nil, nil)
	s.handleJobResult(t.Context(), logrus.WithField("test", t.Name()), &agentv1.JobResult{
		JobId:  job.ID,
		Result: &agentv1.JobResult_MongodbBackup{MongodbBackup: &agentv1.JobResult_MongoDBBackup{}},
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestArtifactMetadataFromProto(t *testing.T) {
	t.Run("all fields are filled", func(t *testing.T) {
		protoMetadata := backuppb.Metadata{
			FileList:           []*backuppb.File{{Name: "dir1", IsDirectory: true}, {Name: "file1"}, {Name: "file2"}},
			RestoreTo:          &timestamppb.Timestamp{Seconds: 123, Nanos: 456},
			BackupToolMetadata: &backuppb.Metadata_PbmMetadata{PbmMetadata: &backuppb.PbmMetadata{Name: "some name"}},
		}

		expected := &models.Metadata{
			FileList:       []models.File{{Name: "dir1", IsDirectory: true}, {Name: "file1"}, {Name: "file2"}},
			RestoreTo:      new(time.Unix(123, 456).UTC()),
			BackupToolData: &models.BackupToolData{PbmMetadata: &models.PbmMetadata{Name: "some name"}},
		}

		actual := artifactMetadataFromProto(&protoMetadata)
		assert.Equal(t, expected, actual)
	})

	t.Run("some fields are empty", func(t *testing.T) {
		protoMetadata := backuppb.Metadata{
			FileList: []*backuppb.File{{Name: "dir1", IsDirectory: true}, {Name: "file1"}, {Name: "file2"}},
		}

		expected := &models.Metadata{
			FileList: []models.File{{Name: "dir1", IsDirectory: true}, {Name: "file1"}, {Name: "file2"}},
		}

		actual := artifactMetadataFromProto(&protoMetadata)
		assert.Equal(t, expected, actual)
	})

	t.Run("argument is nil", func(t *testing.T) {
		var protoMetadata *backuppb.Metadata
		var expected *models.Metadata

		actual := artifactMetadataFromProto(protoMetadata)
		assert.Equal(t, expected, actual)
	})
}
