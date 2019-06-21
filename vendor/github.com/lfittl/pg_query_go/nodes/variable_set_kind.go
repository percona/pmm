// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/* ----------------------
 * SET Statement (includes RESET)
 *
 * "SET var TO DEFAULT" and "RESET var" are semantically equivalent, but we
 * preserve the distinction in VariableSetKind for CreateCommandTag().
 * ----------------------
 */
type VariableSetKind uint

const (
	VAR_SET_VALUE   VariableSetKind = iota /* SET var = value */
	VAR_SET_DEFAULT                        /* SET var TO DEFAULT */
	VAR_SET_CURRENT                        /* SET var FROM CURRENT */
	VAR_SET_MULTI                          /* special case for SET TRANSACTION ... */
	VAR_RESET                              /* RESET var */
	VAR_RESET_ALL                          /* RESET ALL */
)
