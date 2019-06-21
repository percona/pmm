// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/*
 * WithCheckOption -
 *		representation of WITH CHECK OPTION checks to be applied to new tuples
 *		when inserting/updating an auto-updatable view, or RLS WITH CHECK
 *		policies to be applied when inserting/updating a relation with RLS.
 */
type WCOKind uint

const (
	WCO_VIEW_CHECK         WCOKind = iota /* WCO on an auto-updatable view */
	WCO_RLS_INSERT_CHECK                  /* RLS INSERT WITH CHECK policy */
	WCO_RLS_UPDATE_CHECK                  /* RLS UPDATE WITH CHECK policy */
	WCO_RLS_CONFLICT_CHECK                /* RLS ON CONFLICT DO UPDATE USING policy */
)
