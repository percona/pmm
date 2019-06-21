// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node TableLikeClause) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("TableLikeClause")

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
}
