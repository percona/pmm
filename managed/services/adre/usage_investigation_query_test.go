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

package adre

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestQueryInvestigationUsageEvents(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() { require.NoError(t, sqlDB.Close()) }()
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	now := time.Now().UTC().Truncate(time.Millisecond)
	invID := "inv-usage-query-test"
	otherInvID := "inv-usage-query-other"
	msgID := "msg-usage-query-1"

	insertInvestigation := func(id, title string) {
		t.Helper()
		require.NoError(t, db.Insert(&models.Investigation{
			ID:        id,
			Title:     title,
			Status:    "open",
			Severity:  "low",
			CreatedAt: now,
			UpdatedAt: now,
			TimeFrom:  now,
			TimeTo:    now,
		}))
	}
	insertUsage := func(feature, investigationID, featureRef string, tokens int32, at time.Time) {
		t.Helper()
		total := tokens
		require.NoError(t, db.Insert(&models.HolmesUsageEvent{
			CreatedAt:       at,
			Feature:         feature,
			FeatureRef:      featureRef,
			InvestigationID: investigationID,
			Model:           "gpt-4.1",
			TotalTokens:     &total,
			MetadataJSON:    []byte("{}"),
		}))
	}

	insertInvestigation(invID, "Target investigation")
	insertInvestigation(otherInvID, "Other investigation")
	require.NoError(t, db.Insert(&models.InvestigationMessage{
		ID:              msgID,
		InvestigationID: invID,
		Role:            "assistant",
		Content:         "assistant reply",
		CreatedAt:       now,
	}))

	insertUsage("run", invID, "", 100, now)
	insertUsage("format-report", "", invID, 200, now.Add(time.Second))
	insertUsage("chat", "", msgID, 300, now.Add(2*time.Second))
	insertUsage("chat", otherInvID, "", 999, now)

	events, err := QueryInvestigationUsageEvents(db, invID)
	require.NoError(t, err)
	require.Len(t, events, 3)

	assert.Equal(t, "run", events[0].Feature)
	require.NotNil(t, events[0].TotalTokens)
	assert.Equal(t, int32(100), *events[0].TotalTokens)

	assert.Equal(t, "format-report", events[1].Feature)
	require.NotNil(t, events[1].TotalTokens)
	assert.Equal(t, int32(200), *events[1].TotalTokens)

	assert.Equal(t, "chat", events[2].Feature)
	require.NotNil(t, events[2].TotalTokens)
	assert.Equal(t, int32(300), *events[2].TotalTokens)
}
