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

// maybeSendAdvisorNotification emails the completed batch's insights to the configured recipients
// when notifications are enabled. It is best-effort: every failure is logged and swallowed so it
// never affects the check run.
func (s *Service) maybeSendAdvisorNotification(ctx context.Context, batchID string, triggeredBy models.CheckTriggeredBy) {
	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		s.l.Warnf("Advisor notification: failed to load settings: %v", err)
		return
	}
	an := settings.AdvisorNotifications
	if !settings.IsAdvisorNotificationsEnabled() {
		return
	}
	// ChangeSettings rejects this combination, so it only happens when notifications were enabled
	// through PMM_ENABLE_ADVISOR_NOTIFICATIONS without recipients ever being configured.
	if len(an.EmailAddresses) == 0 {
		s.l.Warnf("Advisor notification: enabled, but no recipients are configured, so batch %s was "+
			"not emailed. Set the Advisor notification email addresses in the PMM settings.", batchID)
		return
	}

	results, _, err := s.GetInsights(ctx, models.InsightFilters{BatchID: batchID}, 0, 0)
	if err != nil {
		s.l.Warnf("Advisor notification: failed to load results for batch %s: %v", batchID, err)
		return
	}

	threshold := an.SeverityThreshold
	if threshold == common.Unknown {
		threshold = common.Error
	}

	sCounts := make(map[common.Severity]int)
	tCounts := make(map[models.ServiceType]int)
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
		sCounts[severity]++
		tCounts[r.ServiceType]++
		texts = append(texts, text)
	}

	if len(texts) == 0 {
		return
	}

	subject := fmt.Sprintf("PMM Advisor Insights: %d finding(s) for batch %s", len(texts), batchID)
	body := buildAdvisorEmailReport(batchID, triggeredBy, threshold, sCounts, tCounts, texts)

	err = s.sendAdvisorEmail(an.EmailAddresses, subject, body)
	if err != nil {
		s.l.Warnf("Advisor notification: failed to email batch %s: %v", batchID, err)
		return
	}
	s.l.Infof("Advisor notification: emailed %d insight(s) for batch %s", len(texts), batchID)
}

// advisorTechnologies lists the technologies advisor checks run against, in report order, paired
// with the service type insights record. Checks can only target these, so the per-technology
// summary covers every insight.
var advisorTechnologies = []struct {
	serviceType models.ServiceType
	label       string
}{
	{models.MySQLServiceType, "MySQL"},
	{models.PostgreSQLServiceType, "PostgreSQL"},
	{models.MongoDBServiceType, "MongoDB"},
}

// buildAdvisorEmailReport composes the notification email body: a brief introduction, per-severity
// and per-technology summaries, suggested next steps, and then the insights (formatted like the
// UI's "Copy to text") one after another.
func buildAdvisorEmailReport(
	batchID string,
	triggeredBy models.CheckTriggeredBy,
	threshold common.Severity,
	sCounts map[common.Severity]int,
	tCounts map[models.ServiceType]int,
	insights []string,
) string {
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
		fmt.Fprintf(&b, "  %s: %d\n", capitalize(sev.String()), sCounts[sev])
	}

	// Every technology is listed, zero included, so the reader can tell "no findings" apart from
	// "not covered by this report".
	b.WriteString("\nFindings by technology:\n")
	for _, t := range advisorTechnologies {
		fmt.Fprintf(&b, "  %s: %d\n", t.label, tCounts[t.serviceType])
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
