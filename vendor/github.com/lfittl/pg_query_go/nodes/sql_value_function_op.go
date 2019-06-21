// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

/*
 * SQLValueFunction - parameterless functions with special grammar productions
 *
 * The SQL standard categorizes some of these as <datetime value function>
 * and others as <general value specification>.  We call 'em SQLValueFunctions
 * for lack of a better term.  We store type and typmod of the result so that
 * some code doesn't need to know each function individually, and because
 * we would need to store typmod anyway for some of the datetime functions.
 * Note that currently, all variants return non-collating datatypes, so we do
 * not need a collation field; also, all these functions are stable.
 */
type SQLValueFunctionOp uint

const (
	SVFOP_CURRENT_DATE SQLValueFunctionOp = iota
	SVFOP_CURRENT_TIME
	SVFOP_CURRENT_TIME_N
	SVFOP_CURRENT_TIMESTAMP
	SVFOP_CURRENT_TIMESTAMP_N
	SVFOP_LOCALTIME
	SVFOP_LOCALTIME_N
	SVFOP_LOCALTIMESTAMP
	SVFOP_LOCALTIMESTAMP_N
	SVFOP_CURRENT_ROLE
	SVFOP_CURRENT_USER
	SVFOP_USER
	SVFOP_SESSION_USER
	SVFOP_CURRENT_CATALOG
	SVFOP_CURRENT_SCHEMA
)
