// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/* ----------------------
 *		Create View Statement
 * ----------------------
 */
type ViewCheckOption uint

const (
	NO_CHECK_OPTION ViewCheckOption = iota
	LOCAL_CHECK_OPTION
	CASCADED_CHECK_OPTION
)
