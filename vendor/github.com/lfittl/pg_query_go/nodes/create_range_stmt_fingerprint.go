// Auto-generated - DO NOT EDIT

package pg_query

func (node CreateRangeStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateRangeStmt")

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
}
