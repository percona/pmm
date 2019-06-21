// Auto-generated from postgres/src/include/nodes/lockoptions.h - DO NOT EDIT

package pg_query

/*
 * This enum controls how to deal with rows being locked by FOR UPDATE/SHARE
 * clauses (i.e., it represents the NOWAIT and SKIP LOCKED options).
 * The ordering here is important, because the highest numerical value takes
 * precedence when a RTE is specified multiple ways.  See applyLockingClause.
 */
type LockWaitPolicy uint

const (
	/* Wait for the lock to become available (default behavior) */
	LockWaitBlock LockWaitPolicy = iota

	/* Skip rows that can't be locked (SKIP LOCKED) */
	LockWaitSkip

	/* Raise an error if a row cannot be locked (NOWAIT) */
	LockWaitError
)
