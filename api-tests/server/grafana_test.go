// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
)

func TestGrafanaCallsHomeDisabled(t *testing.T) {
	t.Parallel()

	resp, err := pmmapitests.GetGrafanaClient(t).Admin.AdminGetSettings()
	require.NoError(t, err)
	settings := resp.Payload

	for _, s := range []struct {
		section string
		key     string
		want    string
	}{
		{"analytics", "check_for_updates", "false"},
		{"analytics", "reporting_enabled", "false"},
		{"analytics", "check_for_plugin_updates", "false"},
		{"analytics", "enabled", "false"},
		{"analytics", "feedback_links_enabled", "false"},
		{"news", "news_feed_enabled", "false"},
		{"reporting", "enabled", "false"},
		{"security", "disable_gravatar", "true"},
	} {
		assert.Equalf(t, s.want, settings[s.section][s.key], "[%s] %s", s.section, s.key)
	}
}
