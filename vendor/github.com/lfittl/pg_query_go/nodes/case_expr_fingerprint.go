// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CaseExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CaseExpr")

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

	if node.Casecollid != 0 {
		ctx.WriteString("casecollid")
		ctx.WriteString(strconv.Itoa(int(node.Casecollid)))
	}

	if node.Casetype != 0 {
		ctx.WriteString("casetype")
		ctx.WriteString(strconv.Itoa(int(node.Casetype)))
	}

	if node.Defresult != nil {
		subCtx := FingerprintSubContext{}
		node.Defresult.Fingerprint(&subCtx, node, "Defresult")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("defresult")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
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
