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

package slackbot

import (
	"strings"
	"testing"

	"github.com/slack-go/slack/slackevents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestParseSlackAlert(t *testing.T) {
	t.Parallel()

	t.Run("firing with templated fingerprint", func(t *testing.T) {
		t.Parallel()
		ev := &slackevents.MessageEvent{Text: "[FIRING:1] pmm_mysql_down MySQL\n**Firing**\n" +
			"Value: A=1\nLabels:\n - alertname = pmm_mysql_down\n - severity = critical\n" +
			" - service_name = mysql-mysql\nFingerprint: a1b2c3d4e5f6a1b2"}
		alert, ok := parseSlackAlert(ev)
		require.True(t, ok)
		assert.Equal(t, "a1b2c3d4e5f6a1b2", alert.Fingerprint)
		assert.Equal(t, "firing", alert.Status)
		assert.Equal(t, "pmm_mysql_down", alert.Labels["alertname"])
		assert.Equal(t, "critical", alert.Labels["severity"])
		assert.Equal(t, "mysql-mysql", alert.Labels["service_name"])
	})

	t.Run("resolved status", func(t *testing.T) {
		t.Parallel()
		ev := &slackevents.MessageEvent{Text: "[RESOLVED] pmm_mysql_down\nLabels:\n - alertname = pmm_mysql_down\nFingerprint: deadbeefdeadbeef"}
		alert, ok := parseSlackAlert(ev)
		require.True(t, ok)
		assert.Equal(t, "resolved", alert.Status)
	})

	t.Run("no alertname is not an alert", func(t *testing.T) {
		t.Parallel()
		_, ok := parseSlackAlert(&slackevents.MessageEvent{Text: "just some channel chatter, nothing here"})
		assert.False(t, ok)
	})

	t.Run("derived stable fingerprint when none templated", func(t *testing.T) {
		t.Parallel()
		ev := &slackevents.MessageEvent{Text: "[FIRING:1]\nLabels:\n - alertname = slow_queries\n - severity = warning"}
		alert, ok := parseSlackAlert(ev)
		require.True(t, ok)
		assert.True(t, strings.HasPrefix(alert.Fingerprint, "slack-"))
		again, _ := parseSlackAlert(ev)
		assert.Equal(t, alert.Fingerprint, again.Fingerprint, "derived fingerprint must be stable across re-notifications")
	})
}

func TestIsAlertSource(t *testing.T) {
	t.Parallel()
	s := &models.Settings{}
	s.Adre.SlackAutoInvestigateChannels = []string{"C123"}

	assert.False(t, isAlertSource(s, "C999", "B1"), "channel not in alert list")
	assert.True(t, isAlertSource(s, "C123", "B1"), "alert channel, no bot filter ⇒ any bot")

	s.Adre.SlackAlertBotIDs = []string{"BGRAFANA"}
	assert.False(t, isAlertSource(s, "C123", "B1"), "bot not allow-listed")
	assert.True(t, isAlertSource(s, "C123", "BGRAFANA"), "allow-listed bot in alert channel")
}
