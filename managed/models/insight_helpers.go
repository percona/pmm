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
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/reform.v1"
)

// CreateInsight inserts a single Advisor check result into the history.
func CreateInsight(ctx context.Context, q *reform.Querier, r *Insight) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return q.WithContext(ctx).Insert(r)
}

// InsightFilters specifies filters for querying Advisor insights.
type InsightFilters struct {
	ServiceID string
	// ServiceName is matched as a case-insensitive substring.
	ServiceName string
	// NodeName is matched as a case-insensitive substring.
	NodeName    string
	Category    string
	CheckName   string
	BatchID     string
	TriggeredBy *CheckTriggeredBy
	Severity    *Severity
	Status      *CheckResultStatus
	IsRead      *bool
	From        *time.Time
	To          *time.Time
}

// insightConditions builds the WHERE clause and arguments for the given filters.
func insightConditions(q *reform.Querier, filters InsightFilters) (string, []any) {
	var conditions []string
	var args []any

	if filters.ServiceID != "" {
		conditions = append(conditions, "service_id = "+q.Placeholder(len(args)+1))
		args = append(args, filters.ServiceID)
	}
	if filters.ServiceName != "" {
		conditions = append(conditions, "service_name ILIKE "+q.Placeholder(len(args)+1))
		args = append(args, "%"+filters.ServiceName+"%")
	}
	if filters.NodeName != "" {
		conditions = append(conditions, "node_name ILIKE "+q.Placeholder(len(args)+1))
		args = append(args, "%"+filters.NodeName+"%")
	}
	if filters.Category != "" {
		conditions = append(conditions, "category = "+q.Placeholder(len(args)+1))
		args = append(args, filters.Category)
	}
	if filters.CheckName != "" {
		conditions = append(conditions, "check_name = "+q.Placeholder(len(args)+1))
		args = append(args, filters.CheckName)
	}
	if filters.BatchID != "" {
		conditions = append(conditions, "batch_id = "+q.Placeholder(len(args)+1))
		args = append(args, filters.BatchID)
	}
	if filters.TriggeredBy != nil {
		conditions = append(conditions, "triggered_by = "+q.Placeholder(len(args)+1))
		args = append(args, *filters.TriggeredBy)
	}
	if filters.Severity != nil {
		conditions = append(conditions, "severity = "+q.Placeholder(len(args)+1))
		args = append(args, *filters.Severity)
	}
	if filters.Status != nil {
		conditions = append(conditions, "status = "+q.Placeholder(len(args)+1))
		args = append(args, *filters.Status)
	}
	if filters.IsRead != nil {
		conditions = append(conditions, "is_read = "+q.Placeholder(len(args)+1))
		args = append(args, *filters.IsRead)
	}
	if filters.From != nil {
		conditions = append(conditions, "checked_at >= "+q.Placeholder(len(args)+1))
		args = append(args, *filters.From)
	}
	if filters.To != nil {
		conditions = append(conditions, "checked_at <= "+q.Placeholder(len(args)+1))
		args = append(args, *filters.To)
	}

	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

// FindInsights returns Advisor insights matching the filters, ordered by
// checked_at descending. When pageSize is greater than zero, the results are paginated.
func FindInsights(ctx context.Context, q *reform.Querier, filters InsightFilters, pageIndex, pageSize int) ([]*Insight, error) {
	tail, args := insightConditions(q, filters)
	tail += " ORDER BY checked_at DESC"
	if pageSize > 0 {
		tail += " LIMIT " + q.Placeholder(len(args)+1)
		args = append(args, pageSize)
		tail += " OFFSET " + q.Placeholder(len(args)+1)
		args = append(args, pageIndex*pageSize)
	}

	rows, err := q.WithContext(ctx).SelectAllFrom(InsightTable, tail, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to select insights: %w", err)
	}

	results := make([]*Insight, 0, len(rows))
	for _, r := range rows {
		results = append(results, r.(*Insight)) //nolint:forcetypeassert
	}
	return results, nil
}

// CountInsights returns the number of Advisor insights rows matching the filters.
func CountInsights(ctx context.Context, q *reform.Querier, filters InsightFilters) (int, error) {
	where, args := insightConditions(q, filters)

	var count int
	err := q.QueryRowContext(ctx, "SELECT count(*) FROM "+InsightTable.Name()+" "+where, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count insights: %w", err)
	}
	return count, nil
}

// FindInsightFilterValues returns the distinct service and node names present in the
// Advisor insights, each sorted alphabetically.
func FindInsightFilterValues(ctx context.Context, q *reform.Querier) ([]string, []string, error) {
	distinct := func(column string) ([]string, error) {
		rows, err := q.QueryContext(ctx, "SELECT DISTINCT "+column+" FROM "+InsightTable.Name()+
			" ORDER BY "+column)
		if err != nil {
			return nil, fmt.Errorf("failed to select distinct %s: %w", column, err)
		}
		defer rows.Close() //nolint:errcheck

		var values []string
		for rows.Next() {
			var value string
			err = rows.Scan(&value)
			if err != nil {
				return nil, fmt.Errorf("failed to scan distinct %s: %w", column, err)
			}
			values = append(values, value)
		}
		return values, rows.Err()
	}

	serviceNames, err := distinct("service_name")
	if err != nil {
		return nil, nil, err
	}
	nodeNames, err := distinct("node_name")
	if err != nil {
		return nil, nil, err
	}
	return serviceNames, nodeNames, nil
}

// MarkInsightsRead sets the read state on the insights with the given IDs.
func MarkInsightsRead(ctx context.Context, q *reform.Querier, ids []string, isRead bool) error {
	if len(ids) == 0 {
		return nil
	}

	args := []any{isRead}
	placeholders := make([]string, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, q.Placeholder(len(args)+1))
		args = append(args, id)
	}

	query := "UPDATE " + InsightTable.Name() + " SET is_read = " + q.Placeholder(1) +
		" WHERE id IN (" + strings.Join(placeholders, ", ") + ")"
	_, err := q.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to mark insights as read: %w", err)
	}
	return nil
}

// MarkInsightsReadByFilters sets the read state on all insights matching the filters.
func MarkInsightsReadByFilters(ctx context.Context, q *reform.Querier, filters InsightFilters, isRead bool) error {
	where, args := insightConditions(q, filters)
	args = append(args, isRead)

	query := "UPDATE " + InsightTable.Name() + " SET is_read = " + q.Placeholder(len(args)) + " " + where
	_, err := q.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to mark insights as read by filters: %w", err)
	}
	return nil
}

// CleanupOldInsights deletes Advisor insights older than a specified date.
func CleanupOldInsights(ctx context.Context, q *reform.Querier, olderThan time.Time) error {
	_, err := q.WithContext(ctx).DeleteFrom(InsightTable, " WHERE checked_at <= $1", olderThan)
	return err
}
