// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node ConstraintsSetStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("ConstraintsSetStmt")

	if len(node.Constraints.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Constraints.Fingerprint(&subCtx, node, "Constraints")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("constraints")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Deferred {
		ctx.WriteString("deferred")
		ctx.WriteString(strconv.FormatBool(node.Deferred))
	}
}
