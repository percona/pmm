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

// Package slackbot implements PMM's Slack integration: a Socket Mode bot that proxies authorized
// human messages into the ADRE chat backend and posts answers back into Slack threads, plus a
// notifier that posts auto-investigation output (driven by Grafana Alertmanager via
// services/autoinvestigate, not by scraping Slack messages) to configured channels.
package slackbot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// SlackNotifier posts auto-investigate output to Slack. It is invoked by the autoinvestigate service
// (scrape / reconciliation poll / webhook) independently of the Socket Mode session, so it builds its
// own Slack client from the stored bot token. It satisfies autoinvestigate.Notifier and
// investigations' report-notifier interface.
type SlackNotifier struct {
	db *reform.DB
	l  *logrus.Entry
}

// NewSlackNotifier creates a SlackNotifier.
func NewSlackNotifier(db *reform.DB, l *logrus.Entry) *SlackNotifier {
	return &SlackNotifier{db: db, l: l.WithField("component", "adre-slack-notify")}
}

// slackThreadRef returns the (channel, thread_ts) an investigation was scraped from, or empties when it
// did not originate from a Slack message (webhook/poll path). buildAlertInvestigation stores these in
// the investigation's Config JSON.
func slackThreadRef(inv *models.Investigation) (string, string) {
	if inv == nil || len(inv.Config) == 0 {
		return "", ""
	}
	var cfg struct {
		SlackChannel  string `json:"slack_channel"`
		SlackThreadTS string `json:"slack_thread_ts"`
	}
	if err := json.Unmarshal(inv.Config, &cfg); err != nil {
		return "", ""
	}
	return cfg.SlackChannel, cfg.SlackThreadTS
}

// investigationLink returns the UI link to an investigation, or "" when no public base URL is set.
func (n *SlackNotifier) investigationLink(inv *models.Investigation) string {
	if settings, err := models.GetSettings(n.db); err == nil {
		if base := settings.GetEffectiveSlackLinkBaseURL(); base != "" {
			return base + "/pmm-ui/investigations/" + inv.ID
		}
	}
	return ""
}

// PostAutoInvestigateStarted posts a short "started" notice with a link to the investigation. For an
// alert scraped from Slack it replies in that alert's thread; otherwise (webhook/poll) it posts to the
// configured output channels. No-op when there is nowhere to post or the bot token is unavailable.
func (n *SlackNotifier) PostAutoInvestigateStarted(ctx context.Context, channels []string, inv *models.Investigation) {
	if inv == nil {
		return
	}
	threadChannel, threadTS := slackThreadRef(inv)
	if threadChannel == "" && len(channels) == 0 {
		return
	}
	prov, err := models.GetAdreProvisioning(n.db)
	if err != nil {
		n.l.Debugf("GetAdreProvisioning: %v", err)
		return
	}
	if prov.SlackBotToken == "" {
		return
	}
	api := slack.New(prov.SlackBotToken)

	link := ""
	if u := n.investigationLink(inv); u != "" {
		link = " — " + u
	}
	msg := fmt.Sprintf("🔎 Auto-investigation started for *%s*.%s", inv.Title, link)

	if threadChannel != "" {
		if _, _, err := api.PostMessageContext(ctx, threadChannel, slack.MsgOptionText(msg, false), slack.MsgOptionTS(threadTS)); err != nil {
			n.l.Debugf("PostMessage (thread) to %s: %v", threadChannel, err)
		}
		return
	}
	for _, ch := range channels {
		if _, _, err := api.PostMessageContext(ctx, ch, slack.MsgOptionText(msg, false)); err != nil {
			n.l.Debugf("PostMessage to %s: %v", ch, err)
		}
	}
}

// PostInvestigationReport posts the completed investigation's summary as a reply in the alert's Slack
// thread. It is in-thread only: investigations that did not originate from a scraped Slack message
// (webhook/poll) are a no-op here — they are read in the UI. It satisfies the investigations
// report-notifier interface.
func (n *SlackNotifier) PostInvestigationReport(ctx context.Context, inv *models.Investigation) {
	if inv == nil {
		return
	}
	channel, threadTS := slackThreadRef(inv)
	if channel == "" || threadTS == "" {
		return
	}
	prov, err := models.GetAdreProvisioning(n.db)
	if err != nil || prov.SlackBotToken == "" {
		return
	}
	link := n.investigationLink(inv)
	var body string
	if inv.Status == "failed" {
		body = fmt.Sprintf("❌ Investigation failed for *%s*.", inv.Title)
		if link != "" {
			body += " — " + link
		}
	} else {
		body = buildReportSummary(inv, link)
	}
	if body == "" {
		return
	}
	api := slack.New(prov.SlackBotToken)
	// Post each chunk as a thread reply; continue past a failed chunk so one transient error doesn't
	// truncate the rest of the report (mirrors postAnswer's behaviour).
	for _, chunk := range chunkForSlack(body) {
		postThreadLine(ctx, n.l, api, channel, threadTS, chunk)
	}
}

// buildReportSummary renders a concise Slack summary of a completed investigation: root cause +
// resolution (or the summary as a fallback) + a link to the full report in the UI.
func buildReportSummary(inv *models.Investigation, link string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("✅ Investigation complete: *%s*\n", inv.Title))
	rootCause := strings.TrimSpace(inv.RootCauseSummary)
	resolution := strings.TrimSpace(inv.ResolutionSummary)
	if rootCause != "" {
		b.WriteString("\n*Root cause:* " + rootCause + "\n")
	}
	if resolution != "" {
		b.WriteString("\n*Resolution:* " + resolution + "\n")
	}
	if rootCause == "" && resolution == "" {
		if s := strings.TrimSpace(inv.Summary); s != "" {
			b.WriteString("\n" + s + "\n")
		}
	}
	if link != "" {
		b.WriteString("\nFull report: " + link)
	}
	return strings.TrimSpace(b.String())
}
