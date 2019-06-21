// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node FetchStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("FetchStmt")

	if int(node.Direction) != 0 {
		ctx.WriteString("direction")
		ctx.WriteString(strconv.Itoa(int(node.Direction)))
	}

	if node.HowMany != 0 {
		ctx.WriteString("howMany")
		ctx.WriteString(strconv.Itoa(int(node.HowMany)))
	}

	if node.Ismove {
		ctx.WriteString("ismove")
		ctx.WriteString(strconv.FormatBool(node.Ismove))
	}

	// Intentionally ignoring node.Portalname for fingerprinting
}
