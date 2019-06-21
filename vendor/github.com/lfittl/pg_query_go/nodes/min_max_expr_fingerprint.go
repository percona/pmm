// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node MinMaxExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("MinMaxExpr")

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

	if node.Inputcollid != 0 {
		ctx.WriteString("inputcollid")
		ctx.WriteString(strconv.Itoa(int(node.Inputcollid)))
	}

	// Intentionally ignoring node.Location for fingerprinting

	if node.Minmaxcollid != 0 {
		ctx.WriteString("minmaxcollid")
		ctx.WriteString(strconv.Itoa(int(node.Minmaxcollid)))
	}

	if node.Minmaxtype != 0 {
		ctx.WriteString("minmaxtype")
		ctx.WriteString(strconv.Itoa(int(node.Minmaxtype)))
	}

	if int(node.Op) != 0 {
		ctx.WriteString("op")
		ctx.WriteString(strconv.Itoa(int(node.Op)))
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
