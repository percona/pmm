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
	graphIntentRE = regexp.MustCompile(`(?i)\b(graph|chart|panel|visuali[sz]ation|show me)\b`)
	mdImgRE       = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
)

// NeedsGraphRetry mirrors the standalone Slack bot: user asked for a graph, no blob image in answer, leaked tool markers.
// Heuristic only (wording and markdown variants differ); optional second chat may still not produce a panel.
func NeedsGraphRetry(userText, analysis string) bool {
	if userText == "" || analysis == "" {
		return false
	}
	if !graphIntentRE.MatchString(userText) {
		return false
	}
	if strings.Contains(analysis, "/v1/grafana/render/blob/") {
		return false
	}
	if mdImgRE.MatchString(analysis) {
		return false
	}
	return toolDirectiveRE.MatchString(analysis)
}

const graphRetryPrompt = `Return final user-facing text only. Do not output tool directives like <<{...}>>. ` +
	`If the user asks for a graph/chart/panel, execute pmm_render_grafana_panel and include ` +
	`![panel](image_url) plus [Open in Grafana](dashboard_url).`
