// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

/*
 * CoercionContext - distinguishes the allowed set of type casts
 *
 * NB: ordering of the alternatives is significant; later (larger) values
 * allow more casts than earlier ones.
 */
type CoercionContext uint

const (
	COERCION_IMPLICIT   CoercionContext = iota /* coercion in context of expression */
	COERCION_ASSIGNMENT                        /* coercion in context of assignment */
	COERCION_EXPLICIT                          /* explicit cast operation */
)
