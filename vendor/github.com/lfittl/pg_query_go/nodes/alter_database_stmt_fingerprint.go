// Auto-generated - DO NOT EDIT

package pg_query

func (node AlterDatabaseStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterDatabaseStmt")

	if node.Dbname != nil {
		ctx.WriteString("dbname")
		ctx.WriteString(*node.Dbname)
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
