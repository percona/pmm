// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

/*
 * BoolExpr - expression node for the basic Boolean operators AND, OR, NOT
 *
 * Notice the arguments are given as a List.  For NOT, of course the list
 * must always have exactly one element.  For AND and OR, there can be two
 * or more arguments.
 */
type BoolExprType uint

const (
	AND_EXPR BoolExprType = iota
	OR_EXPR
)
