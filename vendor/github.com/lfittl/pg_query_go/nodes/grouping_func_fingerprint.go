// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node GroupingFunc) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("GroupingFunc")

	if node.Agglevelsup != 0 {
		ctx.WriteString("agglevelsup")
		ctx.WriteString(strconv.Itoa(int(node.Agglevelsup)))
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

	if len(node.Cols.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Cols.Fingerprint(&subCtx, node, "Cols")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("cols")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting

	if len(node.Refs.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Refs.Fingerprint(&subCtx, node, "Refs")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("refs")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
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
