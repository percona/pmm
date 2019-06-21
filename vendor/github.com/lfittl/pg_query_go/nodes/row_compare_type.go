// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

/*
 * RowCompareExpr - row-wise comparison, such as (a, b) <= (1, 2)
 *
 * We support row comparison for any operator that can be determined to
 * act like =, <>, <, <=, >, or >= (we determine this by looking for the
 * operator in btree opfamilies).  Note that the same operator name might
 * map to a different operator for each pair of row elements, since the
 * element datatypes can vary.
 *
 * A RowCompareExpr node is only generated for the < <= > >= cases;
 * the = and <> cases are translated to simple AND or OR combinations
 * of the pairwise comparisons.  However, we include = and <> in the
 * RowCompareType enum for the convenience of parser logic.
 */
type RowCompareType uint

const (
	/* Values of this enum are chosen to match btree strategy numbers */
	ROWCOMPARE_LT RowCompareType = iota
	ROWCOMPARE_LE
	ROWCOMPARE_EQ
	ROWCOMPARE_GE
	ROWCOMPARE_GT
	ROWCOMPARE_NE
)
