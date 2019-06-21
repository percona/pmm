// Auto-generated - DO NOT EDIT

package pg_query

func (node AlterTSDictionaryStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterTSDictionaryStmt")

	if len(node.Dictname.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Dictname.Fingerprint(&subCtx, node, "Dictname")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("dictname")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Options.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Options.Fingerprint(&subCtx, node, "Options")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("options")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
