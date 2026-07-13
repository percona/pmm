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
	"unicode/utf8"
)

// slackMaxMessageLen is a conservative byte cap for a single Slack message body. Slack rejects
// over-long messages with "msg_too_long"; long investigation reports are split into chunks below this.
const slackMaxMessageLen = 3500

// chunkForSlack splits text into pieces no larger than slackMaxMessageLen, preferring to break on a
// blank line, then a newline, then a UTF-8-safe hard cut. Returns nil for empty input.
func chunkForSlack(text string) []string {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil
	}
	var chunks []string
	for len(text) > slackMaxMessageLen {
		cut := slackSplitIndex(text)
		chunks = append(chunks, strings.TrimRight(text[:cut], "\n"))
		text = strings.TrimLeft(text[cut:], "\n")
	}
	if text != "" {
		chunks = append(chunks, text)
	}
	return chunks
}

// slackSplitIndex returns a byte index (<= slackMaxMessageLen) to cut at: the last blank line, else
// the last newline, else a hard cut backed up to a UTF-8 rune boundary. Only called when the input
// is longer than slackMaxMessageLen.
func slackSplitIndex(s string) int {
	window := s[:slackMaxMessageLen]
	if i := strings.LastIndex(window, "\n\n"); i > 0 {
		return i
	}
	if i := strings.LastIndex(window, "\n"); i > 0 {
		return i
	}
	limit := slackMaxMessageLen
	for limit > 0 && !utf8.RuneStart(s[limit]) {
		limit--
	}
	return limit
}

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
		if len(m) != 3 { //nolint:mnd
			return full
		}
		label, u := m[1], m[2]
		// "|" and "<" / ">" in display text break Slack link parsing — normalize minimally.
		label = strings.ReplaceAll(label, "|", " ")
		return "<" + u + "|" + label + ">"
	})
}
