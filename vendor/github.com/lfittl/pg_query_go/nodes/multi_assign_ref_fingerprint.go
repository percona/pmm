// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node MultiAssignRef) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("MultiAssignRef")

	if node.Colno != 0 {
		ctx.WriteString("colno")
		ctx.WriteString(strconv.Itoa(int(node.Colno)))
	}

	if node.Ncolumns != 0 {
		ctx.WriteString("ncolumns")
		ctx.WriteString(strconv.Itoa(int(node.Ncolumns)))
	}

	if node.Source != nil {
		subCtx := FingerprintSubContext{}
		node.Source.Fingerprint(&subCtx, node, "Source")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("source")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
