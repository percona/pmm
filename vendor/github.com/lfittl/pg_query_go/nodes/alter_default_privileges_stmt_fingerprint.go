// Auto-generated - DO NOT EDIT

package pg_query

func (node AlterDefaultPrivilegesStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterDefaultPrivilegesStmt")

	if node.Action != nil {
		subCtx := FingerprintSubContext{}
		node.Action.Fingerprint(&subCtx, node, "Action")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("action")
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
