// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CreateRoleStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateRoleStmt")

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

	if node.Role != nil {
		ctx.WriteString("role")
		ctx.WriteString(*node.Role)
	}

	if int(node.StmtType) != 0 {
		ctx.WriteString("stmt_type")
		ctx.WriteString(strconv.Itoa(int(node.StmtType)))
	}
}
