// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node AlterRoleStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterRoleStmt")

	if node.Action != 0 {
		ctx.WriteString("action")
		ctx.WriteString(strconv.Itoa(int(node.Action)))
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

	if node.Role != nil {
		subCtx := FingerprintSubContext{}
		node.Role.Fingerprint(&subCtx, node, "Role")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("role")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
