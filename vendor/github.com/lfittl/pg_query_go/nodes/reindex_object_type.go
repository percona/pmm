// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/* Reindex options */
type ReindexObjectType uint

const (
	REINDEX_OBJECT_INDEX    ReindexObjectType = iota /* index */
	REINDEX_OBJECT_TABLE                             /* table or materialized view */
	REINDEX_OBJECT_SCHEMA                            /* schema */
	REINDEX_OBJECT_SYSTEM                            /* system catalogs */
	REINDEX_OBJECT_DATABASE                          /* database */
)
