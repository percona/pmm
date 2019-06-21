// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node A_Indices) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("A_Indices")

	if node.IsSlice {
		ctx.WriteString("is_slice")
		ctx.WriteString(strconv.FormatBool(node.IsSlice))
	}

	if node.Lidx != nil {
		subCtx := FingerprintSubContext{}
		node.Lidx.Fingerprint(&subCtx, node, "Lidx")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("lidx")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Uidx != nil {
		subCtx := FingerprintSubContext{}
		node.Uidx.Fingerprint(&subCtx, node, "Uidx")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("uidx")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
