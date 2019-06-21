// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node RangeTableFunc) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("RangeTableFunc")

	if node.Alias != nil {
		subCtx := FingerprintSubContext{}
		node.Alias.Fingerprint(&subCtx, node, "Alias")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("alias")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Columns.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Columns.Fingerprint(&subCtx, node, "Columns")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("columns")
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

	if node.Lateral {
		ctx.WriteString("lateral")
		ctx.WriteString(strconv.FormatBool(node.Lateral))
	}

	// Intentionally ignoring node.Location for fingerprinting

	if len(node.Namespaces.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Namespaces.Fingerprint(&subCtx, node, "Namespaces")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("namespaces")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
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
