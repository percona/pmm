// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node WithClause) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("WithClause")

	if len(node.Ctes.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Ctes.Fingerprint(&subCtx, node, "Ctes")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("ctes")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting

	if node.Recursive {
		ctx.WriteString("recursive")
		ctx.WriteString(strconv.FormatBool(node.Recursive))
	}
}
