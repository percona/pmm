// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node AlterTableSpaceOptionsStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterTableSpaceOptionsStmt")

	if node.IsReset {
		ctx.WriteString("isReset")
		ctx.WriteString(strconv.FormatBool(node.IsReset))
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

	if node.Tablespacename != nil {
		ctx.WriteString("tablespacename")
		ctx.WriteString(*node.Tablespacename)
	}
}
