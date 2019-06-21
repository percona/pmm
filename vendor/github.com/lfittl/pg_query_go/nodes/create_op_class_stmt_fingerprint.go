// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CreateOpClassStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateOpClassStmt")

	if node.Amname != nil {
		ctx.WriteString("amname")
		ctx.WriteString(*node.Amname)
	}

	if node.Datatype != nil {
		subCtx := FingerprintSubContext{}
		node.Datatype.Fingerprint(&subCtx, node, "Datatype")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("datatype")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.IsDefault {
		ctx.WriteString("isDefault")
		ctx.WriteString(strconv.FormatBool(node.IsDefault))
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

	if len(node.Opclassname.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Opclassname.Fingerprint(&subCtx, node, "Opclassname")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("opclassname")
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
