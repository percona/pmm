// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node ArrayRef) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("ArrayRef")

	if node.Refarraytype != 0 {
		ctx.WriteString("refarraytype")
		ctx.WriteString(strconv.Itoa(int(node.Refarraytype)))
	}

	if node.Refassgnexpr != nil {
		subCtx := FingerprintSubContext{}
		node.Refassgnexpr.Fingerprint(&subCtx, node, "Refassgnexpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("refassgnexpr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Refcollid != 0 {
		ctx.WriteString("refcollid")
		ctx.WriteString(strconv.Itoa(int(node.Refcollid)))
	}

	if node.Refelemtype != 0 {
		ctx.WriteString("refelemtype")
		ctx.WriteString(strconv.Itoa(int(node.Refelemtype)))
	}

	if node.Refexpr != nil {
		subCtx := FingerprintSubContext{}
		node.Refexpr.Fingerprint(&subCtx, node, "Refexpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("refexpr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Reflowerindexpr.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Reflowerindexpr.Fingerprint(&subCtx, node, "Reflowerindexpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("reflowerindexpr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Reftypmod != 0 {
		ctx.WriteString("reftypmod")
		ctx.WriteString(strconv.Itoa(int(node.Reftypmod)))
	}

	if len(node.Refupperindexpr.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Refupperindexpr.Fingerprint(&subCtx, node, "Refupperindexpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("refupperindexpr")
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
