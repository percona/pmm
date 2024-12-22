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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestJSON(t *testing.T) { //nolint:tparallel
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Run("Normal", func(t *testing.T) {
		t.Parallel()

		j1 := models.Job{
			ID:         "Normal",
			PMMAgentID: "test-agent",
			Data: &models.JobData{
				MySQLBackup: &models.MySQLBackupJobData{
					ServiceID:  "test_service",
					ArtifactID: "test_artifact",
				},
			},
		}
		err := db.Save(&j1)
		require.NoError(t, err)

		var j2 models.Job
		err = db.FindByPrimaryKeyTo(&j2, j1.ID)
		require.NoError(t, err)
		assert.Equal(t, j1, j2)
	})

	t.Run("Nil", func(t *testing.T) {
		t.Parallel()

		j1 := models.Job{
			PMMAgentID: "test-agent",
			ID:         "Nil",
		}
		err := db.Save(&j1)
		require.NoError(t, err)

		j2 := models.Job{
			Data: &models.JobData{
				MySQLBackup: &models.MySQLBackupJobData{
					ServiceID:  "test_service",
					ArtifactID: "test_artifact",
				},
			},
		}
		err = db.FindByPrimaryKeyTo(&j2, j1.ID)
		require.NoError(t, err)
		assert.Equal(t, j1, j2)
	})
}
