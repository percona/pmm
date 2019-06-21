// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

type TableLikeOption uint

const (
	CREATE_TABLE_LIKE_DEFAULTS TableLikeOption = iota
	CREATE_TABLE_LIKE_CONSTRAINTS
	CREATE_TABLE_LIKE_IDENTITY
	CREATE_TABLE_LIKE_INDEXES
	CREATE_TABLE_LIKE_STORAGE
	CREATE_TABLE_LIKE_COMMENTS
	CREATE_TABLE_LIKE_ALL
)
