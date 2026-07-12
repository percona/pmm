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
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"slices"
	"strings"

	"github.com/slack-go/slack/slackevents"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/autoinvestigate"
)

var (
	// fingerprintRE extracts the Alertmanager fingerprint from a Grafana Slack message. Present only when
	// the contact-point template includes {{ .Fingerprint }}; parseSlackAlert falls back to a derived
	// key otherwise. (Loop safety is handled by the @-mention requirement in handleAlertScrape, not this.)
	fingerprintRE = regexp.MustCompile(`(?i)fingerprint[:=]\s*([a-f0-9]{8,})`)
	// labelLineRE matches Grafana's "Labels:" block lines, e.g. "- alertname = pmm_mysql_down".
	labelLineRE = regexp.MustCompile(`(?m)^\s*-?\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(.+?)\s*$`)
)

// slackMessagePlainBlob joins a Slack alert message's text and attachment fields into one searchable
// blob (Grafana posts the alert body via attachments).
func slackMessagePlainBlob(ev *slackevents.MessageEvent) string {
	var parts []string
	if t := strings.TrimSpace(ev.Text); t != "" {
		parts = append(parts, t)
	}
	if ev.Message != nil {
		for _, a := range ev.Message.Attachments {
			for _, s := range []string{a.Fallback, a.Pretext, a.Title, a.Text} {
				if s = strings.TrimSpace(s); s != "" {
					parts = append(parts, s)
				}
			}
		}
	}
	return strings.Join(parts, "\n")
}

// messageMentionsBot reports whether a Slack message @-mentions the given bot user id (Slack renders a
// mention as "<@Uxxxx>" or "<@Uxxxx|name>"). Alerts are identified by such a mention — the Grafana
// contact point's "Mention Users" is set to the PMM bot — which is the loop-safe marker: PMM writes its
// own notices/reports and never @-mentions itself, so its own posts are never scraped as alerts.
func messageMentionsBot(ev *slackevents.MessageEvent, botUserID string) bool {
	if botUserID == "" {
		return false
	}
	// A mention is "<@ID>" or "<@ID|label>": require the closing ">" or "|" so the whole id token matches.
	// A bare prefix check would let a mention of a different user whose id starts with ours — e.g.
	// "<@U1234>" vs bot "U123" — false-match and trigger a spurious investigation.
	blob := slackMessagePlainBlob(ev)
	return strings.Contains(blob, "<@"+botUserID+">") || strings.Contains(blob, "<@"+botUserID+"|")
}

// parseSlackAlert extracts an alert from a Grafana Slack message. ok is false when the message has no
// alertname. The fingerprint comes from the templated message when present, else a stable key derived
// from alertname+labels so re-notifications of the same alert coalesce.
func parseSlackAlert(ev *slackevents.MessageEvent) (autoinvestigate.Alert, bool) {
	blob := slackMessagePlainBlob(ev)
	if strings.TrimSpace(blob) == "" {
		return autoinvestigate.Alert{}, false
	}

	labels := map[string]string{}
	for _, m := range labelLineRE.FindAllStringSubmatch(blob, -1) {
		k, v := strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		if k == "" || v == "" {
			continue
		}
		if _, exists := labels[k]; !exists {
			labels[k] = v
		}
	}
	if labels["alertname"] == "" {
		return autoinvestigate.Alert{}, false
	}

	status := "firing"
	upper := strings.ToUpper(blob)
	if strings.Contains(upper, "[RESOLVED") || strings.Contains(upper, "RESOLVED]") || strings.Contains(upper, "**RESOLVED**") {
		status = "resolved"
	}

	// Fingerprint from the templated message when present ({{ .Fingerprint }}), else a stable key
	// derived from alertname+labels so re-notifications of the same alert coalesce. Loop safety comes
	// from the @-mention requirement in handleAlertScrape (our own reports/notices never mention the
	// bot), NOT from the fingerprint — so alerts whose template omits the fingerprint still investigate.
	fingerprint := ""
	if m := fingerprintRE.FindStringSubmatch(blob); m != nil {
		fingerprint = strings.ToLower(m[1])
	} else {
		fingerprint = derivedFingerprint(labels)
	}

	return autoinvestigate.Alert{
		Fingerprint: fingerprint,
		Status:      status,
		Labels:      labels,
	}, true
}

// derivedFingerprint is a stable episode key used when the Grafana Slack template does not surface the
// real fingerprint. It hashes alertname + sorted labels; Grafana sends identical labels on every
// re-notification of the same alert, so the key is stable across an episode.
func derivedFingerprint(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(labels[k])
		b.WriteString("\n")
	}
	sum := sha256.Sum256([]byte(b.String()))
	return "slack-" + hex.EncodeToString(sum[:16])
}

// isAlertSource reports whether a bot message in channelID from botID should be scraped as an alert:
// the channel must be a configured alert channel and (when SlackAlertBotIDs is set) the bot must match.
func isAlertSource(settings *models.Settings, channelID, botID string) bool {
	return settings.IsSlackAlertChannel(channelID) && settings.IsSlackAlertBot(botID)
}

// slackSubtypeBotMessage is Slack's subtype for messages posted by a bot/incoming webhook.
const slackSubtypeBotMessage = "bot_message"

// slackBotMessageSubtypeOK accepts only the subtypes Grafana alert posts use: none or "bot_message".
// Other bot subtypes (edits, deletes, joins) are ignored.
func slackBotMessageSubtypeOK(subType string) bool {
	return subType == "" || subType == slackSubtypeBotMessage
}
