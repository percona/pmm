// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node RangeTblFunction) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("RangeTblFunction")

	if len(node.Funccolcollations.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Funccolcollations.Fingerprint(&subCtx, node, "Funccolcollations")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("funccolcollations")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Funccolcount != 0 {
		ctx.WriteString("funccolcount")
		ctx.WriteString(strconv.Itoa(int(node.Funccolcount)))
	}

	if len(node.Funccolnames.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Funccolnames.Fingerprint(&subCtx, node, "Funccolnames")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("funccolnames")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Funccoltypes.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Funccoltypes.Fingerprint(&subCtx, node, "Funccoltypes")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("funccoltypes")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Funccoltypmods.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Funccoltypmods.Fingerprint(&subCtx, node, "Funccoltypmods")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("funccoltypmods")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Funcexpr != nil {
		subCtx := FingerprintSubContext{}
		node.Funcexpr.Fingerprint(&subCtx, node, "Funcexpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("funcexpr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	ctx.WriteString("funcparams")
	for _, val := range node.Funcparams {
		ctx.WriteString(strconv.Itoa(int(val)))
	}
}
