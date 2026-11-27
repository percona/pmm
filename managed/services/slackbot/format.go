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
		cut := splitIndex(text, slackMaxMessageLen)
		chunks = append(chunks, strings.TrimRight(text[:cut], "\n"))
		text = strings.TrimLeft(text[cut:], "\n")
	}
	if text != "" {
		chunks = append(chunks, text)
	}
	return chunks
}

// splitIndex returns a byte index (<= limit) to cut at: the last blank line, else the last newline,
// else a hard cut backed up to a UTF-8 rune boundary. Only called when the input is longer than limit.
func splitIndex(s string, limit int) int {
	window := s[:limit]
	if i := strings.LastIndex(window, "\n\n"); i > 0 {
		return i
	}
	if i := strings.LastIndex(window, "\n"); i > 0 {
		return i
	}
	for limit > 0 && !utf8.RuneStart(s[limit]) {
		limit--
	}
	return limit
}

// slackMaxMarkdownLen is the byte cap for one message's markdown block: Slack allows 12,000 characters
// of markdown per message, and the margin covers the escaping Slack applies.
const slackMaxMarkdownLen = 10000

// chunkMarkdownForSlack splits markdown into messages within slackMaxMarkdownLen, cutting at a blank
// line outside code blocks so tables and code arrive whole. Returns nil for empty input.
func chunkMarkdownForSlack(md string) []string {
	md = strings.Trim(md, "\n")
	var chunks []string
	for len(md) > slackMaxMarkdownLen {
		cut := markdownSplitIndex(md)
		chunks = append(chunks, strings.TrimRight(md[:cut], "\n"))
		md = strings.TrimLeft(md[cut:], "\n")
	}
	if md != "" {
		chunks = append(chunks, md)
	}
	return chunks
}

// markdownSplitIndex returns the last blank line outside a code fence within slackMaxMarkdownLen,
// else the plain split.
func markdownSplitIndex(s string) int {
	inFence := false
	cut := 0
	for i := 0; i < slackMaxMarkdownLen; {
		end := strings.IndexByte(s[i:], '\n')
		if end < 0 || i+end >= slackMaxMarkdownLen {
			break
		}
		line := strings.TrimSpace(s[i : i+end])
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
		}
		if line == "" && !inFence && i > 0 {
			cut = i
		}
		i += end + 1
	}
	if cut > 0 {
		return cut
	}
	return splitIndex(s, slackMaxMarkdownLen)
}

var (
	toolDirectiveRE = regexp.MustCompile(`<<\s*\{.*?\}\s*>>`)
	mdImgHideRE     = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	blobURLHideRE   = regexp.MustCompile(`https?://\S+/v1/grafana/render/blob/[a-f0-9]{64}\.png`)
	imageLineHideRE = regexp.MustCompile(`(?im)^\s*image:\s*\S+\s*$`)
)

// ADRE/Holmes emits Markdown links [label](https://...). Slack mrkdwn uses <https://...|label> instead.
var mdHTTPSLinkRE = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)]+)\)`)

// FormatAnswerForSlack cleans ADRE markdown for a Slack markdown block: drops tool directives and makes
// blob links absolute. When hideImageLinks is true, strips markdown images and raw blob URLs that were
// uploaded as files.
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
	return strings.TrimSpace(text)
}

var (
	mdHeadingRE = regexp.MustCompile(`(?m)^#{1,6}[ \t]+(.+?)[ \t#]*$`)
	mdBoldRE    = regexp.MustCompile(`\*\*(.+?)\*\*`)
)

// slackMrkdwn rewrites markdown into Slack's older mrkdwn for plain-text messages: links, headings and
// bold. Tables stay as written.
func slackMrkdwn(md string) string {
	text := markdownHTTPSLinksToSlackMrkdwn(md)
	text = mdBoldRE.ReplaceAllString(text, "*$1*")
	return mdHeadingRE.ReplaceAllString(text, "*$1*")
}

// slackNotificationLen caps the plain text Slack shows in notifications for a message sent as blocks.
const slackNotificationLen = 150

var mdLinkLabelRE = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)

// notificationText is the plain line Slack shows in notifications and previews: the first line of
// prose, without markdown.
func notificationText(md string) string {
	inFence := false
	for line := range strings.SplitSeq(md, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence || line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "|") || strings.HasPrefix(line, "!") {
			continue
		}
		line = mdLinkLabelRE.ReplaceAllString(line, "$1")
		line = strings.NewReplacer("**", "", "__", "", "`", "").Replace(line)
		line = strings.TrimLeft(line, "-*> ")
		if r := []rune(line); len(r) > slackNotificationLen {
			line = strings.TrimSpace(string(r[:slackNotificationLen])) + "…"
		}
		if line != "" {
			return line
		}
	}
	return "PMM Assistant"
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
