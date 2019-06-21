// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node NamedArgExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("NamedArgExpr")

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

	if node.Argnumber != 0 {
		ctx.WriteString("argnumber")
		ctx.WriteString(strconv.Itoa(int(node.Argnumber)))
	}

	// Intentionally ignoring node.Location for fingerprinting

	if node.Name != nil {
		ctx.WriteString("name")
		ctx.WriteString(*node.Name)
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
