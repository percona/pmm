// Auto-generated - DO NOT EDIT

package pg_query

func (node CollateClause) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CollateClause")

	if node.Arg != nil {
		subCtx := FingerprintSubContext{}
		node.Arg.Fingerprint(&subCtx, node, "Arg")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("arg")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Collname.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Collname.Fingerprint(&subCtx, node, "Collname")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("collname")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting
}
