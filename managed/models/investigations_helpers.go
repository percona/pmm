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
	"fmt"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

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
		if errors.As(err, &reform.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &inv, nil
}

// allowedOrderBy columns that can be used in ORDER BY (safe, no user-controlled SQL).
var allowedOrderBy = map[string]bool{"title": true, "status": true, "created_at": true, "updated_at": true}

// allowedOrder directions for ORDER BY.
var allowedOrder = map[string]bool{"asc": true, "desc": true}

// ListInvestigations returns investigations with optional status filter and configurable sort. StatusFilter empty means all.
func ListInvestigations(q *reform.DB, statusFilter string, limit, offset int, orderBy, order string) ([]*Investigation, error) {
	if !allowedOrderBy[orderBy] {
		orderBy = "updated_at"
	}
	if !allowedOrder[order] {
		order = "desc"
	}
	where := fmt.Sprintf("ORDER BY %s %s", orderBy, order)
	var args []any
	if statusFilter != "" {
		where = fmt.Sprintf("WHERE status = $1 ORDER BY %s %s", orderBy, order)
		args = append(args, statusFilter)
	}
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
		result[i] = r.(*Investigation)
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
		if errors.As(err, &reform.ErrNoRows) {
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
		result[i] = r.(*InvestigationBlock)
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
		if errors.As(err, &reform.ErrNoRows) {
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
		result[i] = r.(*InvestigationMessage)
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
		result[i] = r.(*InvestigationComment)
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
		result[i] = r.(*InvestigationTimelineEvent)
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
		result[i] = r.(*InvestigationArtifact)
	}
	return result, nil
}
