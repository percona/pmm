// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package investigations

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func alertInvestigation(t *testing.T, id, alertname, service, node string, at time.Time, rootCause string) *models.Investigation {
	t.Helper()
	snapshot, err := json.Marshal([]map[string]any{{"labels": map[string]string{
		"alertname": alertname, "service_name": service, "node_name": node,
	}}})
	require.NoError(t, err)
	cfg, err := json.Marshal(map[string]string{"alert_snapshot": string(snapshot), "node_name": node})
	require.NoError(t, err)
	return &models.Investigation{
		ID: id, Title: "Alert: " + alertname, Status: "completed", CreatedAt: at, Config: cfg,
		RootCauseSummary: rootCause,
	}
}

func TestSimilarInvestigations(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	current := alertInvestigation(t, "now", "MySQL replication SQL thread stopped", "ps-8.4-2", "ps-node-2", now,
		"Duplicate key error on the replica after writes to both source and replica.")
	past := []*models.Investigation{
		current,
		alertInvestigation(t, "same-alert", "MySQL replication SQL thread stopped", "ps-8.4-2", "ps-node-2", now.Add(-72*time.Hour),
			"The replica is writable; a script wrote to both source and replica, causing a duplicate key error."),
		alertInvestigation(t, "same-node", "PMM agent down", "pmm-agent", "ps-node-2", now.Add(-24*time.Hour), "The agent was restarted."),
		alertInvestigation(t, "unrelated", "MongoDB primary changed", "rs101", "rs-node-1", now.Add(-time.Hour), "Manual stepdown."),
		alertInvestigation(t, "later", "MySQL replication SQL thread stopped", "ps-8.4-2", "ps-node-2", now.Add(time.Hour), "Later."),
		{ID: "running", Title: "Alert: MySQL replication SQL thread stopped", Status: "running", CreatedAt: now.Add(-time.Minute)},
	}

	got := similarInvestigations(incidentOf(current), past, current.ID, now)

	require.Len(t, got, 2, "unrelated, running, later and the investigation itself are left out")
	assert.Equal(t, "same-alert", got[0].ID)
	assert.Equal(t, []string{"Same alert", "Same service", "Same node", "Similar description"}, got[0].Reasons)
	assert.Equal(t, "same-node", got[1].ID)
	assert.Equal(t, []string{"Same node"}, got[1].Reasons)
	assert.Greater(t, got[0].Score, got[1].Score)
}

func TestIncidentFromQuery(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	past := []*models.Investigation{
		alertInvestigation(t, "pg", "PostgreSQL high transaction rollbacks", "pdpgsql_17", "pg-node", now.Add(-time.Hour),
			"A burst of pgbench sessions rolled back on deadlocks."),
	}

	byAlert := similarInvestigations(incident("PostgreSQL high transaction rollbacks", "", "", "", ""), past, "", now)
	require.Len(t, byAlert, 1)
	assert.Equal(t, []string{"Same alert"}, byAlert[0].Reasons)

	byText := similarInvestigations(incident("", "", "", "", "Why so many deadlocks and rollbacks on pgbench?"), past, "", now)
	require.Len(t, byText, 1)
	assert.Equal(t, []string{"Similar description"}, byText[0].Reasons)

	assert.Empty(t, similarInvestigations(incident("", "", "", "", "disk full on mongo"), past, "", now))
}

func TestSimilarContext(t *testing.T) {
	assert.Empty(t, similarContext(nil))

	at := time.Date(2026, 10, 16, 13, 1, 0, 0, time.UTC)
	text := similarContext([]SimilarInvestigation{
		{
			Title: "Alert: A", CreatedAt: at, Status: "completed", Reasons: []string{"Same alert"}, RootCauseSummary: "Writes on the replica.",
			ResolutionSummary: "Set read_only=1.",
		},
		{Title: "Alert: B", CreatedAt: at, Status: "open", Reasons: []string{"Same node"}},
		{Title: "Alert: C"},
		{Title: "Alert: D"},
	})

	assert.Contains(t, text, "- Alert: A (2026-10-16, completed; Same alert)\n  Root cause: Writes on the replica.\n  Resolution: Set read_only=1.\n")
	assert.Contains(t, text, "- Alert: B (2026-10-16, open; Same node)\n")
	assert.Contains(t, text, "Alert: C")
	assert.NotContains(t, text, "Alert: D", "the three closest are enough context")
}

func TestSimilarDescriptionAgainstTheShorterText(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	report := alertInvestigation(t, "old", "MySQL replication SQL thread stopped", "ps-8.4-2", "ps-node-2", now.Add(-72*time.Hour),
		"The replica is writable (read_only=0). A client or script writes to both the source and the replica. It inserted id 900001 "+
			"into shop.customers directly on the replica in the same minute as on the source, so the replicated Write_rows event "+
			"failed with duplicate key error 1062.")

	got := similarInvestigations(incident("", "", "", "", "MySQL replication SQL thread stopped"), []*models.Investigation{report}, "", now)

	require.Len(t, got, 1, "a one-line title still matches the long report it names")
	assert.Equal(t, []string{"Similar description"}, got[0].Reasons)
	assert.Empty(t, similarInvestigations(incident("", "", "", "", "MySQL replica lag"), []*models.Investigation{report}, "", now),
		"two shared words are not enough")
}
