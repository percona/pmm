// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node TableFunc) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("TableFunc")

	if len(node.Colcollations.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Colcollations.Fingerprint(&subCtx, node, "Colcollations")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("colcollations")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Coldefexprs.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Coldefexprs.Fingerprint(&subCtx, node, "Coldefexprs")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("coldefexprs")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Colexprs.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Colexprs.Fingerprint(&subCtx, node, "Colexprs")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("colexprs")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Colnames.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Colnames.Fingerprint(&subCtx, node, "Colnames")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("colnames")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Coltypes.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Coltypes.Fingerprint(&subCtx, node, "Coltypes")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("coltypes")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Coltypmods.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Coltypmods.Fingerprint(&subCtx, node, "Coltypmods")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("coltypmods")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Docexpr != nil {
		subCtx := FingerprintSubContext{}
		node.Docexpr.Fingerprint(&subCtx, node, "Docexpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("docexpr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
	// Intentionally ignoring node.Location for fingerprinting

	ctx.WriteString("notnulls")
	for _, val := range node.Notnulls {
		ctx.WriteString(strconv.Itoa(int(val)))
	}

	if len(node.NsNames.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.NsNames.Fingerprint(&subCtx, node, "NsNames")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("ns_names")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.NsUris.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.NsUris.Fingerprint(&subCtx, node, "NsUris")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("ns_uris")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Ordinalitycol != 0 {
		ctx.WriteString("ordinalitycol")
		ctx.WriteString(strconv.Itoa(int(node.Ordinalitycol)))
	}

	if node.Rowexpr != nil {
		subCtx := FingerprintSubContext{}
		node.Rowexpr.Fingerprint(&subCtx, node, "Rowexpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("rowexpr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
