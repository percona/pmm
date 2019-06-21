// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node XmlSerialize) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("XmlSerialize")

	if node.Expr != nil {
		subCtx := FingerprintSubContext{}
		node.Expr.Fingerprint(&subCtx, node, "Expr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("expr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting

	if node.TypeName != nil {
		subCtx := FingerprintSubContext{}
		node.TypeName.Fingerprint(&subCtx, node, "TypeName")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("typeName")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Xmloption) != 0 {
		ctx.WriteString("xmloption")
		ctx.WriteString(strconv.Itoa(int(node.Xmloption)))
	}
}
