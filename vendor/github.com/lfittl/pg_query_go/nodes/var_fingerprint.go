// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node Var) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("Var")
	// Intentionally ignoring node.Location for fingerprinting

	if node.Varattno != 0 {
		ctx.WriteString("varattno")
		ctx.WriteString(strconv.Itoa(int(node.Varattno)))
	}

	if node.Varcollid != 0 {
		ctx.WriteString("varcollid")
		ctx.WriteString(strconv.Itoa(int(node.Varcollid)))
	}

	if node.Varlevelsup != 0 {
		ctx.WriteString("varlevelsup")
		ctx.WriteString(strconv.Itoa(int(node.Varlevelsup)))
	}

	if node.Varno != 0 {
		ctx.WriteString("varno")
		ctx.WriteString(strconv.Itoa(int(node.Varno)))
	}

	if node.Varnoold != 0 {
		ctx.WriteString("varnoold")
		ctx.WriteString(strconv.Itoa(int(node.Varnoold)))
	}

	if node.Varoattno != 0 {
		ctx.WriteString("varoattno")
		ctx.WriteString(strconv.Itoa(int(node.Varoattno)))
	}

	if node.Vartype != 0 {
		ctx.WriteString("vartype")
		ctx.WriteString(strconv.Itoa(int(node.Vartype)))
	}

	if node.Vartypmod != 0 {
		ctx.WriteString("vartypmod")
		ctx.WriteString(strconv.Itoa(int(node.Vartypmod)))
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
