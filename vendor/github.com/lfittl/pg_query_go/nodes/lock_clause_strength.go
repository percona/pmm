// Auto-generated from postgres/src/include/nodes/lockoptions.h - DO NOT EDIT

package pg_query

/*
 * This enum represents the different strengths of FOR UPDATE/SHARE clauses.
 * The ordering here is important, because the highest numerical value takes
 * precedence when a RTE is specified multiple ways.  See applyLockingClause.
 */
type LockClauseStrength uint

const (
	LCS_NONE           LockClauseStrength = iota /* no such clause - only used in PlanRowMark */
	LCS_FORKEYSHARE                              /* FOR KEY SHARE */
	LCS_FORSHARE                                 /* FOR SHARE */
	LCS_FORNOKEYUPDATE                           /* FOR NO KEY UPDATE */
	LCS_FORUPDATE                                /* FOR UPDATE */
)
