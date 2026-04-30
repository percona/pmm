// Copyright (C) 2026 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package slackbot

import (
	"regexp"
	"strings"
)

var toolDirectiveRE = regexp.MustCompile(`<<\s*\{.*?\}\s*>>`)
var mdImgHideRE = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
var blobURLHideRE = regexp.MustCompile(`https?://\S+/v1/grafana/render/blob/[a-f0-9]{64}\.png`)
var imageLineHideRE = regexp.MustCompile(`(?im)^\s*image:\s*\S+\s*$`)

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
	return strings.TrimSpace(text)
}
