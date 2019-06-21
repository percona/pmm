// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node RangeSubselect) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("RangeSubselect")

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

	if node.Lateral {
		ctx.WriteString("lateral")
		ctx.WriteString(strconv.FormatBool(node.Lateral))
	}

	if node.Subquery != nil {
		subCtx := FingerprintSubContext{}
		node.Subquery.Fingerprint(&subCtx, node, "Subquery")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("subquery")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
