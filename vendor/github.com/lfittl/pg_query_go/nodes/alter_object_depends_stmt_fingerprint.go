// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node AlterObjectDependsStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterObjectDependsStmt")

	if node.Extname != nil {
		subCtx := FingerprintSubContext{}
		node.Extname.Fingerprint(&subCtx, node, "Extname")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("extname")
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
