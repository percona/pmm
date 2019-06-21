// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

type DropBehavior uint

const (
	DROP_RESTRICT DropBehavior = iota /* drop fails if any dependent objects */
	DROP_CASCADE                      /* remove dependent objects too */
)
