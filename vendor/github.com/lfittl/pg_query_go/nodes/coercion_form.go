// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

/*
 * CoercionForm - how to display a node that could have come from a cast
 *
 * NB: equal() ignores CoercionForm fields, therefore this *must* not carry
 * any semantically significant information.  We need that behavior so that
 * the planner will consider equivalent implicit and explicit casts to be
 * equivalent.  In cases where those actually behave differently, the coercion
 * function's arguments will be different.
 */
type CoercionForm uint

const (
	COERCE_EXPLICIT_CALL CoercionForm = iota /* display as a function call */
	COERCE_EXPLICIT_CAST                     /* display as an explicit cast */
	COERCE_IMPLICIT_CAST                     /* implicit cast, so hide it */
)
