package models

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

// DumpFilters represents filters for artifacts list.
type DumpFilters struct {
	// Return only artifacts by specified status.
	Status BackupStatus
}

// FindDumps returns dumps list sorted by creation time in DESCENDING order.
func FindDumps(q *reform.Querier, filters DumpFilters) ([]*Dump, error) {
	var conditions []string
	var args []interface{}
	var idx int

	if filters.Status != "" {
		idx++
		conditions = append(conditions, fmt.Sprintf("status = %s", q.Placeholder(idx)))
		args = append(args, filters.Status)
	}

	var whereClause string
	if len(conditions) != 0 {
		whereClause = fmt.Sprintf("WHERE %s", strings.Join(conditions, " AND "))
	}
	rows, err := q.SelectAllFrom(DumpTable, fmt.Sprintf("%s ORDER BY created_at DESC", whereClause), args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select dumps")
	}

	dumps := make([]*Dump, 0, len(rows))
	for _, r := range rows {
		dumps = append(dumps, r.(*Dump)) //nolint:forcetypeassert
	}

	return dumps, nil
}

// FindDumpByID returns dump by given ID if found, ErrNotFound if not.
func FindDumpByID(q *reform.Querier, id string) (*Dump, error) {
	if id == "" {
		return nil, errors.New("provided dump id is empty")
	}

	dump := &Dump{ID: id}
	err := q.Reload(dump)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, errors.Wrapf(ErrNotFound, "dump by id '%s'", id)
		}
		return nil, errors.WithStack(err)
	}

	return dump, nil
}

// DeleteDump removes dump by ID.
func DeleteDump(q *reform.Querier, id string) error {
	if _, err := FindDumpByID(q, id); err != nil {
		return err
	}

	if err := q.Delete(&Dump{ID: id}); err != nil {
		return errors.Wrapf(err, "failed to delete dump by id '%s'", id)
	}
	return nil
}

// IsDumpFinalStatus checks if dump status is one of the final ones.
func IsDumpFinalStatus(dumpStatus DumpStatus) bool {
	switch dumpStatus {
	case DumpStatusSuccess, DumpStatusError:
		return true
	default:
		return false
	}
}
