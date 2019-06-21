// Auto-generated - DO NOT EDIT

package pg_query

func (node PartitionBoundSpec) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("PartitionBoundSpec")

	if len(node.Listdatums.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Listdatums.Fingerprint(&subCtx, node, "Listdatums")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("listdatums")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting

	if len(node.Lowerdatums.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Lowerdatums.Fingerprint(&subCtx, node, "Lowerdatums")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("lowerdatums")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Strategy != 0 {
		ctx.WriteString("strategy")
		ctx.WriteString(string(node.Strategy))

	}

	if len(node.Upperdatums.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Upperdatums.Fingerprint(&subCtx, node, "Upperdatums")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("upperdatums")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
