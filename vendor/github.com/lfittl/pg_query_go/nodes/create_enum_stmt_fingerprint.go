// Auto-generated - DO NOT EDIT

package pg_query

func (node CreateEnumStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateEnumStmt")

	if len(node.TypeName.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.TypeName.Fingerprint(&subCtx, node, "TypeName")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("typeName")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Vals.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Vals.Fingerprint(&subCtx, node, "Vals")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("vals")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
