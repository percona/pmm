// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node AlterOwnerStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterOwnerStmt")

	if node.Newowner != nil {
		subCtx := FingerprintSubContext{}
		node.Newowner.Fingerprint(&subCtx, node, "Newowner")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("newowner")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Object != nil {
		subCtx := FingerprintSubContext{}
		node.Object.Fingerprint(&subCtx, node, "Object")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("object")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.ObjectType) != 0 {
		ctx.WriteString("objectType")
		ctx.WriteString(strconv.Itoa(int(node.ObjectType)))
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
