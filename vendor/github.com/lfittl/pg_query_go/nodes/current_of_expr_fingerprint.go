// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CurrentOfExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CurrentOfExpr")

	if node.CursorName != nil {
		ctx.WriteString("cursor_name")
		ctx.WriteString(*node.CursorName)
	}

	if node.CursorParam != 0 {
		ctx.WriteString("cursor_param")
		ctx.WriteString(strconv.Itoa(int(node.CursorParam)))
	}

	if node.Cvarno != 0 {
		ctx.WriteString("cvarno")
		ctx.WriteString(strconv.Itoa(int(node.Cvarno)))
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
