// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node NextValueExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("NextValueExpr")

	if node.Seqid != 0 {
		ctx.WriteString("seqid")
		ctx.WriteString(strconv.Itoa(int(node.Seqid)))
	}

	if node.TypeId != 0 {
		ctx.WriteString("typeId")
		ctx.WriteString(strconv.Itoa(int(node.TypeId)))
	}

	if node.Xpr != nil {
		subCtx := FingerprintSubContext{}
		node.Xpr.Fingerprint(&subCtx, node, "Xpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("xpr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
