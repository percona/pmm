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
	"regexp"
	"strings"
)

var (
	toolDirectiveRE = regexp.MustCompile(`<<\s*\{.*?\}\s*>>`)
	mdImgHideRE     = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	blobURLHideRE   = regexp.MustCompile(`https?://\S+/v1/grafana/render/blob/[a-f0-9]{64}\.png`)
	imageLineHideRE = regexp.MustCompile(`(?im)^\s*image:\s*\S+\s*$`)
)

// ADRE/Holmes emits Markdown links [label](https://...). Slack mrkdwn uses <https://...|label> instead.
var mdHTTPSLinkRE = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)]+)\)`)

// FormatAnswerForSlack converts ADRE markdown-ish output to plain text suitable for Slack chat.update.
// When hideImageLinks is true, strips markdown images and raw blob URLs that were uploaded as files.
func FormatAnswerForSlack(raw, publicBase string, hideImageLinks bool) string {
	if raw == "" {
		return raw
	}
	text := toolDirectiveRE.ReplaceAllString(raw, "")
	text = strings.TrimSpace(text)
	if hideImageLinks {
		text = mdImgHideRE.ReplaceAllString(text, "")
		text = blobURLHideRE.ReplaceAllString(text, "")
		text = imageLineHideRE.ReplaceAllString(text, "")
		text = strings.TrimSpace(text)
	}
	// Rewrite relative blob paths using public base when set.
	if publicBase != "" {
		base := strings.TrimRight(publicBase, "/")
		text = strings.ReplaceAll(text, "](/v1/grafana/render/blob/", "]("+base+"/v1/grafana/render/blob/")
		text = strings.ReplaceAll(text, "(/v1/grafana/render/blob/", "("+base+"/v1/grafana/render/blob/")
	}
	text = markdownHTTPSLinksToSlackMrkdwn(text)
	return strings.TrimSpace(text)
}

// markdownHTTPSLinksToSlackMrkdwn turns [label](https://host/path) into <https://host/path|label> for Slack mrkdwn.
// URLs must not contain ")" (common for Grafana); relative [label](/path) links are left unchanged.
func markdownHTTPSLinksToSlackMrkdwn(s string) string {
	return mdHTTPSLinkRE.ReplaceAllStringFunc(s, func(full string) string {
		m := mdHTTPSLinkRE.FindStringSubmatch(full)
		if len(m) != 3 {
			return full
		}
		label, u := m[1], m[2]
		// "|" and "<" / ">" in display text break Slack link parsing — normalize minimally.
		label = strings.ReplaceAll(label, "|", " ")
		return "<" + u + "|" + label + ">"
	})
}
