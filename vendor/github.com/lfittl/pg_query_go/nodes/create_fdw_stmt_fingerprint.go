// Auto-generated - DO NOT EDIT

package pg_query

func (node CreateFdwStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateFdwStmt")

	if node.Fdwname != nil {
		ctx.WriteString("fdwname")
		ctx.WriteString(*node.Fdwname)
	}

	if len(node.FuncOptions.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.FuncOptions.Fingerprint(&subCtx, node, "FuncOptions")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("func_options")
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
