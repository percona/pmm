// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node TransactionStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("TransactionStmt")
	// Intentionally ignoring node.Gid for fingerprinting

	if int(node.Kind) != 0 {
		ctx.WriteString("kind")
		ctx.WriteString(strconv.Itoa(int(node.Kind)))
	}

	// Intentionally ignoring node.Options for fingerprinting
}
