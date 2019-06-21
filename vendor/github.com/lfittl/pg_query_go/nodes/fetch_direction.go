// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/* ----------------------
 *		Fetch Statement (also Move)
 * ----------------------
 */
type FetchDirection uint

const (
	/* for these, howMany is how many rows to fetch; FETCH_ALL means ALL */
	FETCH_FORWARD FetchDirection = iota
	FETCH_BACKWARD

	/* for these, howMany indicates a position; only one row is fetched */
	FETCH_ABSOLUTE
	FETCH_RELATIVE
)
