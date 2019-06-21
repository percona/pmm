// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

/*
 * PartitionRangeDatum - one of the values in a range partition bound
 *
 * This can be MINVALUE, MAXVALUE or a specific bounded value.
 */
type PartitionRangeDatumKind uint

const (
	PARTITION_RANGE_DATUM_MINVALUE PartitionRangeDatumKind = iota
	PARTITION_RANGE_DATUM_VALUE
	PARTITION_RANGE_DATUM_MAXVALUE
)
