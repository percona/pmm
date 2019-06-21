// Auto-generated - DO NOT EDIT

package pg_query

func (node ReassignOwnedStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("ReassignOwnedStmt")

	if node.Newrole != nil {
		subCtx := FingerprintSubContext{}
		node.Newrole.Fingerprint(&subCtx, node, "Newrole")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("newrole")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Roles.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Roles.Fingerprint(&subCtx, node, "Roles")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("roles")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
