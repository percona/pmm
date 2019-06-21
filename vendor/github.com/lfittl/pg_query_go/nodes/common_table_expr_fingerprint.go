// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CommonTableExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CommonTableExpr")

	if len(node.Aliascolnames.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Aliascolnames.Fingerprint(&subCtx, node, "Aliascolnames")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("aliascolnames")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Ctecolcollations.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Ctecolcollations.Fingerprint(&subCtx, node, "Ctecolcollations")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("ctecolcollations")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Ctecolnames.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Ctecolnames.Fingerprint(&subCtx, node, "Ctecolnames")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("ctecolnames")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Ctecoltypes.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Ctecoltypes.Fingerprint(&subCtx, node, "Ctecoltypes")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("ctecoltypes")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Ctecoltypmods.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Ctecoltypmods.Fingerprint(&subCtx, node, "Ctecoltypmods")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("ctecoltypmods")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Ctename != nil {
		ctx.WriteString("ctename")
		ctx.WriteString(*node.Ctename)
	}

	if node.Ctequery != nil {
		subCtx := FingerprintSubContext{}
		node.Ctequery.Fingerprint(&subCtx, node, "Ctequery")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("ctequery")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Cterecursive {
		ctx.WriteString("cterecursive")
		ctx.WriteString(strconv.FormatBool(node.Cterecursive))
	}

	if node.Cterefcount != 0 {
		ctx.WriteString("cterefcount")
		ctx.WriteString(strconv.Itoa(int(node.Cterefcount)))
	}

	// Intentionally ignoring node.Location for fingerprinting
}
