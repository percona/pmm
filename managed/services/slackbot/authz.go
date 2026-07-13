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

import "github.com/percona/pmm/managed/models"

// slackTrigger identifies how a human Slack turn was initiated (for audit and metrics labels).
type slackTrigger string

const (
	triggerMention slackTrigger = "mention" // explicit @mention of the bot
	triggerThread  slackTrigger = "thread"  // human reply in a bot thread
)

// authDecision is the result of authorizing a human Slack interaction.
type authDecision struct {
	allow  bool
	notify bool   // post the "ask an admin" reply (explicit interactions only)
	reason string // for metrics/log; never posted verbatim
}

// authorizeHuman authorizes a human chat interaction (mention or thread reply). It is fail-closed on
// both channel AND user: both must be allow-listed. A denied interaction is marked notify so the
// caller posts a single (rate-limited) informative reply.
func authorizeHuman(s *models.Settings, channelID, userID string) authDecision {
	if s.IsSlackChannelAllowed(channelID) && s.IsSlackUserAllowed(userID) {
		return authDecision{allow: true}
	}
	return authDecision{allow: false, notify: true, reason: "not_allowlisted"}
}
