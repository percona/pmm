// Auto-generated - DO NOT EDIT

package pg_query

func (node AlterOperatorStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterOperatorStmt")

	if node.Opername != nil {
		subCtx := FingerprintSubContext{}
		node.Opername.Fingerprint(&subCtx, node, "Opername")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("opername")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Options.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Options.Fingerprint(&subCtx, node, "Options")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("options")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
