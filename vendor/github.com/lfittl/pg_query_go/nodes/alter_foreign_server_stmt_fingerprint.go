// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node AlterForeignServerStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterForeignServerStmt")

	if node.HasVersion {
		ctx.WriteString("has_version")
		ctx.WriteString(strconv.FormatBool(node.HasVersion))
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

	if node.Servername != nil {
		ctx.WriteString("servername")
		ctx.WriteString(*node.Servername)
	}

	if node.Version != nil {
		ctx.WriteString("version")
		ctx.WriteString(*node.Version)
	}
}
