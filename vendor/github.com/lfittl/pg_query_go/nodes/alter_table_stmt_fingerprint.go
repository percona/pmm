// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node AlterTableStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterTableStmt")

	if len(node.Cmds.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Cmds.Fingerprint(&subCtx, node, "Cmds")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("cmds")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.MissingOk {
		ctx.WriteString("missing_ok")
		ctx.WriteString(strconv.FormatBool(node.MissingOk))
	}

	if node.Relation != nil {
		subCtx := FingerprintSubContext{}
		node.Relation.Fingerprint(&subCtx, node, "Relation")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("relation")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Relkind) != 0 {
		ctx.WriteString("relkind")
		ctx.WriteString(strconv.Itoa(int(node.Relkind)))
	}
}
