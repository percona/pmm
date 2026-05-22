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

package models

import (
	"encoding/json"
	"strings"
)

const (
	otelLogSourcesLabel   = "log_sources"
	otelLogFilePathsLabel = "log_file_paths"
	otelPresetRaw         = "raw"
)

// OtelLogSourceEntry mirrors agents.logSourceEntry and the JSON in custom_labels["log_sources"].
type OtelLogSourceEntry struct {
	Path   string `json:"path"`
	Preset string `json:"preset"`
}

// ParseOtelLogSourcesFromLabels returns path+preset pairs from OTEL collector custom_labels.
// It matches the logic in managed/services/agents/otelcollector.go getLogSourcesFromAgent.
func ParseOtelLogSourcesFromLabels(labels map[string]string) ([]OtelLogSourceEntry, error) {
	if labels == nil {
		return nil, nil
	}
	if s := labels[otelLogSourcesLabel]; s != "" {
		var entries []OtelLogSourceEntry
		err := json.Unmarshal([]byte(s), &entries)
		if err != nil {
			return nil, err
		}
		return entries, nil
	}
	if s := labels[otelLogFilePathsLabel]; s != "" {
		var entries []OtelLogSourceEntry
		for p := range strings.SplitSeq(s, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				entries = append(entries, OtelLogSourceEntry{Path: p, Preset: otelPresetRaw})
			}
		}
		return entries, nil
	}
	return nil, nil
}
