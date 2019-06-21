// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node FunctionParameter) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("FunctionParameter")

	if node.ArgType != nil {
		subCtx := FingerprintSubContext{}
		node.ArgType.Fingerprint(&subCtx, node, "ArgType")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("argType")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Defexpr != nil {
		subCtx := FingerprintSubContext{}
		node.Defexpr.Fingerprint(&subCtx, node, "Defexpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("defexpr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Mode) != 0 {
		ctx.WriteString("mode")
		ctx.WriteString(strconv.Itoa(int(node.Mode)))
	}

	if node.Name != nil {
		ctx.WriteString("name")
		ctx.WriteString(*node.Name)
	}
}
