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

package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"gopkg.in/reform.v1"
)

// activeInvestigationStatusSQL is the status set considered "active" for an alert episode: an
// investigation in one of these statuses blocks a duplicate claim for the same alert fingerprint.
// It MUST stay in sync with the investigations_active_alert partial unique index (database.go).
const activeInvestigationStatusSQL = "('open','in_progress','investigating','running')"

// uniqueViolationCode is the PostgreSQL SQLSTATE for a unique-constraint violation.
const uniqueViolationCode = "23505"

// CreateInvestigation inserts a new investigation. ID must be set (e.g. NewInvestigationID()).
func CreateInvestigation(q *reform.DB, inv *Investigation) error {
	if inv.ID == "" {
		return errors.New("investigation id is required")
	}
	now := time.Now().UTC()
	inv.CreatedAt = now
	inv.UpdatedAt = now
	return q.Save(inv)
}

// GetInvestigationByID loads an investigation by id. Returns nil, nil if not found.
func GetInvestigationByID(q *reform.DB, id string) (*Investigation, error) {
	var inv Investigation
	err := q.FindByPrimaryKeyTo(&inv, id)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, nil //nolint:nilnil // "not found" sentinel matching managed/models convention
		}
		return nil, err
	}
	return &inv, nil
}

// FindActiveInvestigationByFingerprint returns the most recent active (non-terminal) investigation
// for an alert fingerprint, or nil if none. An empty fingerprint returns nil.
func FindActiveInvestigationByFingerprint(q *reform.DB, fingerprint string) (*Investigation, error) {
	if fingerprint == "" {
		return nil, nil //nolint:nilnil // "not found" sentinel matching managed/models convention
	}
	tail := "WHERE alert_fingerprint = $1 AND status IN " + activeInvestigationStatusSQL + " ORDER BY created_at DESC LIMIT 1"
	records, err := q.SelectAllFrom(InvestigationTable, tail, fingerprint)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil //nolint:nilnil // "not found" sentinel matching managed/models convention
	}
	return records[0].(*Investigation), nil //nolint:forcetypeassert // SelectAllFrom on InvestigationTable guarantees this type
}

// FindLatestInvestigationByFingerprint returns the most recent investigation (any status) for an
// alert fingerprint, or nil if none. The auto-investigate trigger uses it to reason about firing
// episodes (e.g. whether a re-notification belongs to an already-investigated episode).
func FindLatestInvestigationByFingerprint(q *reform.DB, fingerprint string) (*Investigation, error) {
	if fingerprint == "" {
		return nil, nil //nolint:nilnil // "not found" sentinel matching managed/models convention
	}
	records, err := q.SelectAllFrom(InvestigationTable, "WHERE alert_fingerprint = $1 ORDER BY created_at DESC LIMIT 1", fingerprint)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil //nolint:nilnil // "not found" sentinel matching managed/models convention
	}
	return records[0].(*Investigation), nil //nolint:forcetypeassert // SelectAllFrom on InvestigationTable guarantees this type
}

