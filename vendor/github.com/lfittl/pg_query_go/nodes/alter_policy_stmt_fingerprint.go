// Auto-generated - DO NOT EDIT

package pg_query

func (node AlterPolicyStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterPolicyStmt")

	if node.PolicyName != nil {
		ctx.WriteString("policy_name")
		ctx.WriteString(*node.PolicyName)
	}

	if node.Qual != nil {
		subCtx := FingerprintSubContext{}
		node.Qual.Fingerprint(&subCtx, node, "Qual")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("qual")
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

	if node.Table != nil {
		subCtx := FingerprintSubContext{}
		node.Table.Fingerprint(&subCtx, node, "Table")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("table")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.WithCheck != nil {
		subCtx := FingerprintSubContext{}
		node.WithCheck.Fingerprint(&subCtx, node, "WithCheck")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("with_check")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
