// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node DeclareCursorStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("DeclareCursorStmt")

	if node.Options != 0 {
		ctx.WriteString("options")
		ctx.WriteString(strconv.Itoa(int(node.Options)))
	}

	// Intentionally ignoring node.Portalname for fingerprinting

	if node.Query != nil {
		subCtx := FingerprintSubContext{}
		node.Query.Fingerprint(&subCtx, node, "Query")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("query")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