// CountAutoInvestigationsSince returns the number of auto-investigations (created_by =
// 'auto-investigate') created at or after since. It backs a global, HA-safe, restart-safe hourly cap.
func CountAutoInvestigationsSince(q *reform.DB, since time.Time) (int, error) {
	var n int
	err := q.QueryRow(
		"SELECT COUNT(*) FROM investigations WHERE created_by = $1 AND created_at >= $2",
		AutoInvestigateCreatedBy, since,
	).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// AutoInvestigateCreatedBy marks investigations created by the auto-investigate pipeline (also used
// as the created_by filter for the hourly cap).
const AutoInvestigateCreatedBy = "auto-investigate"

// ClaimInvestigationForAlert atomically ensures at most one active investigation per alert
// fingerprint. If an active investigation already exists it is returned with claimed=false; otherwise
// inv is inserted and returned with claimed=true. Concurrent claims are serialized by the
// investigations_active_alert partial unique index: the loser observes a unique violation and is
// handed the winning investigation.
func ClaimInvestigationForAlert(db *reform.DB, inv *Investigation) (*Investigation, bool, error) {
	if inv.AlertFingerprint == "" {
		return nil, false, errors.New("alert_fingerprint is required to claim an investigation")
	}
	existing, err := FindActiveInvestigationByFingerprint(db, inv.AlertFingerprint)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		return existing, false, nil
	}
	err = CreateInvestigation(db, inv)
	if err == nil {
		return inv, true, nil
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && string(pqErr.Code) == uniqueViolationCode {
		existing, lookupErr := FindActiveInvestigationByFingerprint(db, inv.AlertFingerprint)
		if lookupErr != nil {
			return nil, false, lookupErr
		}
		if existing != nil {
			return existing, false, nil
		}
	}
	return nil, false, err
}

// allowedOrderBy columns that can be used in ORDER BY (safe, no user-controlled SQL).
var allowedOrderBy = map[string]bool{"title": true, "status": true, "created_at": true, "updated_at": true} //nolint:goconst

// allowedOrder directions for ORDER BY.
var allowedOrder = map[string]bool{"asc": true, "desc": true}

// ListInvestigations returns investigations with optional status and trigger filters and configurable
// sort. StatusFilter empty means all statuses. triggerFilter is "auto" (created by the auto-investigate
// pipeline), "manual" (anything else), or "" (all) — keyed on created_by, since source_type cannot
// distinguish a manual "Investigate this alert" from an auto one.
func ListInvestigations(q *reform.DB, statusFilter, triggerFilter string, limit, offset int, orderBy, order string) ([]*Investigation, error) {
	if !allowedOrderBy[orderBy] {
		orderBy = "updated_at"
	}
	if !allowedOrder[order] {
		order = "desc"
	}
	var conds []string
	var args []any
	if statusFilter != "" {
		args = append(args, statusFilter)
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)))
	}
	switch triggerFilter {
	case "auto":
		args = append(args, AutoInvestigateCreatedBy)
		conds = append(conds, fmt.Sprintf("created_by = $%d", len(args)))
	case "manual":
		args = append(args, AutoInvestigateCreatedBy)
		conds = append(conds, fmt.Sprintf("created_by <> $%d", len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ") + " "
	}
	where += fmt.Sprintf("ORDER BY %s %s", orderBy, order)
	if limit > 0 {
		where += fmt.Sprintf(" LIMIT %d", limit)
	}
	if offset > 0 {
		where += fmt.Sprintf(" OFFSET %d", offset)
	}
	records, err := q.SelectAllFrom(InvestigationTable, where, args...)
	if err != nil {
		return nil, err
	}
	result := make([]*Investigation, len(records))
	for i, r := range records {
		result[i] = r.(*Investigation) //nolint:forcetypeassert // reform.SelectAllFrom on InvestigationTable guarantees this type
	}
	return result, nil
}

// UpdateInvestigation updates an existing investigation.
func UpdateInvestigation(q *reform.DB, inv *Investigation) error {
	inv.UpdatedAt = time.Now().UTC()
	return q.Save(inv)
}

// DeleteInvestigation deletes an investigation (cascade deletes blocks, messages, etc.).
func DeleteInvestigation(q *reform.DB, id string) error {
	var inv Investigation
	err := q.FindByPrimaryKeyTo(&inv, id)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return err
	}
	return q.Delete(&inv)
}

// CreateInvestigationBlock inserts a block. ID must be set.
func CreateInvestigationBlock(q *reform.DB, b *InvestigationBlock) error {
	now := time.Now().UTC()
	b.CreatedAt = now
	b.UpdatedAt = now
	return q.Save(b)
}

// GetInvestigationBlocks returns blocks for an investigation ordered by position.
func GetInvestigationBlocks(q *reform.DB, investigationID string) ([]*InvestigationBlock, error) {
	records, err := q.SelectAllFrom(InvestigationBlockTable, "WHERE investigation_id = $1 ORDER BY position ASC", investigationID)
	if err != nil {
		return nil, err
	}
	result := make([]*InvestigationBlock, len(records))
	for i, r := range records {
		result[i] = r.(*InvestigationBlock) //nolint:forcetypeassert // reform.SelectAllFrom on InvestigationBlockTable guarantees this type
	}
	return result, nil
}

// UpdateInvestigationBlock updates a block.
func UpdateInvestigationBlock(q *reform.DB, b *InvestigationBlock) error {
	b.UpdatedAt = time.Now().UTC()
	return q.Save(b)
}

// DeleteInvestigationBlock deletes a block.
func DeleteInvestigationBlock(q *reform.DB, id string) error {
	var b InvestigationBlock
	err := q.FindByPrimaryKeyTo(&b, id)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return err
	}
	return q.Delete(&b)
}

