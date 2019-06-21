// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node NullTest) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("NullTest")

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

	if node.Argisrow {
		ctx.WriteString("argisrow")
		ctx.WriteString(strconv.FormatBool(node.Argisrow))
	}

	// Intentionally ignoring node.Location for fingerprinting

	if int(node.Nulltesttype) != 0 {
		ctx.WriteString("nulltesttype")
		ctx.WriteString(strconv.Itoa(int(node.Nulltesttype)))
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
