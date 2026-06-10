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
	"strings"
)

// PanelQuery is a compact PromQL target extracted from a Grafana dashboard panel.
type PanelQuery struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Type         string `json:"type"`
	Description  string `json:"description,omitempty"`
	Expr         string `json:"expr,omitempty"`
	LegendFormat string `json:"legend_format,omitempty"`
}

func extractPanelQueries(d dashboardInner, wantIDs map[int]struct{}) []PanelQuery {
	out := make([]PanelQuery, 0, len(wantIDs))
	walkPanelQueries(d.Panels, wantIDs, &out)
	return out
}

func walkPanelQueries(panels []dashboardPanel, wantIDs map[int]struct{}, out *[]PanelQuery) {
	for _, p := range panels {
		if p.Type == "row" {
			walkPanelQueries(p.Panels, wantIDs, out)
			continue
		}
		if p.ID == 0 {
			continue
		}
		if wantIDs != nil {
			if _, ok := wantIDs[p.ID]; !ok {
				continue
			}
		}
		expr, legend := mergePanelTargets(p.Targets)
		if expr == "" && wantIDs == nil {
			continue
		}
		*out = append(*out, PanelQuery{
			ID:           p.ID,
			Title:        strings.TrimSpace(p.Title),
			Type:         p.Type,
			Description:  strings.TrimSpace(p.Description),
			Expr:         expr,
			LegendFormat: legend,
		})
	}
}

func mergePanelTargets(targets []panelTarget) (expr, legend string) { //nolint:nonamedreturns
	for _, t := range targets {
		e := strings.TrimSpace(t.Expr)
		if e == "" {
			continue
		}
		return e, strings.TrimSpace(t.LegendFormat)
	}
	return "", ""
}

func panelIDSet(ids []int) map[int]struct{} {
	if len(ids) == 0 {
		return nil
	}
	m := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		m[id] = struct{}{}
	}
	return m
}
