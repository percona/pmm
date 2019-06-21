// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node RowExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("RowExpr")

	if len(node.Args.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Args.Fingerprint(&subCtx, node, "Args")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("args")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Colnames.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Colnames.Fingerprint(&subCtx, node, "Colnames")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("colnames")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting

	if int(node.RowFormat) != 0 {
		ctx.WriteString("row_format")
		ctx.WriteString(strconv.Itoa(int(node.RowFormat)))
	}

	if node.RowTypeid != 0 {
		ctx.WriteString("row_typeid")
		ctx.WriteString(strconv.Itoa(int(node.RowTypeid)))
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
