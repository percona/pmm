// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node DropUserMappingStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("DropUserMappingStmt")

	if node.MissingOk {
		ctx.WriteString("missing_ok")
		ctx.WriteString(strconv.FormatBool(node.MissingOk))
	}

	if node.Servername != nil {
		ctx.WriteString("servername")
		ctx.WriteString(*node.Servername)
	}

	if node.User != nil {
		subCtx := FingerprintSubContext{}
		node.User.Fingerprint(&subCtx, node, "User")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("user")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
