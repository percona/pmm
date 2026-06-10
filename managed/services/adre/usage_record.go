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
	"context"
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// UsageRecordInput describes one Holmes /api/chat completion to persist.
type UsageRecordInput struct {
	DB                     *reform.DB
	Feature                string
	FeatureRef             string
	AdreConversationID     *int64
	InvestigationID        string
	Model                  string
	Metadata               json.RawMessage
	TriggeredBy            string
	Stream                 bool
	LatencyMs              int
	AdreMessageID          *int64
	InvestigationMessageID *string
	QanQueryID             string
	QanServiceID           string
}

// RecordHolmesUsage inserts a holmes_usage_events row and updates linked entities.
func RecordHolmesUsage(ctx context.Context, in UsageRecordInput) (int64, error) { //nolint:gocognit,unparam
	if in.DB == nil {
		return 0, nil
	}
	usage := ParseHolmesMetadata(in.Metadata)
	model := ResolveModelName(in.Model, usage)
	if usage == nil && model == "" && len(in.Metadata) == 0 {
		return 0, nil
	}

	metaJSON := []byte("{}")
	if usage != nil && len(usage.RawMetadata) > 0 {
		metaJSON = usage.RawMetadata
	} else if len(in.Metadata) > 0 {
		metaJSON = in.Metadata
	}

	var eventID int64
	err := in.DB.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		now := time.Now().UTC()
		ev := &models.HolmesUsageEvent{
			CreatedAt:       now,
			Feature:         in.Feature,
			FeatureRef:      in.FeatureRef,
			InvestigationID: in.InvestigationID,
			Model:           model,
			TriggeredBy:     in.TriggeredBy,
			Stream:          in.Stream,
			MetadataJSON:    metaJSON,
		}
		if in.AdreConversationID != nil {
			ev.AdreConversationID = in.AdreConversationID
		}
		if usage != nil {
			ev.PromptTokens = usage.PromptTokens
			ev.CompletionTokens = usage.CompletionTokens
			ev.TotalTokens = usage.TotalTokens
			ev.CachedTokens = usage.CachedTokens
			ev.TotalCost = usage.TotalCost
			ev.CostPrompt = usage.CostPrompt
			ev.CostCompletion = usage.CostCompletion
			ev.CostCached = usage.CostCached
		}
		if in.LatencyMs > 0 {
			ms := int32(in.LatencyMs) //nolint:gosec
			ev.LatencyMs = &ms
		}
		err := tx.Save(ev)
		if err != nil {
			return err
		}
		eventID = ev.ID

		if in.AdreMessageID != nil && usage != nil {
			_, err := tx.Exec(
				`UPDATE adre_messages SET cached_tokens = $1, total_cost = $2, usage_event_id = $3,
				 prompt_tokens = COALESCE($4, prompt_tokens), completion_tokens = COALESCE($5, completion_tokens),
				 total_tokens = COALESCE($6, total_tokens), model = CASE WHEN $7 <> '' THEN $7 ELSE model END
				 WHERE id = $8`,
				usage.CachedTokens, usage.TotalCost, eventID,
				usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, model, *in.AdreMessageID,
			)
			if err != nil {
				return err
			}
		}

		if in.InvestigationMessageID != nil && usage != nil {
			_, err := tx.Exec(
				`UPDATE investigation_messages SET model = $1, prompt_tokens = $2, completion_tokens = $3,
				 total_tokens = $4, cached_tokens = $5, total_cost = $6, usage_event_id = $7, holmes_feature = $8
				 WHERE id = $9`,
				model, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens,
				usage.CachedTokens, usage.TotalCost, eventID, in.Feature, *in.InvestigationMessageID,
			)
			if err != nil {
				return err
			}
		}

		if in.QanQueryID != "" && in.QanServiceID != "" && usage != nil {
			_, err := tx.Exec(
				`UPDATE qan_insights_cache SET model = $1, prompt_tokens = $2, completion_tokens = $3,
				 total_tokens = $4, cached_tokens = $5, total_cost = $6, usage_event_id = $7
				 WHERE query_id = $8 AND service_id = $9`,
				model, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens,
				usage.CachedTokens, usage.TotalCost, eventID, in.QanQueryID, in.QanServiceID,
			)
			if err != nil {
				return err
			}
		}

		if in.InvestigationID != "" && usage != nil {
			var addTokens int64
			if usage.TotalTokens != nil {
				addTokens = int64(*usage.TotalTokens)
			}
			var addCost float64
			if usage.TotalCost != nil {
				addCost = *usage.TotalCost
			}
			_, err := tx.Exec(
				`UPDATE investigations SET holmes_total_tokens = holmes_total_tokens + $1,
				 holmes_total_cost = holmes_total_cost + $2, holmes_call_count = holmes_call_count + 1
				 WHERE id = $3`,
				addTokens, addCost, in.InvestigationID,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		logrus.WithField("component", "adre").Warnf("RecordHolmesUsage: %v", err)
		return 0, err
	}
	if usage != nil {
		observeHolmesUsage(in.Feature, usage)
	}
	return eventID, nil
}

// ApplyHolmesUsageToAdreMessage sets token/cost fields on an AdreMessage before insert.
func ApplyHolmesUsageToAdreMessage(m *models.AdreMessage, requestModel string, metadata json.RawMessage) {
	if m == nil {
		return
	}
	usage := ParseHolmesMetadata(metadata)
	if usage == nil {
		return
	}
	if model := ResolveModelName(requestModel, usage); model != "" {
		m.Model = model
	}
	m.PromptTokens = usage.PromptTokens
	m.CompletionTokens = usage.CompletionTokens
	m.TotalTokens = usage.TotalTokens
	m.CachedTokens = usage.CachedTokens
	m.TotalCost = usage.TotalCost
}

// ApplyHolmesUsageToInvestigationMessage sets usage fields on an investigation assistant message.
func ApplyHolmesUsageToInvestigationMessage(m *models.InvestigationMessage, feature, requestModel string, metadata json.RawMessage) {
	if m == nil {
		return
	}
	usage := ParseHolmesMetadata(metadata)
	if usage == nil {
		return
	}
	m.HolmesFeature = feature
	if model := ResolveModelName(requestModel, usage); model != "" {
		m.Model = model
	}
	m.PromptTokens = usage.PromptTokens
	m.CompletionTokens = usage.CompletionTokens
	m.TotalTokens = usage.TotalTokens
	m.CachedTokens = usage.CachedTokens
	m.TotalCost = usage.TotalCost
}
