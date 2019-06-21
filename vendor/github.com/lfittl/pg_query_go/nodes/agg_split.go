// Auto-generated from postgres/src/include/nodes/nodes.h - DO NOT EDIT

package pg_query

/* Supported operating modes (i.e., useful combinations of these options): */
type AggSplit uint

const (
	/* Basic, non-split aggregation: */
	AGGSPLIT_SIMPLE AggSplit = iota

	/* Initial phase of partial aggregation, with serialization: */
	AGGSPLIT_INITIAL_SERIAL

	/* Final phase of partial aggregation, with deserialization: */
	AGGSPLIT_FINAL_DESERIAL
)
