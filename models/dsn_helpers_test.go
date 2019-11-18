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

package models_test

import (
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
)

func TestFindDSNByServiceID(t *testing.T) {
	now, origNowF := models.Now(), models.Now
	models.Now = func() time.Time {
		return now
	}
	sqlDB := testdb.Open(t, models.SkipFixtures)
	defer func() {
		models.Now = origNowF
		require.NoError(t, sqlDB.Close())
	}()

	setup := func(t *testing.T) (q *reform.Querier, teardown func(t *testing.T)) {
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		q = tx.Querier

		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:   "N1",
				NodeType: models.GenericNodeType,
				NodeName: "Node with Service",
			},

			&models.Service{
				ServiceID:   "S1",
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service on N1",
				NodeID:      "N1",
			},

			&models.Agent{
				AgentID:      "PA1",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("N1"),
			},

			&models.Service{
				ServiceID:   "S2",
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service-2 on N1",
				NodeID:      "N1",
				Address:     pointer.ToString("127.0.0.1"),
			},
			&models.Agent{
				AgentID:      "A1",
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   pointer.ToString("PA1"),
				RunsOnNodeID: nil,
				ServiceID:    pointer.ToString("S1"),
			},
			&models.Agent{
				AgentID:      "A2",
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   pointer.ToString("PA1"),
				RunsOnNodeID: nil,
				Username:     pointer.ToString("pmm-user"),
				ServiceID:    pointer.ToString("S2"),
			},
		} {
			require.NoError(t, q.Insert(str))
		}

		teardown = func(t *testing.T) {
			require.NoError(t, tx.Rollback())
		}
		return
	}

	t.Run("FindDSNByServiceIDandPMMAgentID", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		dsn, err := models.FindDSNByServiceIDandPMMAgentID(q, "S2", "PA1", "test")
		require.NoError(t, err)
		expected := "pmm-user@tcp(127.0.0.1:0)/test?timeout=1s"
		assert.Equal(t, expected, dsn)
	})
}
