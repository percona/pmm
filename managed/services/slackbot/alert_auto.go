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
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// SlackNotifier posts auto-investigate output to Slack. It is invoked by the autoinvestigate service
// (reconciliation poll / webhook) independently of the Socket Mode session, so it builds its own
// Slack client from the stored bot token. It satisfies autoinvestigate.Notifier.
type SlackNotifier struct {
	db *reform.DB
	l  *logrus.Entry
}

// NewSlackNotifier creates a SlackNotifier.
func NewSlackNotifier(db *reform.DB, l *logrus.Entry) *SlackNotifier {
	return &SlackNotifier{db: db, l: l.WithField("component", "adre-slack-notify")}
}

// PostAutoInvestigateStarted posts a short notice (with a link to the investigation) to each output
// channel. It is a no-op when channels is empty or the Slack bot token is unavailable.
func (n *SlackNotifier) PostAutoInvestigateStarted(ctx context.Context, channels []string, inv *models.Investigation) {
	if len(channels) == 0 || inv == nil {
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
	if settings, sErr := models.GetSettings(n.db); sErr == nil {
		if base := settings.GetEffectiveSlackLinkBaseURL(); base != "" {
			link = " — " + base + "/pmm-ui/investigations/" + inv.ID
		}
	}
	msg := fmt.Sprintf("🔎 Auto-investigation started for *%s*.%s", inv.Title, link)

	for _, ch := range channels {
		if _, _, err := api.PostMessageContext(ctx, ch, slack.MsgOptionText(msg, false)); err != nil {
			n.l.Debugf("PostMessage to %s: %v", ch, err)
		}
	}
}
