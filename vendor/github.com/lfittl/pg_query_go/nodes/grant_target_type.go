// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/* ----------------------
 *		Grant|Revoke Statement
 * ----------------------
 */
type GrantTargetType uint

const (
	ACL_TARGET_OBJECT        GrantTargetType = iota /* grant on specific named object(s) */
	ACL_TARGET_ALL_IN_SCHEMA                        /* grant on all objects in given schema(s) */
	ACL_TARGET_DEFAULTS                             /* ALTER DEFAULT PRIVILEGES */
)
