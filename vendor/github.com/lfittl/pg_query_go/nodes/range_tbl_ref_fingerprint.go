// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node RangeTblRef) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("RangeTblRef")

	if node.Rtindex != 0 {
		ctx.WriteString("rtindex")
		ctx.WriteString(strconv.Itoa(int(node.Rtindex)))
	}
}
