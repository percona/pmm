// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node RefreshMatViewStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("RefreshMatViewStmt")

	if node.Concurrent {
		ctx.WriteString("concurrent")
		ctx.WriteString(strconv.FormatBool(node.Concurrent))
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

	if node.SkipData {
		ctx.WriteString("skipData")
		ctx.WriteString(strconv.FormatBool(node.SkipData))
	}
}
