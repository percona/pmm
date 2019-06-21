// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node BlockIdData) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("BlockIdData")

	if node.BiHi != 0 {
		ctx.WriteString("bi_hi")
		ctx.WriteString(strconv.Itoa(int(node.BiHi)))
	}

	if node.BiLo != 0 {
		ctx.WriteString("bi_lo")
		ctx.WriteString(strconv.Itoa(int(node.BiLo)))
	}
}
