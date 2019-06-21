// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node RangeFunction) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("RangeFunction")

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

	if len(node.Coldeflist.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Coldeflist.Fingerprint(&subCtx, node, "Coldeflist")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("coldeflist")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Functions.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Functions.Fingerprint(&subCtx, node, "Functions")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("functions")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.IsRowsfrom {
		ctx.WriteString("is_rowsfrom")
		ctx.WriteString(strconv.FormatBool(node.IsRowsfrom))
	}

	if node.Lateral {
		ctx.WriteString("lateral")
		ctx.WriteString(strconv.FormatBool(node.Lateral))
	}

	if node.Ordinality {
		ctx.WriteString("ordinality")
		ctx.WriteString(strconv.FormatBool(node.Ordinality))
	}
}
