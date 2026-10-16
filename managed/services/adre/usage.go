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
	"cmp"
	"encoding/json"
	"strings"
)

// Holmes feature names stored in holmes_usage_events.feature.
const (
	HolmesFeatureAdreChat            = "adre_chat"
	HolmesFeatureInvestigationChat   = "investigation_chat"
	HolmesFeatureInvestigationRun    = "investigation_run"
	HolmesFeatureInvestigationFormat = "investigation_format"
	HolmesFeatureQanInsights         = "qan_insights"
	HolmesFeatureSlackChat           = "slack_chat"
)

// HolmesUsage is parsed token/cost data from Holmes metadata.
type HolmesUsage struct {
	Model            string
	PromptTokens     *int32
	CompletionTokens *int32
	TotalTokens      *int32
	CachedTokens     *int32
	TotalCost        *float64
	RawMetadata      json.RawMessage
}

// holmesTokenCounts is the token block shared by Holmes metadata "usage" (last LLM call) and "costs" (whole request).
type holmesTokenCounts struct {
	PromptTokens     *float64 `json:"prompt_tokens"`
	CompletionTokens *float64 `json:"completion_tokens"`
	TotalTokens      *float64 `json:"total_tokens"`
	CachedTokens     *float64 `json:"cached_tokens"`
}

func tokenCount(v *float64) *int32 {
	if v == nil {
		return nil
	}
	n := int32(*v) //nolint:gosec
	return &n
}

// ParseHolmesMetadata extracts usage and cost fields from Holmes /api/chat metadata.
// Token counts come from "costs", which sums every LLM call of the request; "usage" covers only the last call
// and is used when "costs" lacks a field.
func ParseHolmesMetadata(raw json.RawMessage) *HolmesUsage {
	if len(raw) == 0 {
		return nil
	}
	out := &HolmesUsage{RawMetadata: append(json.RawMessage(nil), raw...)}
	var meta struct {
		Usage *holmesTokenCounts `json:"usage"`
		Costs *struct {
			holmesTokenCounts
			TotalCost *float64 `json:"total_cost"`
		} `json:"costs"`
		Model string `json:"model"`
	}
	if json.Unmarshal(raw, &meta) != nil {
		return out
	}
	out.Model = meta.Model
	var last, all holmesTokenCounts
	if meta.Usage != nil {
		last = *meta.Usage
	}
	if meta.Costs != nil {
		all = meta.Costs.holmesTokenCounts
		out.TotalCost = meta.Costs.TotalCost
	}
	out.PromptTokens = tokenCount(cmp.Or(all.PromptTokens, last.PromptTokens))
	out.CompletionTokens = tokenCount(cmp.Or(all.CompletionTokens, last.CompletionTokens))
	out.TotalTokens = tokenCount(cmp.Or(all.TotalTokens, last.TotalTokens))
	out.CachedTokens = tokenCount(cmp.Or(all.CachedTokens, last.CachedTokens))
	return out
}

func extractUsageFromMetadata(raw json.RawMessage) (prompt, completion, total *int32) { //nolint:nonamedreturns
	u := ParseHolmesMetadata(raw)
	if u == nil {
		return nil, nil, nil
	}
	return u.PromptTokens, u.CompletionTokens, u.TotalTokens
}

// ResolveModelName returns req model or Holmes metadata model.
func ResolveModelName(requestModel string, usage *HolmesUsage) string {
	if s := trimModel(requestModel); s != "" {
		return s
	}
	if usage != nil {
		if s := trimModel(usage.Model); s != "" {
			return s
		}
	}
	return ""
}

func trimModel(s string) string {
	return strings.TrimSpace(s)
}

// HolmesUsageMap is the API representation of usage fields.
func HolmesUsageMap(u *HolmesUsage, model string) map[string]any {
	if u == nil {
		return nil
	}
	m := map[string]any{}
	if model != "" {
		m["model"] = model
	} else if u.Model != "" {
		m["model"] = u.Model
	}
	if u.PromptTokens != nil {
		m["prompt_tokens"] = *u.PromptTokens
	}
	if u.CompletionTokens != nil {
		m["completion_tokens"] = *u.CompletionTokens
	}
	if u.TotalTokens != nil {
		m["total_tokens"] = *u.TotalTokens
	}
	if u.CachedTokens != nil {
		m["cached_tokens"] = *u.CachedTokens
	}
	if u.TotalCost != nil {
		m["total_cost"] = *u.TotalCost
	}
	return m
}
