// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/* ----------------------
 *		Vacuum and Analyze Statements
 *
 * Even though these are nominally two statements, it's convenient to use
 * just one node type for both.  Note that at least one of VACOPT_VACUUM
 * and VACOPT_ANALYZE must be set in options.
 * ----------------------
 */
type VacuumOption uint

const (
	VACOPT_VACUUM VacuumOption = iota
	VACOPT_ANALYZE
	VACOPT_VERBOSE
	VACOPT_FREEZE
	VACOPT_FULL
	VACOPT_NOWAIT
	VACOPT_SKIPTOAST
	VACOPT_DISABLE_PAGE_SKIPPING
)
