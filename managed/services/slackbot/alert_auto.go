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

// Package slackbot implements PMM's Slack integration: a Socket Mode
// bot that proxies user messages and alerts into the ADRE chat backend
// and posts answers back into Slack threads.
package slackbot

import (
	"strings"

	"github.com/slack-go/slack/slackevents"
)

const slackAutoInvestigatePrefix = "Investigate this firing alert and summarize likely cause and next checks:\n\n"

// slackMessagePlainBlob joins top-level text and attachment fields for FIRING/RESOLVED gates
// and the Holmes prompt (v0 naive join).
func slackMessagePlainBlob(ev *slackevents.MessageEvent) string {
	var parts []string
	if t := strings.TrimSpace(ev.Text); t != "" {
		parts = append(parts, t)
	}
	if ev.Message != nil {
		for _, a := range ev.Message.Attachments {
			if s := strings.TrimSpace(a.Fallback); s != "" {
				parts = append(parts, s)
			}
			if s := strings.TrimSpace(a.Title); s != "" {
				parts = append(parts, s)
			}
			if s := strings.TrimSpace(a.Text); s != "" {
				parts = append(parts, s)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func slackBotMessageSubtypeOK(subType string) bool {
	return subType == "" || subType == slackSubtypeBotMessage
}

// slackSubtypeBotMessage matches Slack's message/bot_message subtype.
const slackSubtypeBotMessage = "bot_message"
