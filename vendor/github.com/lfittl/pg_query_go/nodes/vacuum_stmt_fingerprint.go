// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node VacuumStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("VacuumStmt")

	if node.Options != 0 {
		ctx.WriteString("options")
		ctx.WriteString(strconv.Itoa(int(node.Options)))
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

	if len(node.VaCols.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.VaCols.Fingerprint(&subCtx, node, "VaCols")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("va_cols")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
