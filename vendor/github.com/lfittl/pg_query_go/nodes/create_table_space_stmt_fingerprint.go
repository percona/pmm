// Auto-generated - DO NOT EDIT

package pg_query

func (node CreateTableSpaceStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateTableSpaceStmt")
	// Intentionally ignoring node.Location for fingerprinting

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

	if node.Owner != nil {
		subCtx := FingerprintSubContext{}
		node.Owner.Fingerprint(&subCtx, node, "Owner")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("owner")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Tablespacename != nil {
		ctx.WriteString("tablespacename")
		ctx.WriteString(*node.Tablespacename)
	}
}
