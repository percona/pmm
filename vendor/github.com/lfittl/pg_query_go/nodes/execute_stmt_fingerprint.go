// Auto-generated - DO NOT EDIT

package pg_query

func (node ExecuteStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("ExecuteStmt")
	// Intentionally ignoring node.Name for fingerprinting

	if len(node.Params.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Params.Fingerprint(&subCtx, node, "Params")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("params")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
