// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node IndexElem) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("IndexElem")

	if len(node.Collation.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Collation.Fingerprint(&subCtx, node, "Collation")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("collation")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Expr != nil {
		subCtx := FingerprintSubContext{}
		node.Expr.Fingerprint(&subCtx, node, "Expr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("expr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Indexcolname != nil {
		ctx.WriteString("indexcolname")
		ctx.WriteString(*node.Indexcolname)
	}

	if node.Name != nil {
		ctx.WriteString("name")
		ctx.WriteString(*node.Name)
	}

	if int(node.NullsOrdering) != 0 {
		ctx.WriteString("nulls_ordering")
		ctx.WriteString(strconv.Itoa(int(node.NullsOrdering)))
	}

	if len(node.Opclass.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Opclass.Fingerprint(&subCtx, node, "Opclass")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("opclass")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Ordering) != 0 {
		ctx.WriteString("ordering")
		ctx.WriteString(strconv.Itoa(int(node.Ordering)))
	}
}
