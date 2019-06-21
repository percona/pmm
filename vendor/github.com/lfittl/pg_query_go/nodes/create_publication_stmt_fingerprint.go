// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CreatePublicationStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreatePublicationStmt")

	if node.ForAllTables {
		ctx.WriteString("for_all_tables")
		ctx.WriteString(strconv.FormatBool(node.ForAllTables))
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

	if node.Pubname != nil {
		ctx.WriteString("pubname")
		ctx.WriteString(*node.Pubname)
	}

	if len(node.Tables.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Tables.Fingerprint(&subCtx, node, "Tables")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("tables")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
