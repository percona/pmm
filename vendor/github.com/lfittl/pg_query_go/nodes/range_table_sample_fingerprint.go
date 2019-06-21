// Auto-generated - DO NOT EDIT

package pg_query

func (node RangeTableSample) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("RangeTableSample")

	if len(node.Args.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Args.Fingerprint(&subCtx, node, "Args")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("args")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting

	if len(node.Method.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Method.Fingerprint(&subCtx, node, "Method")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("method")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Relation != nil {
		subCtx := FingerprintSubContext{}
		node.Relation.Fingerprint(&subCtx, node, "Relation")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("relation")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Repeatable != nil {
		subCtx := FingerprintSubContext{}
		node.Repeatable.Fingerprint(&subCtx, node, "Repeatable")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("repeatable")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
