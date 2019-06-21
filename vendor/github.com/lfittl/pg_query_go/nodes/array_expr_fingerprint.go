// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node ArrayExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("ArrayExpr")

	if node.ArrayCollid != 0 {
		ctx.WriteString("array_collid")
		ctx.WriteString(strconv.Itoa(int(node.ArrayCollid)))
	}

	if node.ArrayTypeid != 0 {
		ctx.WriteString("array_typeid")
		ctx.WriteString(strconv.Itoa(int(node.ArrayTypeid)))
	}

	if node.ElementTypeid != 0 {
		ctx.WriteString("element_typeid")
		ctx.WriteString(strconv.Itoa(int(node.ElementTypeid)))
	}

	if len(node.Elements.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Elements.Fingerprint(&subCtx, node, "Elements")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("elements")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting

	if node.Multidims {
		ctx.WriteString("multidims")
		ctx.WriteString(strconv.FormatBool(node.Multidims))
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
