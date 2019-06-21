// Auto-generated from postgres/src/include/postgres.h - DO NOT EDIT

package pg_query

/*
 * Type tag for the various sorts of "TOAST pointer" datums.  The peculiar
 * value for VARTAG_ONDISK comes from a requirement for on-disk compatibility
 * with a previous notion that the tag field was the pointer datum's length.
 */
type vartag_external uint

const (
	VARTAG_INDIRECT vartag_external = iota
	VARTAG_EXPANDED_RO
	VARTAG_EXPANDED_RW
	VARTAG_ONDISK
)
