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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestMessagesToHolmesHistory_OmitsToolRows(t *testing.T) {
	msgs := []models.AdreMessage{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
		{Role: "tool", ToolName: "navigate", ToolResultJSON: []byte(`{"ok":true}`)},
	}
	out := MessagesToHolmesHistory(msgs)
	require.Len(t, out, 2)
	m0, ok := out[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "user", m0["role"])
	m1, ok := out[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "assistant", m1["role"])
}