// DeleteInvestigationBlocksForInvestigation removes all blocks for an investigation (e.g. before replacing with a new report).
func DeleteInvestigationBlocksForInvestigation(q *reform.DB, investigationID string) error {
	_, err := q.DeleteFrom(InvestigationBlockTable, " WHERE investigation_id = $1", investigationID)
	return err
}

// CreateInvestigationMessage inserts a message.
func CreateInvestigationMessage(q *reform.DB, m *InvestigationMessage) error {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	return q.Save(m)
}

// GetInvestigationMessages returns messages for an investigation, newest first, with limit and offset.
func GetInvestigationMessages(q *reform.DB, investigationID string, limit, offset int) ([]*InvestigationMessage, error) {
	where := "WHERE investigation_id = $1 ORDER BY created_at DESC"
	args := []any{investigationID}
	if limit > 0 {
		where += fmt.Sprintf(" LIMIT %d", limit)
	}
	if offset > 0 {
		where += fmt.Sprintf(" OFFSET %d", offset)
	}
	records, err := q.SelectAllFrom(InvestigationMessageTable, where, args...)
	if err != nil {
		return nil, err
	}
	result := make([]*InvestigationMessage, len(records))
	for i, r := range records {
		result[i] = r.(*InvestigationMessage) //nolint:forcetypeassert // reform.SelectAllFrom on InvestigationMessageTable guarantees this type
	}
	return result, nil
}

// CreateInvestigationComment inserts a comment.
func CreateInvestigationComment(q *reform.DB, c *InvestigationComment) error {
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	return q.Save(c)
}

// GetInvestigationComments returns comments for an investigation, optionally filtered by block_id.
func GetInvestigationComments(q *reform.DB, investigationID string, blockID *string) ([]*InvestigationComment, error) {
	where := "WHERE investigation_id = $1"
	args := []any{investigationID}
	if blockID != nil && *blockID != "" {
		where += " AND block_id = $2"
		args = append(args, *blockID)
	}
	where += " ORDER BY created_at ASC"
	records, err := q.SelectAllFrom(InvestigationCommentTable, where, args...)
	if err != nil {
		return nil, err
	}
	result := make([]*InvestigationComment, len(records))
	for i, r := range records {
		result[i] = r.(*InvestigationComment) //nolint:forcetypeassert // reform.SelectAllFrom on InvestigationCommentTable guarantees this type
	}
	return result, nil
}

// CreateInvestigationTimelineEvent inserts a timeline event.
func CreateInvestigationTimelineEvent(q *reform.DB, e *InvestigationTimelineEvent) error {
	return q.Save(e)
}

// GetInvestigationTimelineEvents returns timeline events for an investigation ordered by event_time.
func GetInvestigationTimelineEvents(q *reform.DB, investigationID string) ([]*InvestigationTimelineEvent, error) {
	records, err := q.SelectAllFrom(InvestigationTimelineEventTable, "WHERE investigation_id = $1 ORDER BY event_time ASC", investigationID)
	if err != nil {
		return nil, err
	}
	result := make([]*InvestigationTimelineEvent, len(records))
	for i, r := range records {
		result[i] = r.(*InvestigationTimelineEvent) //nolint:forcetypeassert // reform.SelectAllFrom on InvestigationTimelineEventTable guarantees this type
	}
	return result, nil
}

// DeleteInvestigationTimelineEventsForInvestigation removes all timeline events for an investigation (e.g. before replacing with a new report).
func DeleteInvestigationTimelineEventsForInvestigation(q *reform.DB, investigationID string) error {
	_, err := q.DeleteFrom(InvestigationTimelineEventTable, " WHERE investigation_id = $1", investigationID)
	return err
}

// CreateInvestigationArtifact inserts an artifact.
func CreateInvestigationArtifact(q *reform.DB, a *InvestigationArtifact) error {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	return q.Save(a)
}

// GetInvestigationArtifacts returns artifacts for an investigation.
func GetInvestigationArtifacts(q *reform.DB, investigationID string) ([]*InvestigationArtifact, error) {
	records, err := q.SelectAllFrom(InvestigationArtifactTable, "WHERE investigation_id = $1 ORDER BY created_at ASC", investigationID)
	if err != nil {
		return nil, err
	}
	result := make([]*InvestigationArtifact, len(records))
	for i, r := range records {
		result[i] = r.(*InvestigationArtifact) //nolint:forcetypeassert // reform.SelectAllFrom on InvestigationArtifactTable guarantees this type
	}
	return result, nil
}
