// Auto-generated - DO NOT EDIT

package pg_query

func (node AlterDatabaseSetStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterDatabaseSetStmt")

	if node.Dbname != nil {
		ctx.WriteString("dbname")
		ctx.WriteString(*node.Dbname)
	}

	if node.Setstmt != nil {
		subCtx := FingerprintSubContext{}
		node.Setstmt.Fingerprint(&subCtx, node, "Setstmt")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("setstmt")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
