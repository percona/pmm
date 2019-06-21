// Auto-generated - DO NOT EDIT

package pg_query

func (node AlterCollationStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterCollationStmt")

	if len(node.Collname.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Collname.Fingerprint(&subCtx, node, "Collname")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("collname")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
