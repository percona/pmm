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
	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestPMMAgentSupported(t *testing.T) {
	t.Parallel()
	prefix := "testing prefix"
	minVersion := version.Must(version.NewVersion("2.30.5"))

	tests := []struct {
		name         string
		agentVersion string
		errString    string
	}{
		{
			name:         "Empty version string",
			agentVersion: "",
			errString:    "failed to parse PMM agent version",
		},
		{
			name:         "Wrong version string",
			agentVersion: "Some version",
			errString:    "failed to parse PMM agent version",
		},
		{
			name:         "Less than min version",
			agentVersion: "2.30.4",
			errString:    "not supported by pmm-agent",
		},
		{
			name:         "Equals min version",
			agentVersion: "2.30.5",
			errString:    "",
		},
		{
			name:         "Greater than min version",
			agentVersion: "2.30.6",
			errString:    "",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			agentModel := models.Agent{
				AgentID: "Test agent ID",
				Version: pointer.ToString(test.agentVersion),
			}
			err := models.IsAgentSupported(&agentModel, prefix, minVersion)
			if test.errString == "" {
				assert.NoError(t, err)
			} else {
				assert.Contains(t, err.Error(), test.errString)
			}
		})
	}

	t.Run("No version info", func(t *testing.T) {
		err := models.IsAgentSupported(&models.Agent{AgentID: "Test agent ID"}, prefix, version.Must(version.NewVersion("2.30.0")))
		assert.Contains(t, err.Error(), "has no version info")
	})

	t.Run("Nil agent", func(t *testing.T) {
		err := models.IsAgentSupported(nil, prefix, version.Must(version.NewVersion("2.30.0")))
		assert.Contains(t, err.Error(), "nil agent")
	})
}

func TestIsPostgreSQLSSLSniSupported(t *testing.T) {
	now, origNowF := models.Now(), models.Now
	models.Now = func() time.Time {
		return now
	}
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	defer func() {
		models.Now = origNowF
		require.NoError(t, sqlDB.Close())
	}()

	setup := func(t *testing.T) (q *reform.Querier, teardown func(t *testing.T)) {
		t.Helper()
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		q = tx.Querier

		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:   "N1",
				NodeType: models.GenericNodeType,
				NodeName: "Generic Node",
			},

			&models.Agent{
				AgentID:      "New",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("N1"),
				Version:      pointer.ToString("2.41.0"),
			},

			&models.Agent{
				AgentID:      "Old",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("N1"),
				Version:      pointer.ToString("2.40.1"),
			},
		} {
			require.NoError(t, q.Insert(str), "failed to INSERT %+v", str)
		}

		teardown = func(t *testing.T) {
			t.Helper()
			require.NoError(t, tx.Rollback())
		}
		return
	}
	q, teardown := setup(t)
	defer teardown(t)

	tests := []struct {
		pmmAgentID string
		expected   bool
	}{
		{
			"New",
			true,
		},
		{
			"Old",
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.pmmAgentID, func(t *testing.T) {
			actual, err := models.IsPostgreSQLSSLSniSupported(q, tt.pmmAgentID)
			assert.Equal(t, tt.expected, actual)
			assert.NoError(t, err)
		})
	}

	t.Run("Non-existing ID", func(t *testing.T) {
		_, err := models.IsPostgreSQLSSLSniSupported(q, "Not exist")
		assert.Error(t, err)
	})
}
