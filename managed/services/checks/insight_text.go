// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package checks

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/common"
)

// insightTimeFormat mirrors TIME_FORMAT in ui/apps/pmm/src/lib/constants.ts.
const insightTimeFormat = "2006-01-02 15:04:05"

// The maps below mirror the display labels in ui/apps/pmm/src/lib/constants.ts so that the
// webhook payload reads identically to the UI's "Copy to text" output.

var insightStatusText = map[models.CheckResultStatus]string{
	models.CheckResultOK:     "OK",
	models.CheckResultFailed: "Failed",
	models.CheckResultError:  "Error",
}

var insightSeverityText = map[models.Severity]string{
	models.Severity(common.Critical): "Critical",
	models.Severity(common.Error):    "Error",
	models.Severity(common.Warning):  "Warning",
	models.Severity(common.Info):     "Info",
}

var insightIntervalText = map[models.Interval]string{
	models.Standard: "Standard",
	models.Rare:     "Rare",
	models.Frequent: "Frequent",
}

var insightTriggeredByText = map[models.CheckTriggeredBy]string{
	models.CheckTriggeredByUser:      "User",
	models.CheckTriggeredByScheduler: "Scheduler",
}

// insightToText renders an Advisor check result as a human-readable narrative, matching the
// UI's "Copy to text" command (insightToText in ui/apps/pmm/src/pages/advisors/insights/AdvisorInsights.utils.ts).
func insightToText(r *models.Insight) (string, error) {
	labelsMap, err := r.GetLabels()
	if err != nil {
		return "", fmt.Errorf("failed to decode labels for check result %q: %w", r.ID, err)
	}

	keys := make([]string, 0, len(labelsMap))
	for k := range labelsMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	labelPairs := make([]string, 0, len(keys))
	for _, k := range keys {
		labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, labelsMap[k]))
	}
	labels := strings.Join(labelPairs, ", ")

	var checkedAt string
	if !r.CheckedAt.IsZero() {
		checkedAt = r.CheckedAt.Format(insightTimeFormat)
	}

	details := [][2]string{
		{"ID", r.ID},
		{"Run ID", r.RunID},
		{"Check Name", r.CheckName},
		{"Category", r.Category},
		{"Service Name", r.ServiceName},
		{"Service Type", string(r.ServiceType)},
		{"Node Name", r.NodeName},
		{"Environment", r.Environment},
		{"Cluster", r.Cluster},
		{"Replication Set", r.ReplicationSet},
		{"Interval", insightIntervalLabel(r.Interval)},
		{"Triggered By", insightTriggeredByText[r.TriggeredBy]},
		{"Read", insightReadLabel(r.IsRead)},
		{"Summary", r.Summary},
		{"Description", r.Description},
		{"Outcome", r.Outcome},
		{"Severity", insightSeverityLabel(r.Severity)},
		{"Read More", r.ReadMoreURL},
		{"Labels", labels},
	}

	var detailLines []string
	for _, d := range details {
		if d[1] == "" {
			continue
		}
		detailLines = append(detailLines, fmt.Sprintf("  %s: %s", d[0], d[1]))
	}

	return fmt.Sprintf(
		"The Advisor Check %q completed at %s with status %q.\n\nCheck Details:\n%s",
		r.Summary, checkedAt, insightStatusLabel(r.Status), strings.Join(detailLines, "\n"),
	), nil
}

// insightIntervalLabel maps a stored interval to its display label; an empty interval means
// standard (see convertModelInterval in managed/services/management/checks.go).
func insightIntervalLabel(interval models.Interval) string {
	if interval == "" {
		return insightIntervalText[models.Standard]
	}
	if label, ok := insightIntervalText[interval]; ok {
		return label
	}
	return "Unspecified"
}

func insightStatusLabel(status models.CheckResultStatus) string {
	if label, ok := insightStatusText[status]; ok {
		return label
	}
	return "Unspecified"
}

func insightSeverityLabel(severity models.Severity) string {
	if label, ok := insightSeverityText[severity]; ok {
		return label
	}
	return "Unspecified"
}

func insightReadLabel(isRead bool) string {
	if isRead {
		return "Read"
	}
	return "Unread"
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
