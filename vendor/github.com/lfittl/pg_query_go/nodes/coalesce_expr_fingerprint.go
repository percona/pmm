// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CoalesceExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CoalesceExpr")

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

	if node.Coalescecollid != 0 {
		ctx.WriteString("coalescecollid")
		ctx.WriteString(strconv.Itoa(int(node.Coalescecollid)))
	}

	if node.Coalescetype != 0 {
		ctx.WriteString("coalescetype")
		ctx.WriteString(strconv.Itoa(int(node.Coalescetype)))
	}

	// Intentionally ignoring node.Location for fingerprinting

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
