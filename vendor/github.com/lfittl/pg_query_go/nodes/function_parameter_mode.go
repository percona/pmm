// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

type FunctionParameterMode uint

const (
	/* the assigned enum values appear in pg_proc, don't change 'em! */
	FUNC_PARAM_IN FunctionParameterMode = iota
	FUNC_PARAM_OUT
	FUNC_PARAM_INOUT
	FUNC_PARAM_VARIADIC
	FUNC_PARAM_TABLE
)
