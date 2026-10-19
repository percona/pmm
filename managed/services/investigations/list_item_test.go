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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/managed/models"
)

func TestNewListItem(t *testing.T) {
	at := time.Date(2026, 10, 16, 13, 0, 0, 0, time.UTC)
	inv := &models.Investigation{
		ID: "inv1", Title: "Alert: replication stopped", Status: "completed", Severity: "critical",
		CreatedBy: models.AutoInvestigateCreatedBy, CreatedAt: at, UpdatedAt: at, TimeFrom: at, TimeTo: at,
		SourceType: "alert", Summary: "Replication stopped.", RootCauseSummary: "A script writes to the replica.",
		Config: []byte(`{"node_name":"ps-1","service_name":"ps-1-mysql"}`),
	}

	item := newListItem(inv)
	assert.Equal(t, "A script writes to the replica.", item.Summary)
	assert.Equal(t, "critical", item.Severity)
	assert.Equal(t, models.AutoInvestigateCreatedBy, item.CreatedBy)
	assert.Equal(t, "2026-10-16T13:00:00Z", item.CreatedAt)
	assert.Equal(t, "ps-1", item.NodeName)
	assert.Equal(t, "ps-1-mysql", item.ServiceName)

	inv.RootCauseSummary = ""
	assert.Equal(t, "Replication stopped.", newListItem(inv).Summary)

	inv.Severity = ""
	inv.Config = []byte(`{"alert_snapshot":"[{\"labels\":{\"severity\":\"warning\"}}]"}`)
	assert.Equal(t, "warning", newListItem(inv).Severity)
}
