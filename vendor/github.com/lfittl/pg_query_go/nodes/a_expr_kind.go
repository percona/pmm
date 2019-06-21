// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/*
 * A_Expr - infix, prefix, and postfix expressions
 */
type A_Expr_Kind uint

const (
	AEXPR_OP              A_Expr_Kind = iota /* normal operator */
	AEXPR_OP_ANY                             /* scalar op ANY (array) */
	AEXPR_OP_ALL                             /* scalar op ALL (array) */
	AEXPR_DISTINCT                           /* IS DISTINCT FROM - name must be "=" */
	AEXPR_NOT_DISTINCT                       /* IS NOT DISTINCT FROM - name must be "=" */
	AEXPR_NULLIF                             /* NULLIF - name must be "=" */
	AEXPR_OF                                 /* IS [NOT] OF - name must be "=" or "<>" */
	AEXPR_IN                                 /* [NOT] IN - name must be "=" or "<>" */
	AEXPR_LIKE                               /* [NOT] LIKE - name must be "~~" or "!~~" */
	AEXPR_ILIKE                              /* [NOT] ILIKE - name must be "~~*" or "!~~*" */
	AEXPR_SIMILAR                            /* [NOT] SIMILAR - name must be "~" or "!~" */
	AEXPR_BETWEEN                            /* name must be "BETWEEN" */
	AEXPR_NOT_BETWEEN                        /* name must be "NOT BETWEEN" */
	AEXPR_BETWEEN_SYM                        /* name must be "BETWEEN SYMMETRIC" */
	AEXPR_NOT_BETWEEN_SYM                    /* name must be "NOT BETWEEN SYMMETRIC" */
	AEXPR_PAREN                              /* nameless dummy node for parentheses */
)
