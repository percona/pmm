// Auto-generated - DO NOT EDIT

package pg_query

func (node AlterExtensionStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterExtensionStmt")

	if node.Extname != nil {
		ctx.WriteString("extname")
		ctx.WriteString(*node.Extname)
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
