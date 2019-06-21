// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node AlterOpFamilyStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterOpFamilyStmt")

	if node.Amname != nil {
		ctx.WriteString("amname")
		ctx.WriteString(*node.Amname)
	}

	if node.IsDrop {
		ctx.WriteString("isDrop")
		ctx.WriteString(strconv.FormatBool(node.IsDrop))
	}

	if len(node.Items.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Items.Fingerprint(&subCtx, node, "Items")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("items")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
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
