// Auto-generated from postgres/src/include/access/sdir.h - DO NOT EDIT

package pg_query

/*
 * ScanDirection was an int8 for no apparent reason. I kept the original
 * values because I'm not sure if I'll break anything otherwise.  -ay 2/95
 */
type ScanDirection uint

const (
	BackwardScanDirection ScanDirection = iota
	NoMovementScanDirection
	ForwardScanDirection
)
