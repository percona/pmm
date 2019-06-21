// Auto-generated - DO NOT EDIT

package pg_query

func (node CreateOpFamilyStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateOpFamilyStmt")

	if node.Amname != nil {
		ctx.WriteString("amname")
		ctx.WriteString(*node.Amname)
	}

	if len(node.Opfamilyname.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Opfamilyname.Fingerprint(&subCtx, node, "Opfamilyname")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("opfamilyname")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
