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

package models

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/reform.v1"
)

// StartAdvisorRun records the beginning of an Advisor checks execution. The
// counts stay zero and finished_at stays NULL until FinishAdvisorRun is called.
func StartAdvisorRun(ctx context.Context, q *reform.Querier, r *AdvisorRun) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return q.WithContext(ctx).Insert(r)
}

// AdvisorRunCounts holds the totals denormalized onto a run when it completes.
type AdvisorRunCounts struct {
	ChecksCount    int
	ServicesCount  int
	FindingsCount  int
	ErrorsCount    int
	SeverityCounts map[Severity]int
}

// FinishAdvisorRun marks a run as complete and stores its totals. A missing run
// row is not an error: runs recorded before this table existed have nothing to
// update, and a failed insert must not break the check run itself.
func FinishAdvisorRun(ctx context.Context, q *reform.Querier, id string, finishedAt time.Time, counts AdvisorRunCounts) error {
	run := &AdvisorRun{ID: id}
	err := q.WithContext(ctx).Reload(run)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("failed to load advisor run '%s': %w", id, err)
	}

	run.FinishedAt = &finishedAt
	run.ChecksCount = counts.ChecksCount
	run.ServicesCount = counts.ServicesCount
	run.FindingsCount = counts.FindingsCount
	run.ErrorsCount = counts.ErrorsCount
	err = run.SetSeverityCounts(counts.SeverityCounts)
	if err != nil {
		return err
	}

	err = q.WithContext(ctx).Update(run)
	if err != nil {
		return fmt.Errorf("failed to update advisor run '%s': %w", id, err)
	}
	return nil
}

// ComputeAdvisorRunCounts derives a run's totals from the insights it recorded.
// The insights are the authoritative record of what the run produced, so the
// stored counts cannot drift from the rows they summarize.
func ComputeAdvisorRunCounts(ctx context.Context, q *reform.Querier, runID string) (AdvisorRunCounts, error) {
	var counts AdvisorRunCounts

	failed := CheckResultFailed
	errored := CheckResultError
	err := q.QueryRowContext(
		ctx,
		"SELECT count(DISTINCT check_name), count(DISTINCT service_id), "+
			"count(*) FILTER (WHERE status = $1), count(*) FILTER (WHERE status = $2) "+
			"FROM "+InsightTable.Name()+" WHERE run_id = $3",
		failed, errored, runID,
	).Scan(&counts.ChecksCount, &counts.ServicesCount, &counts.FindingsCount, &counts.ErrorsCount)
	if err != nil {
		return counts, fmt.Errorf("failed to count insights for run '%s': %w", runID, err)
	}

	rows, err := q.QueryContext(
		ctx,
		"SELECT severity, count(*) FROM "+InsightTable.Name()+
			" WHERE run_id = $1 AND status = $2 GROUP BY severity",
		runID, failed,
	)
	if err != nil {
		return counts, fmt.Errorf("failed to count severities for run '%s': %w", runID, err)
	}
	defer rows.Close() //nolint:errcheck

	counts.SeverityCounts = make(map[Severity]int)
	for rows.Next() {
		var severity Severity
		var count int
		err = rows.Scan(&severity, &count)
		if err != nil {
			return counts, fmt.Errorf("failed to scan severity count for run '%s': %w", runID, err)
		}
		counts.SeverityCounts[severity] = count
	}
	err = rows.Err()
	if err != nil {
		return counts, fmt.Errorf("failed to read severity counts for run '%s': %w", runID, err)
	}

	return counts, nil
}

// FindUnfinishedAdvisorRuns returns runs that never recorded a completion. After
// a restart these cannot still be running, so the caller closes them out.
func FindUnfinishedAdvisorRuns(ctx context.Context, q *reform.Querier) ([]*AdvisorRun, error) {
	rows, err := q.WithContext(ctx).SelectAllFrom(AdvisorRunTable, "WHERE finished_at IS NULL")
	if err != nil {
		return nil, fmt.Errorf("failed to select unfinished advisor runs: %w", err)
	}

	runs := make([]*AdvisorRun, 0, len(rows))
	for _, r := range rows {
		runs = append(runs, r.(*AdvisorRun)) //nolint:forcetypeassert
	}
	return runs, nil
}

// LastInsightTimeForRun returns when the run last recorded an insight. The
// second result is false when the run produced none.
func LastInsightTimeForRun(ctx context.Context, q *reform.Querier, runID string) (time.Time, bool, error) {
	var last *time.Time
	err := q.QueryRowContext(
		ctx,
		"SELECT max(checked_at) FROM "+InsightTable.Name()+" WHERE run_id = $1", runID,
	).Scan(&last)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("failed to read last insight time for run '%s': %w", runID, err)
	}
	if last == nil {
		return time.Time{}, false, nil
	}
	return last.UTC(), true, nil
}

// AdvisorRunFilters specifies filters for querying Advisor runs.
type AdvisorRunFilters struct {
	TriggeredBy *CheckTriggeredBy
	From        *time.Time
	To          *time.Time
}

// advisorRunConditions builds the WHERE clause and arguments for the given filters.
func advisorRunConditions(q *reform.Querier, filters AdvisorRunFilters) (string, []any) {
	var conditions []string
	var args []any

	if filters.TriggeredBy != nil {
		conditions = append(conditions, "triggered_by = "+q.Placeholder(len(args)+1))
		args = append(args, *filters.TriggeredBy)
	}
	if filters.From != nil {
		conditions = append(conditions, "started_at >= "+q.Placeholder(len(args)+1))
		args = append(args, *filters.From)
	}
	if filters.To != nil {
		conditions = append(conditions, "started_at <= "+q.Placeholder(len(args)+1))
		args = append(args, *filters.To)
	}

	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

// FindAdvisorRuns returns Advisor runs matching the filters, newest first. When
// pageSize is greater than zero, the results are paginated.
func FindAdvisorRuns(ctx context.Context, q *reform.Querier, filters AdvisorRunFilters, pageIndex, pageSize int) ([]*AdvisorRun, error) {
	tail, args := advisorRunConditions(q, filters)
	tail += " ORDER BY started_at DESC"
	if pageSize > 0 {
		tail += " LIMIT " + q.Placeholder(len(args)+1)
		args = append(args, pageSize)
		tail += " OFFSET " + q.Placeholder(len(args)+1)
		args = append(args, pageIndex*pageSize)
	}

	rows, err := q.WithContext(ctx).SelectAllFrom(AdvisorRunTable, tail, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to select advisor runs: %w", err)
	}

	runs := make([]*AdvisorRun, 0, len(rows))
	for _, r := range rows {
		runs = append(runs, r.(*AdvisorRun)) //nolint:forcetypeassert
	}
	return runs, nil
}

// CountAdvisorRuns returns the number of Advisor runs matching the filters.
func CountAdvisorRuns(ctx context.Context, q *reform.Querier, filters AdvisorRunFilters) (int, error) {
	where, args := advisorRunConditions(q, filters)

	var count int
	err := q.QueryRowContext(ctx, "SELECT count(*) FROM "+AdvisorRunTable.Name()+" "+where, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count advisor runs: %w", err)
	}
	return count, nil
}

// CleanupOldAdvisorRuns deletes Advisor runs started at or before the given time.
// Runs are pruned by their own start time rather than with their insights, so a
// run whose insights are already gone still reports its stored totals.
func CleanupOldAdvisorRuns(ctx context.Context, q *reform.Querier, olderThan time.Time) error {
	_, err := q.WithContext(ctx).DeleteFrom(AdvisorRunTable, " WHERE started_at <= $1", olderThan)
	return err
}
