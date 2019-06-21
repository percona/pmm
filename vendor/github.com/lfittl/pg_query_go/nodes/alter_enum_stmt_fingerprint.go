// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node AlterEnumStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterEnumStmt")

	if node.NewVal != nil {
		ctx.WriteString("newVal")
		ctx.WriteString(*node.NewVal)
	}

	if node.NewValIsAfter {
		ctx.WriteString("newValIsAfter")
		ctx.WriteString(strconv.FormatBool(node.NewValIsAfter))
	}

	if node.NewValNeighbor != nil {
		ctx.WriteString("newValNeighbor")
		ctx.WriteString(*node.NewValNeighbor)
	}

	if node.OldVal != nil {
		ctx.WriteString("oldVal")
		ctx.WriteString(*node.OldVal)
	}

	if node.SkipIfNewValExists {
		ctx.WriteString("skipIfNewValExists")
		ctx.WriteString(strconv.FormatBool(node.SkipIfNewValExists))
	}

	if len(node.TypeName.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.TypeName.Fingerprint(&subCtx, node, "TypeName")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("typeName")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
