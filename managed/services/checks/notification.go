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
	"context"
	"fmt"
	"strings"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/common"
)

// maybeSendAdvisorNotification emails the completed batch's insights to the configured Advisor email
// contact point when notifications are enabled. It is best-effort: every failure is logged and
// swallowed so it never affects the check run. The recipients are read from settings (cached at
// enable time by the server service), because this runs in a background context that cannot reach
// Grafana.
func (s *Service) maybeSendAdvisorNotification(ctx context.Context, batchID string, triggeredBy models.CheckTriggeredBy) {
	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		s.l.Warnf("Advisor notification: failed to load settings: %v", err)
		return
	}
	an := settings.AdvisorNotifications
	if !settings.IsAdvisorNotificationsEnabled() || len(an.EmailAddresses) == 0 {
		return
	}

	results, _, err := s.GetCheckResultsHistory(ctx, models.CheckResultFilters{BatchID: batchID}, 0, 0)
	if err != nil {
		s.l.Warnf("Advisor notification: failed to load results for batch %s: %v", batchID, err)
		return
	}

	threshold := an.SeverityThreshold
	if threshold == common.Unknown {
		threshold = common.Error
	}

	counts := make(map[common.Severity]int)
	texts := make([]string, 0, len(results))
	for _, r := range results {
		severity := common.Severity(r.Severity)
		// Keep only insights at least as severe as the threshold (a smaller value is more severe).
		if severity < common.Critical || severity > threshold {
			continue
		}
		text, err := insightToText(r)
		if err != nil {
			s.l.Warnf("Advisor notification: failed to format insight %s: %v", r.ID, err)
			continue
		}
		counts[severity]++
		texts = append(texts, text)
	}

	if len(texts) == 0 {
		return
	}

	subject := fmt.Sprintf("PMM Advisor Insights: %d finding(s) for batch %s", len(texts), batchID)
	body := buildAdvisorEmailReport(batchID, triggeredBy, threshold, counts, texts)

	err = s.sendAdvisorEmail(an.EmailAddresses, subject, body)
	if err != nil {
		s.l.Warnf("Advisor notification: failed to email batch %s: %v", batchID, err)
		return
	}
	s.l.Infof("Advisor notification: emailed %d insight(s) for batch %s", len(texts), batchID)
}

// buildAdvisorEmailReport composes the notification email body: a brief introduction, a per-severity
// summary, suggested next steps, and then the insights (formatted like the UI's "Copy to text")
// one after another.
func buildAdvisorEmailReport(batchID string, triggeredBy models.CheckTriggeredBy, threshold common.Severity, counts map[common.Severity]int, insights []string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Percona Monitoring and Management runs Advisor checks against your monitored "+
		"databases to surface potential issues. This report covers batch %s, which was %s. It found %d "+
		"insight(s) at or above the %q severity level that may need your attention.\n\n",
		batchID, triggerPhrase(triggeredBy), len(insights), capitalize(threshold.String()))

	b.WriteString("Findings by severity:\n")
	// Iterate from the most severe advisor level down to the configured threshold
	// (Critical=3 .. threshold), skipping the retired Notice level.
	for sev := common.Critical; sev <= threshold; sev++ {
		if sev == common.Notice {
			continue
		}
		fmt.Fprintf(&b, "  %s: %d\n", capitalize(sev.String()), counts[sev])
	}

	b.WriteString("\nNext steps:\n")
	b.WriteString("  - Review the insights below, addressing the most severe findings first.\n")
	b.WriteString("  - Follow each insight's \"Read More\" link for remediation guidance.\n")
	b.WriteString("  - Prioritize issues affecting production services.\n")
	b.WriteString("  - See full details in PMM under Advisors -> Insights.\n\n")

	fmt.Fprintf(&b, "Advisor Insights (%d):\n\n", len(insights))
	b.WriteString(strings.Join(insights, "\n\n"))

	return b.String()
}

// triggerPhrase describes, in prose, how the batch run was initiated.
func triggerPhrase(triggeredBy models.CheckTriggeredBy) string {
	switch triggeredBy {
	case models.CheckTriggeredByScheduler:
		return "run automatically on schedule"
	case models.CheckTriggeredByUser:
		return "triggered manually by an operator"
	default:
		return "run"
	}
}
