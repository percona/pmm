// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node FuncExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("FuncExpr")

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

	if node.Funccollid != 0 {
		ctx.WriteString("funccollid")
		ctx.WriteString(strconv.Itoa(int(node.Funccollid)))
	}

	if int(node.Funcformat) != 0 {
		ctx.WriteString("funcformat")
		ctx.WriteString(strconv.Itoa(int(node.Funcformat)))
	}

	if node.Funcid != 0 {
		ctx.WriteString("funcid")
		ctx.WriteString(strconv.Itoa(int(node.Funcid)))
	}

	if node.Funcresulttype != 0 {
		ctx.WriteString("funcresulttype")
		ctx.WriteString(strconv.Itoa(int(node.Funcresulttype)))
	}

	if node.Funcretset {
		ctx.WriteString("funcretset")
		ctx.WriteString(strconv.FormatBool(node.Funcretset))
	}

	if node.Funcvariadic {
		ctx.WriteString("funcvariadic")
		ctx.WriteString(strconv.FormatBool(node.Funcvariadic))
	}

	if node.Inputcollid != 0 {
		ctx.WriteString("inputcollid")
		ctx.WriteString(strconv.Itoa(int(node.Inputcollid)))
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
