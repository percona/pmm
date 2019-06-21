// Auto-generated - DO NOT EDIT

package pg_query

func (node PartitionElem) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("PartitionElem")

	if len(node.Collation.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Collation.Fingerprint(&subCtx, node, "Collation")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("collation")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Expr != nil {
		subCtx := FingerprintSubContext{}
		node.Expr.Fingerprint(&subCtx, node, "Expr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("expr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting

	if node.Name != nil {
		ctx.WriteString("name")
		ctx.WriteString(*node.Name)
	}

	if len(node.Opclass.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Opclass.Fingerprint(&subCtx, node, "Opclass")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("opclass")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
