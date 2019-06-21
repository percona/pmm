// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node ConvertRowtypeExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("ConvertRowtypeExpr")

	if node.Arg != nil {
		subCtx := FingerprintSubContext{}
		node.Arg.Fingerprint(&subCtx, node, "Arg")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("arg")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Convertformat) != 0 {
		ctx.WriteString("convertformat")
		ctx.WriteString(strconv.Itoa(int(node.Convertformat)))
	}

	// Intentionally ignoring node.Location for fingerprinting

	if node.Resulttype != 0 {
		ctx.WriteString("resulttype")
		ctx.WriteString(strconv.Itoa(int(node.Resulttype)))
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
