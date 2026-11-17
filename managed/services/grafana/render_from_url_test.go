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

func TestFromToForGrafanaImageRendererQuery_AbsoluteToEpochMs(t *testing.T) {
	from, to := fromToForGrafanaImageRendererQuery("2026-04-29T14:30:00Z", "2026-04-29T20:30:00Z")
	assert.NotContains(t, from, ":")
	assert.NotContains(t, to, ":")
	assert.Len(t, from, 13)
	assert.Len(t, to, 13)
	assert.NotEqual(t, "2026-04-29T14:30:00Z", from)
}

func TestFromToForGrafanaImageRendererQuery_RelativePassthrough(t *testing.T) {
	from, to := fromToForGrafanaImageRendererQuery("now-6h", "now")
	assert.Equal(t, "now-6h", from)
	assert.Equal(t, "now", to)
}
