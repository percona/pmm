// Auto-generated - DO NOT EDIT

package pg_query

func (node PartitionCmd) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("PartitionCmd")

	if node.Bound != nil {
		subCtx := FingerprintSubContext{}
		node.Bound.Fingerprint(&subCtx, node, "Bound")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("bound")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Name != nil {
		subCtx := FingerprintSubContext{}
		node.Name.Fingerprint(&subCtx, node, "Name")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("name")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
