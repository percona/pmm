// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node OnConflictExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("OnConflictExpr")

	if int(node.Action) != 0 {
		ctx.WriteString("action")
		ctx.WriteString(strconv.Itoa(int(node.Action)))
	}

	if len(node.ArbiterElems.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.ArbiterElems.Fingerprint(&subCtx, node, "ArbiterElems")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("arbiterElems")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.ArbiterWhere != nil {
		subCtx := FingerprintSubContext{}
		node.ArbiterWhere.Fingerprint(&subCtx, node, "ArbiterWhere")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("arbiterWhere")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Constraint != 0 {
		ctx.WriteString("constraint")
		ctx.WriteString(strconv.Itoa(int(node.Constraint)))
	}

	if node.ExclRelIndex != 0 {
		ctx.WriteString("exclRelIndex")
		ctx.WriteString(strconv.Itoa(int(node.ExclRelIndex)))
	}

	if len(node.ExclRelTlist.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.ExclRelTlist.Fingerprint(&subCtx, node, "ExclRelTlist")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("exclRelTlist")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.OnConflictSet.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.OnConflictSet.Fingerprint(&subCtx, node, "OnConflictSet")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("onConflictSet")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.OnConflictWhere != nil {
		subCtx := FingerprintSubContext{}
		node.OnConflictWhere.Fingerprint(&subCtx, node, "OnConflictWhere")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("onConflictWhere")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
