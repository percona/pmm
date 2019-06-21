// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/* ----------------------
 * Discard Statement
 * ----------------------
 */
type DiscardMode uint

const (
	DISCARD_ALL DiscardMode = iota
	DISCARD_PLANS
	DISCARD_SEQUENCES
	DISCARD_TEMP
)
