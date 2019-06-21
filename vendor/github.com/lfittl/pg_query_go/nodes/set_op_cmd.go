// Auto-generated from postgres/src/include/nodes/nodes.h - DO NOT EDIT

package pg_query

/*
 * SetOpCmd and SetOpStrategy -
 *	  overall semantics and execution strategies for SetOp plan nodes
 *
 * This is needed in both plannodes.h and relation.h, so put it here...
 */
type SetOpCmd uint

const (
	SETOPCMD_INTERSECT SetOpCmd = iota
	SETOPCMD_INTERSECT_ALL
	SETOPCMD_EXCEPT
	SETOPCMD_EXCEPT_ALL
)
