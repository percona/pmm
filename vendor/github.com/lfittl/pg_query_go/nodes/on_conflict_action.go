// Auto-generated from postgres/src/include/nodes/nodes.h - DO NOT EDIT

package pg_query

/*
 * OnConflictAction -
 *	  "ON CONFLICT" clause type of query
 *
 * This is needed in both parsenodes.h and plannodes.h, so put it here...
 */
type OnConflictAction uint

const (
	ONCONFLICT_NONE    OnConflictAction = iota /* No "ON CONFLICT" clause */
	ONCONFLICT_NOTHING                         /* ON CONFLICT ... DO NOTHING */
	ONCONFLICT_UPDATE                          /* ON CONFLICT ... DO UPDATE */
)
