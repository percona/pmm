// Auto-generated - DO NOT EDIT

package pg_query

func (node AlterRoleSetStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterRoleSetStmt")

	if node.Database != nil {
		ctx.WriteString("database")
		ctx.WriteString(*node.Database)
	}

	if node.Role != nil {
		subCtx := FingerprintSubContext{}
		node.Role.Fingerprint(&subCtx, node, "Role")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("role")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Setstmt != nil {
		subCtx := FingerprintSubContext{}
		node.Setstmt.Fingerprint(&subCtx, node, "Setstmt")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("setstmt")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
