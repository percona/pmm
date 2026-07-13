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

func TestBuildSlackChatRequestPrependsSystemWhenMissing(t *testing.T) {
	enabled := true
	s := &models.Settings{}
	s.Adre.Enabled = &enabled
	s.Adre.URL = "http://holmes.test"
	s.Adre.AdreMaxConversationMessages = 40

	hist := []any{
		map[string]any{"role": "user", "content": "prior"},
	}
	req := BuildSlackChatRequest(s, "ask", hist, "")
	require.NotNil(t, req)
	require.GreaterOrEqual(t, len(req.ConversationHistory), 2)

	first, ok := req.ConversationHistory[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "system", first["role"])
}
