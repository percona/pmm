// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node ClusterStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("ClusterStmt")

	if node.Indexname != nil {
		ctx.WriteString("indexname")
		ctx.WriteString(*node.Indexname)
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

	if node.Verbose {
		ctx.WriteString("verbose")
		ctx.WriteString(strconv.FormatBool(node.Verbose))
	}
}
