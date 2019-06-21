// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node InsertStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("InsertStmt")

	if len(node.Cols.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Cols.Fingerprint(&subCtx, node, "Cols")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("cols")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.OnConflictClause != nil {
		subCtx := FingerprintSubContext{}
		node.OnConflictClause.Fingerprint(&subCtx, node, "OnConflictClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("onConflictClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Override) != 0 {
		ctx.WriteString("override")
		ctx.WriteString(strconv.Itoa(int(node.Override)))
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

	if len(node.ReturningList.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.ReturningList.Fingerprint(&subCtx, node, "ReturningList")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("returningList")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.SelectStmt != nil {
		subCtx := FingerprintSubContext{}
		node.SelectStmt.Fingerprint(&subCtx, node, "SelectStmt")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("selectStmt")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.WithClause != nil {
		subCtx := FingerprintSubContext{}
		node.WithClause.Fingerprint(&subCtx, node, "WithClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("withClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
