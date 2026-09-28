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

	"github.com/google/uuid"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/common"
)

// maybeSendAdvisorNotification emails the completed run's insights to the configured recipients
// when notifications are enabled. It is best-effort: every failure is logged and swallowed so it
// never affects the check run.
func (s *Service) maybeSendAdvisorNotification(ctx context.Context, runID string, triggeredBy models.CheckTriggeredBy) {
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
		s.l.Warnf("Advisor notification: enabled, but no recipients are configured, so run %s was "+
			"not emailed. Set the Advisor notification email addresses in the PMM settings.", runID)
		return
	}

	results, _, err := s.GetInsights(ctx, models.InsightFilters{RunID: runID}, 0, 0)
	if err != nil {
		s.l.Warnf("Advisor notification: failed to load results for run %s: %v", runID, err)
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

	subject := fmt.Sprintf("PMM Advisor Insights: %d finding(s) for run %s", len(texts), runID)
	body := buildAdvisorEmailReport(runID, triggeredBy, threshold, sCounts, tCounts, texts)

	err = s.sendAdvisorEmail(an.EmailAddresses, subject, body)
	if err != nil {
		s.l.Warnf("Advisor notification: failed to email run %s: %v", runID, err)
		return
	}
	s.l.Infof("Advisor notification: emailed %d insight(s) for run %s", len(texts), runID)
}

// SendTestNotification emails a sample Advisor report to the given recipients so that email
// delivery can be verified without waiting for a run to produce findings. It executes no checks
// and persists nothing; the findings in the report are made up.
func (s *Service) SendTestNotification(recipients []string) error {
	threshold := models.AdvisorNotificationSeverityDefault
	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		// the report only needs the threshold to describe itself, so a settings
		// failure must not stand between the operator and a delivery test
		s.l.Warnf("Advisor test notification: failed to load settings, reporting the default threshold: %v", err)
	} else if settings.AdvisorNotifications.SeverityThreshold != common.Unknown {
		threshold = settings.AdvisorNotifications.SeverityThreshold
	}

	runID := uuid.New().String()
	sCounts := make(map[common.Severity]int)
	tCounts := make(map[models.ServiceType]int)
	texts := make([]string, 0, len(sampleInsights))
	for _, sample := range sampleInsights {
		r := sample(runID)
		severity := common.Severity(r.Severity)
		// mirror the real report's filter, so the sample shows what this
		// threshold would actually deliver
		if severity > threshold {
			continue
		}
		text, err := insightToText(r)
		if err != nil {
			return fmt.Errorf("failed to format the sample insight: %w", err)
		}
		sCounts[severity]++
		tCounts[r.ServiceType]++
		texts = append(texts, text)
	}

	subject := fmt.Sprintf("[Test] PMM Advisor Insights: %d sample finding(s)", len(texts))
	body := testReportPreamble + buildAdvisorEmailReport(
		runID, models.CheckTriggeredByUser, threshold, sCounts, tCounts, texts,
	)

	return s.sendAdvisorEmail(recipients, subject, body)
}

// testReportPreamble opens the test email, so that a recipient who was not the one pressing the
// button cannot mistake the made-up findings below it for real ones.
const testReportPreamble = "This is a test message sent from the PMM settings to confirm that " +
	"Advisor notifications reach this address. The findings below are samples: no Advisor checks " +
	"were run, and nothing was recorded in PMM.\n\n"

// sampleInsights builds one made-up finding per advisor severity level, covering all three
// technologies, so a test report exercises the same formatting a real one does. Each takes the run
// ID so the sample reads consistently.
var sampleInsights = []func(runID string) *models.Insight{
	func(runID string) *models.Insight {
		return sampleInsight(runID, models.Severity(common.Critical), "mongodb_auth", "Security",
			"mongo-prod-1", models.MongoDBServiceType,
			"MongoDB authentication is disabled",
			"Warns if MongoDB authentication is disabled.",
			"https://docs.mongodb.com/manual/tutorial/enable-authentication/")
	},
	func(runID string) *models.Insight {
		return sampleInsight(runID, models.Severity(common.Error), "postgresql_fsync", "Durability",
			"pg-prod-1", models.PostgreSQLServiceType,
			"PostgreSQL fsync is set to off",
			"This check returns an error if the fsync configuration option is off which can lead to database corruption.",
			"https://www.postgresql.org/docs/current/runtime-config-wal.html")
	},
	func(runID string) *models.Insight {
		return sampleInsight(runID, models.Severity(common.Warning), "mysql_version", "Versions",
			"mysql-prod-1", models.MySQLServiceType,
			"MySQL version 8.0.36 is not the latest",
			"This check returns warnings if MySQL, Percona Server for MySQL, or MariaDB version is not the latest one.",
			"https://www.percona.com/downloads")
	},
	func(runID string) *models.Insight {
		return sampleInsight(runID, models.Severity(common.Info), "mysql_tables_without_pk", "Schema & indexes",
			"mysql-prod-1", models.MySQLServiceType,
			"2 table(s) have no primary key",
			"Checks tables without primary keys.",
			"https://docs.percona.com/percona-monitoring-and-management/3/advisors/checks/mysql-tables-without-pk.html")
	},
}

// sampleInsight fills the fields insightToText renders, so a sample formats exactly like a real
// insight. Identifiers are marked as samples rather than looking like real IDs.
func sampleInsight(
	runID string,
	severity models.Severity,
	checkName, category, serviceName string,
	serviceType models.ServiceType,
	summary, description, readMoreURL string,
) *models.Insight {
	return &models.Insight{
		ID:             "sample-" + checkName,
		RunID:          runID,
		CheckName:      checkName,
		Category:       category,
		Interval:       models.Standard,
		ServiceID:      "sample-service-id",
		ServiceName:    serviceName,
		ServiceType:    serviceType,
		NodeID:         "sample-node-id",
		NodeName:       serviceName + "-node",
		Environment:    "production",
		Cluster:        "sample-cluster",
		ReplicationSet: "sample-rs",
		Status:         models.CheckResultFailed,
		Summary:        summary,
		Description:    description,
		Outcome:        summary,
		ReadMoreURL:    readMoreURL,
		Severity:       severity,
		CheckedAt:      models.Now(),
		TriggeredBy:    models.CheckTriggeredByUser,
	}
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
	runID string,
	triggeredBy models.CheckTriggeredBy,
	threshold common.Severity,
	sCounts map[common.Severity]int,
	tCounts map[models.ServiceType]int,
	insights []string,
) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Percona Monitoring and Management runs Advisor checks against your monitored "+
		"databases to surface potential issues. This report covers run %s, which was %s. It found %d "+
		"insight(s) at or above the %q severity level that may need your attention.\n\n",
		runID, triggerPhrase(triggeredBy), len(insights), capitalize(threshold.String()))

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

// triggerPhrase describes, in prose, how the run was initiated.
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
