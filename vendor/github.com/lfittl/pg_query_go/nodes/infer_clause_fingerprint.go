// Auto-generated - DO NOT EDIT

package pg_query

func (node InferClause) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("InferClause")

	if node.Conname != nil {
		ctx.WriteString("conname")
		ctx.WriteString(*node.Conname)
	}

	if len(node.IndexElems.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.IndexElems.Fingerprint(&subCtx, node, "IndexElems")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("indexElems")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting

	if node.WhereClause != nil {
		subCtx := FingerprintSubContext{}
		node.WhereClause.Fingerprint(&subCtx, node, "WhereClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("whereClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
