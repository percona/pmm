// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/* Sort ordering options for ORDER BY and CREATE INDEX */
type SortByDir uint

const (
	SORTBY_DEFAULT SortByDir = iota
	SORTBY_ASC
	SORTBY_DESC
	SORTBY_USING /* not allowed in CREATE INDEX ... */
)
