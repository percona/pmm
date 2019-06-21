// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node DropRoleStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("DropRoleStmt")

	if node.MissingOk {
		ctx.WriteString("missing_ok")
		ctx.WriteString(strconv.FormatBool(node.MissingOk))
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
