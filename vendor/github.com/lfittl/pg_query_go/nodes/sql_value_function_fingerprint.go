// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node SQLValueFunction) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("SQLValueFunction")
	// Intentionally ignoring node.Location for fingerprinting

	if int(node.Op) != 0 {
		ctx.WriteString("op")
		ctx.WriteString(strconv.Itoa(int(node.Op)))
	}

	if node.Type != 0 {
		ctx.WriteString("type")
		ctx.WriteString(strconv.Itoa(int(node.Type)))
	}

	if node.Typmod != 0 {
		ctx.WriteString("typmod")
		ctx.WriteString(strconv.Itoa(int(node.Typmod)))
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
