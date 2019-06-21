// Auto-generated - DO NOT EDIT

package pg_query

func (node RawStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("RawStmt")

	if node.Stmt != nil {
		subCtx := FingerprintSubContext{}
		node.Stmt.Fingerprint(&subCtx, node, "Stmt")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("stmt")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.StmtLen for fingerprinting

	// Intentionally ignoring node.StmtLocation for fingerprinting
}
