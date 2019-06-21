// Auto-generated - DO NOT EDIT

package pg_query

func (node CreateAmStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateAmStmt")

	if node.Amname != nil {
		ctx.WriteString("amname")
		ctx.WriteString(*node.Amname)
	}

	if node.Amtype != 0 {
		ctx.WriteString("amtype")
		ctx.WriteString(string(node.Amtype))

	}

	if len(node.HandlerName.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.HandlerName.Fingerprint(&subCtx, node, "HandlerName")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("handler_name")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
