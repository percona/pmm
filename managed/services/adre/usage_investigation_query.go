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
	"time"

	"gopkg.in/reform.v1"
)

const investigationUsageEventsSQL = `
		SELECT id, created_at, feature, model, total_tokens, cached_tokens, total_cost, latency_ms
		FROM holmes_usage_events
		WHERE investigation_id = $1
		   OR feature_ref = $1
		   OR (
		        feature_ref <> '' AND feature_ref IN (
		            SELECT id FROM investigation_messages WHERE investigation_id = $1
		        )
		   )
		ORDER BY created_at ASC`

// InvestigationUsageEvent is one Holmes usage row linked to an investigation.
type InvestigationUsageEvent struct {
	ID           int64
	CreatedAt    time.Time
	Feature      string
	Model        string
	TotalTokens  *int32
	CachedTokens *int32
	TotalCost    *float64
	LatencyMs    *int32
}

// QueryInvestigationUsageEvents returns usage rows for an investigation, including rows
// linked only via feature_ref (e.g. format-report calls keyed by investigation id).
func QueryInvestigationUsageEvents(db *reform.DB, investigationID string) ([]InvestigationUsageEvent, error) {
	rows, err := db.Query(investigationUsageEventsSQL, investigationID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []InvestigationUsageEvent
	for rows.Next() {
		var ev InvestigationUsageEvent
		err := rows.Scan(
			&ev.ID, &ev.CreatedAt, &ev.Feature, &ev.Model,
			&ev.TotalTokens, &ev.CachedTokens, &ev.TotalCost, &ev.LatencyMs,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil { //nolint:noinlineerr
		return nil, err
	}
	return out, nil
}
