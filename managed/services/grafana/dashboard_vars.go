// Copyright (C) 2026 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// dashboardAPIEnvelope matches Grafana GET /api/dashboards/uid/:uid top-level JSON.
type dashboardAPIEnvelope struct {
	Dashboard dashboardInner `json:"dashboard"`
}

type dashboardInner struct {
	SchemaVersion int               `json:"schemaVersion"`
	Panels        []dashboardPanel  `json:"panels"`
	Templating    templatingWrapper `json:"templating"`
}

type templatingWrapper struct {
	List []templateVariable `json:"list"`
}

type templateVariable struct {
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Multi       bool            `json:"multi"`
	IncludeAll  bool            `json:"includeAll"`
	AllValue    string          `json:"allValue"`
	Current     json.RawMessage `json:"current"`
	Hide        int             `json:"hide"` // 2 = hidden
}

type dashboardPanel struct {
	ID       int              `json:"id"`
	Type     string           `json:"type"`
	Panels   []dashboardPanel `json:"panels"`
	Repeated json.RawMessage  `json:"repeat"` // presence indicates repeat row
}

func fetchDashboardEnvelope(ctx context.Context, client *Client, dashboardUID string, headers http.Header) (*dashboardAPIEnvelope, error) {
	path := "/graph/api/dashboards/uid/" + dashboardUID
	body, _, err := client.DoRaw(ctx, http.MethodGet, path, "", headers, nil)
	if err != nil {
		return nil, err
	}
	var env dashboardAPIEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, errors.Wrap(err, "decode dashboard JSON")
	}
	return &env, nil
}

func panelExistsInDashboard(d dashboardInner, panelID string) bool {
	return panelExistsWalk(d.Panels, panelID)
}

func panelExistsWalk(panels []dashboardPanel, panelID string) bool {
	for _, p := range panels {
		if strconv.Itoa(p.ID) == panelID {
			return true
		}
		if panelExistsWalk(p.Panels, panelID) {
			return true
		}
	}
	return false
}

func variableCurrentValue(raw json.RawMessage) string {
	var cur map[string]json.RawMessage
	if err := json.Unmarshal(raw, &cur); err != nil {
		return ""
	}
	valRaw, ok := cur["value"]
	if !ok || len(valRaw) == 0 {
		return ""
	}
	var v interface{}
	if err := json.Unmarshal(valRaw, &v); err != nil {
		return ""
	}
	return formatVarValue(v)
}

func formatVarValue(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case bool:
		return strconv.FormatBool(t)
	case []interface{}:
		parts := make([]string, 0, len(t))
		for _, x := range t {
			parts = append(parts, formatVarValue(x))
		}
		return strings.Join(parts, ",")
	default:
		return fmt.Sprint(t)
	}
}

// MergeDashboardVars merges dashboard saved defaults with POST overrides. Keys in overrides may be
// logical names (service_name) or already-prefixed var-service_name.
func MergeDashboardVars(d dashboardInner, overrides map[string]string) (map[string]string, error) {
	merged := make(map[string]string)
	validNames := make(map[string]struct{})
	for _, tv := range d.Templating.List {
		if tv.Name == "" {
			continue
		}
		validNames[tv.Name] = struct{}{}
		key := "var-" + tv.Name
		def := variableCurrentValue(tv.Current)
		if def == "" && tv.IncludeAll && tv.Multi {
			def = "$__all"
		}
		if def != "" {
			merged[key] = def
		}
	}

	resolveVarName := func(name string) (string, bool) {
		if _, ok := validNames[name]; ok {
			return name, true
		}
		for n := range validNames {
			if strings.EqualFold(n, name) {
				return n, true
			}
		}
		return "", false
	}

	for k, val := range overrides {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		name := k
		if len(k) >= 4 && strings.EqualFold(k[:4], "var-") {
			name = k[4:]
		}
		canonical, ok := resolveVarName(name)
		if !ok {
			valid := make([]string, 0, len(validNames))
			for n := range validNames {
				valid = append(valid, n)
			}
			sort.Strings(valid)
			return nil, fmt.Errorf("unknown override %q; valid template variables: %v", k, valid)
		}
		for _, tv := range d.Templating.List {
			if tv.Name == canonical && val == "$__all" && !(tv.IncludeAll && tv.Multi) {
				return nil, fmt.Errorf("override %q cannot use $__all for variable %q", k, canonical)
			}
		}
		merged["var-"+canonical] = val
	}

	return merged, nil
}

// buildGrafanaRenderQueryValues constructs Grafana Image Renderer query parameters for /render/d-solo/{uid}/ .
func buildGrafanaRenderQueryValues(panelID, from, to string, orgID, width, height, scale int, tz, theme string, schemaVersion int, mergedVars map[string]string) url.Values {
	renderParams := url.Values{}
	renderParams.Set("panelId", panelID)
	renderParams.Set("orgId", strconv.Itoa(orgID))
	renderParams.Set("from", from)
	renderParams.Set("to", to)
	renderParams.Set("width", strconv.Itoa(width))
	renderParams.Set("height", strconv.Itoa(height))
	renderParams.Set("scale", strconv.Itoa(scale))
	if tz == "" {
		tz = "browser"
	}
	renderParams.Set("tz", tz)
	if theme != "" {
		renderParams.Set("theme", theme)
	}
	if schemaVersion >= 39 {
		renderParams.Set("__feature.dashboardSceneSolo", "true")
		renderParams.Set("viewPanel", "panel-"+panelID)
	}
	for k, v := range mergedVars {
		if strings.HasPrefix(k, "var-") && v != "" {
			renderParams.Set(k, v)
		}
	}
	return renderParams
}
