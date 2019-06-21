// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/* ----------------------
 *		Import Foreign Schema Statement
 * ----------------------
 */
type ImportForeignSchemaType uint

const (
	FDW_IMPORT_SCHEMA_ALL      ImportForeignSchemaType = iota /* all relations wanted */
	FDW_IMPORT_SCHEMA_LIMIT_TO                                /* include only listed tables in import */
	FDW_IMPORT_SCHEMA_EXCEPT                                  /* exclude listed tables from import */
)
