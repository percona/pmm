// Auto-generated from postgres/src/include/nodes/nodes.h - DO NOT EDIT

package pg_query

/*
 * JoinType -
 *	  enums for types of relation joins
 *
 * JoinType determines the exact semantics of joining two relations using
 * a matching qualification.  For example, it tells what to do with a tuple
 * that has no match in the other relation.
 *
 * This is needed in both parsenodes.h and plannodes.h, so put it here...
 */
type JoinType uint

const (
	/*
	 * The canonical kinds of joins according to the SQL JOIN syntax. Only
	 * these codes can appear in parser output (e.g., JoinExpr nodes).
	 */
	JOIN_INNER JoinType = iota /* matching tuple pairs only */
	JOIN_LEFT                  /* pairs + unmatched LHS tuples */
	JOIN_FULL                  /* pairs + unmatched LHS + unmatched RHS */
	JOIN_RIGHT                 /* pairs + unmatched RHS tuples */

	/*
	 * Semijoins and anti-semijoins (as defined in relational theory) do not
	 * appear in the SQL JOIN syntax, but there are standard idioms for
	 * representing them (e.g., using EXISTS).  The planner recognizes these
	 * cases and converts them to joins.  So the planner and executor must
	 * support these codes.  NOTE: in JOIN_SEMI output, it is unspecified
	 * which matching RHS row is joined to.  In JOIN_ANTI output, the row is
	 * guaranteed to be null-extended.
	 */
	JOIN_SEMI /* 1 copy of each LHS row that has match(es) */
	JOIN_ANTI /* 1 copy of each LHS row that has no match */

	/*
	 * These codes are used internally in the planner, but are not supported
	 * by the executor (nor, indeed, by most of the planner).
	 */
	JOIN_UNIQUE_OUTER /* LHS path must be made unique */
	JOIN_UNIQUE_INNER /* RHS path must be made unique */

	/*
	 * We might need additional join types someday.
	 */
)
