// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node AlterTableMoveAllStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterTableMoveAllStmt")

	if node.NewTablespacename != nil {
		ctx.WriteString("new_tablespacename")
		ctx.WriteString(*node.NewTablespacename)
	}

	if node.Nowait {
		ctx.WriteString("nowait")
		ctx.WriteString(strconv.FormatBool(node.Nowait))
	}

	if int(node.Objtype) != 0 {
		ctx.WriteString("objtype")
		ctx.WriteString(strconv.Itoa(int(node.Objtype)))
	}

	if node.OrigTablespacename != nil {
		ctx.WriteString("orig_tablespacename")
		ctx.WriteString(*node.OrigTablespacename)
	}

	if len(node.Roles.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Roles.Fingerprint(&subCtx, node, "Roles")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("roles")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
