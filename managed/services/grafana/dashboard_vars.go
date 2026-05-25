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
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Multi      bool            `json:"multi"`
	IncludeAll bool            `json:"includeAll"`
	AllValue   string          `json:"allValue"`
	Current    json.RawMessage `json:"current"`
	Hide       int             `json:"hide"` // 2 = hidden
}

type dashboardPanel struct {
	ID          int              `json:"id"`
	Type        string           `json:"type"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Targets     []panelTarget    `json:"targets"`
	Panels      []dashboardPanel `json:"panels"`
	Repeated    json.RawMessage  `json:"repeat"` // presence indicates repeat row
}

type panelTarget struct {
	Expr         string `json:"expr"`
	LegendFormat string `json:"legendFormat"`
	RefID        string `json:"refId"`
}

func fetchDashboardEnvelope(ctx context.Context, client *Client, dashboardUID string, headers http.Header) (*dashboardAPIEnvelope, error) {
	path := "/graph/api/dashboards/uid/" + dashboardUID
	body, _, err := client.DoRaw(ctx, http.MethodGet, path, "", headers, nil)
	if err != nil {
		return nil, err
	}
	var env dashboardAPIEnvelope
	err = json.Unmarshal(body, &env)
	if err != nil {
		return nil, fmt.Errorf("decode dashboard JSON: %w", err)
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
	err := json.Unmarshal(raw, &cur)
	if err != nil {
		return ""
	}
	valRaw, ok := cur["value"]
	if !ok || len(valRaw) == 0 {
		return ""
	}
	var v any
	err = json.Unmarshal(valRaw, &v)
	if err != nil {
		return ""
	}
	return formatVarValue(v)
}

func formatVarValue(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case bool:
		return strconv.FormatBool(t)
	case []any:
		parts := make([]string, 0, len(t))
		for _, x := range t {
			parts = append(parts, formatVarValue(x))
		}
		return strings.Join(parts, ",")
	default:
		return fmt.Sprint(t)
	}
}

// stripPathStyleVar removes Grafana/PMM path-style prefixes from template current values.
// Dashboard JSON often stores e.g. "/service_id/<uuid>"; d-solo image render expects the bare UUID.
func stripPathStyleVar(value, asciiPrefix string) string {
	if len(value) < len(asciiPrefix) {
		return value
	}
	if !strings.EqualFold(value[:len(asciiPrefix)], asciiPrefix) {
		return value
	}
	return strings.TrimSpace(value[len(asciiPrefix):])
}

func sanitizeTemplateValue(varName, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.EqualFold(varName, "interval") && strings.Contains(value, "$__auto_interval") {
		return "$__auto"
	}
	if strings.EqualFold(varName, "agent_id") && strings.HasPrefix(value, "/agent_id/") {
		return ""
	}
	if strings.EqualFold(varName, "node_id") {
		value = stripPathStyleVar(value, "/node_id/")
	}
	if strings.EqualFold(varName, "service_id") {
		value = stripPathStyleVar(value, "/service_id/")
	}
	if value == "" {
		return ""
	}
	return value
}

// MergeDashboardVars merges dashboard saved defaults with POST overrides. Keys in overrides may be
// logical names (service_name) or already-prefixed var-service_name.
//
// An override whose value is only whitespace, or the empty string, clears the saved dashboard
// default for that variable by forcing an explicit empty var-* in the final query (var-x=),
// matching Grafana UI share links for blank cluster/region fields. When the raw value is non-empty
// but sanitizeTemplateValue returns empty (e.g. invalid agent_id), the default is left unchanged.
func MergeDashboardVars(d dashboardInner, overrides map[string]string) (map[string]string, error) { //nolint:gocognit
	merged := make(map[string]string)
	validNames := make(map[string]struct{})
	defByName := make(map[string]templateVariable)
	for _, tv := range d.Templating.List {
		if tv.Name == "" {
			continue
		}
		validNames[tv.Name] = struct{}{}
		defByName[tv.Name] = tv
		key := "var-" + tv.Name
		def := variableCurrentValue(tv.Current)
		if def == "" && tv.IncludeAll && tv.Multi {
			def = "$__all"
		}
		def = sanitizeTemplateValue(tv.Name, def)
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
		raw := val
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
		tv := defByName[canonical]
		if val == "$__all" && (!tv.IncludeAll || !tv.Multi) {
			return nil, fmt.Errorf("override %q cannot use $__all for variable %q", k, canonical)
		}
		val = sanitizeTemplateValue(canonical, val)
		if val == "" {
			if strings.TrimSpace(raw) == "" {
				merged["var-"+canonical] = ""
			}
			continue
		}
		merged["var-"+canonical] = val
	}

	return merged, nil
}

// buildGrafanaRenderQueryValues constructs Grafana Image Renderer query parameters for /render/d-solo/{uid}/ .
//
// Query shape matches what the Grafana UI issues for “Direct link rendered image” on current Grafana (PMM 3):
// panelId=panel-<id>, __feature.dashboardScene=true, hideLogo=true. Relying on dashboard JSON schemaVersion
// to choose legacy numeric panelId breaks against live Grafana: bundled dashboards can stay schemaVersion 34
// while the server runs Scenes, which then mis-handles d-solo and can stall until the image-renderer times out.
//
// The UI passes timezone=browser (dashboard time mode) while tz= is the IANA zone for the panel; setting both
// to the same string breaks that split and can change panel readiness/render behavior.
func buildGrafanaRenderQueryValues(panelID, from, to string, orgID, width, height, scale int, tz, theme string, mergedVars map[string]string) url.Values {
	renderParams := url.Values{}
	pid := NormalizePanelID(panelID)
	scenePanel := "panel-" + pid
	renderParams.Set("panelId", scenePanel)
	renderParams.Set("__feature.dashboardScene", "true")
	renderParams.Set("hideLogo", "true")
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
	renderParams.Set("timezone", "browser")
	// Match Grafana "Direct link rendered image" (HAR): refresh on the d-solo query, not a substitute for a wrong URL.
	renderParams.Set("refresh", "1m")
	if theme != "" {
		renderParams.Set("theme", theme)
	}
	for k, v := range mergedVars {
		if strings.HasPrefix(k, "var-") {
			renderParams.Set(k, v)
		}
	}
	return renderParams
}
