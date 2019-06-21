// Auto-generated - DO NOT EDIT

package pg_query

func (node A_ArrayExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("A_ArrayExpr")

	if len(node.Elements.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Elements.Fingerprint(&subCtx, node, "Elements")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("elements")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting
}
