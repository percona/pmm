// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/*
 * TS Configuration stmts: DefineStmt, RenameStmt and DropStmt are default
 */
type AlterTSConfigType uint

const (
	ALTER_TSCONFIG_ADD_MAPPING AlterTSConfigType = iota
	ALTER_TSCONFIG_ALTER_MAPPING_FOR_TOKEN
	ALTER_TSCONFIG_REPLACE_DICT
	ALTER_TSCONFIG_REPLACE_DICT_FOR_TOKEN
	ALTER_TSCONFIG_DROP_MAPPING
)
