// Auto-generated from postgres/src/include/nodes/nodes.h - DO NOT EDIT

package pg_query

/*
 * AggStrategy -
 *	  overall execution strategies for Agg plan nodes
 *
 * This is needed in both plannodes.h and relation.h, so put it here...
 */
type AggStrategy uint

const (
	AGG_PLAIN  AggStrategy = iota /* simple agg across all input rows */
	AGG_SORTED                    /* grouped agg, input must be sorted */
	AGG_HASHED                    /* grouped agg, use internal hashtable */
	AGG_MIXED                     /* grouped agg, hash and sort both used */
)
