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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentHashFromRenderParams_orderIndependentVars(t *testing.T) {
	a := ContentHashFromRenderParams(RenderCanonicalParams{
		DashboardUID: "Dash",
		PanelID:      "12",
		From:         "2026-04-29T10:00:00Z",
		To:           "2026-04-29T11:00:00Z",
		OrgID:        1,
		Width:        1000,
		Height:       500,
		Scale:        1,
		TZ:           "browser",
		Vars:         map[string]string{"var-b": "2", "var-a": "1"},
	})
	b := ContentHashFromRenderParams(RenderCanonicalParams{
		DashboardUID: "dash",
		PanelID:      "12",
		From:         "2026-04-29T10:00:00Z",
		To:           "2026-04-29T11:00:00Z",
		OrgID:        1,
		Width:        1000,
		Height:       500,
		Scale:        1,
		TZ:           "browser",
		Vars:         map[string]string{"var-a": "1", "var-b": "2"},
	})
	assert.Equal(t, a, b)
}

func TestNormalizePanelID(t *testing.T) {
	assert.Equal(t, "47", NormalizePanelID("panel-47"))
	assert.Equal(t, "47", NormalizePanelID("47"))
}

func TestNormalizeFromToForCanonical_MixedAbsoluteRelative(t *testing.T) {
	from, to := normalizeFromToForCanonical("2026-04-29T10:00:00Z", "now")
	assert.Equal(t, "2026-04-29T10:00:00Z", from)
	assert.Equal(t, "now", to)
}
