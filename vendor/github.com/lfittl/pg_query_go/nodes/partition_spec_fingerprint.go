// Auto-generated - DO NOT EDIT

package pg_query

func (node PartitionSpec) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("PartitionSpec")
	// Intentionally ignoring node.Location for fingerprinting

	if len(node.PartParams.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.PartParams.Fingerprint(&subCtx, node, "PartParams")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("partParams")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Strategy != nil {
		ctx.WriteString("strategy")
		ctx.WriteString(*node.Strategy)
	}
}
