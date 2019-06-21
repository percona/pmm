// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/*
 * RoleSpec - a role name or one of a few special values.
 */
type RoleSpecType uint

const (
	ROLESPEC_CSTRING      RoleSpecType = iota /* role name is stored as a C string */
	ROLESPEC_CURRENT_USER                     /* role spec is CURRENT_USER */
	ROLESPEC_SESSION_USER                     /* role spec is SESSION_USER */
	ROLESPEC_PUBLIC                           /* role name is "public" */
)
