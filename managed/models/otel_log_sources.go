// Copyright (C) 2026 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
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
		if err := json.Unmarshal([]byte(s), &entries); err != nil {
			return nil, errors.WithStack(err)
		}
		return entries, nil
	}
	if s := labels[otelLogFilePathsLabel]; s != "" {
		var entries []OtelLogSourceEntry
		for _, p := range strings.Split(s, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				entries = append(entries, OtelLogSourceEntry{Path: p, Preset: otelPresetRaw})
			}
		}
		return entries, nil
	}
	return nil, nil
}
