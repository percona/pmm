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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/utils/vault"
)

func TestSealingKeyHandler(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	agents, err := models.FindAgents(db.Querier, models.AgentFilters{PMMAgentID: models.PMMServerAgentID})
	require.NoError(t, err)
	require.NotEmpty(t, agents)
	exporterID := agents[0].AgentID

	r := newTestRegistry()
	r.db = db
	mux := http.NewServeMux()
	mux.Handle("GET /v1/inventory/agents/{agent_id}/sealing-key", r.SealingKeyHandler())
	get := func(t *testing.T, id string) (int, map[string]string) {
		t.Helper()
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/inventory/agents/"+id+"/sealing-key", nil))
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		return rec.Code, body
	}

	code, body := get(t, models.PMMServerAgentID)
	assert.Equal(t, http.StatusConflict, code)
	assert.Contains(t, body["message"], "is not connected")

	r.agents[models.PMMServerAgentID] = &pmmAgentInfo{id: models.PMMServerAgentID}
	code, body = get(t, models.PMMServerAgentID)
	assert.Equal(t, http.StatusConflict, code)
	assert.Contains(t, body["message"], "does not seal secrets, upgrade it")

	r.agents[models.PMMServerAgentID].sealingKey = "BASE64KEY"
	want := map[string]string{"pmm_agent_id": models.PMMServerAgentID, "public_key": "BASE64KEY", "algorithm": vault.SealingAlgorithm}
	code, body = get(t, models.PMMServerAgentID)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, want, body)

	// An Agent of the pmm-agent, for a change of it.
	assert.Equal(t, models.PMMServerAgentID, pointer.GetString(agents[0].PMMAgentID))
	code, body = get(t, exporterID)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, want, body)

	code, _ = get(t, "no-such-agent")
	assert.Equal(t, http.StatusNotFound, code)
}
