// Copyright (C) 2024 Percona LLC
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
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestJSON(t *testing.T) { //nolint:tparallel
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Run("Normal", func(t *testing.T) {
		t.Parallel()

		c1 := models.Channel{
			ID: "Normal",
			EmailConfig: &models.EmailConfig{
				To: []string{"foo@bar.test"},
			},
		}
		err := db.Save(&c1)
		require.NoError(t, err)

		var c2 models.Channel
		err = db.FindByPrimaryKeyTo(&c2, c1.ID)
		require.NoError(t, err)
		assert.Equal(t, c1, c2)
	})

	t.Run("Nil", func(t *testing.T) {
		t.Parallel()

		c1 := models.Channel{
			ID: "Nil",
		}
		err := db.Save(&c1)
		require.NoError(t, err)

		c2 := models.Channel{
			EmailConfig: &models.EmailConfig{
				To: []string{"foo@bar.test"},
			},
		}
		err = db.FindByPrimaryKeyTo(&c2, c1.ID)
		require.NoError(t, err)
		assert.Equal(t, c1, c2)
	})
}
