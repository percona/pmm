// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node LockStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("LockStmt")

	if node.Mode != 0 {
		ctx.WriteString("mode")
		ctx.WriteString(strconv.Itoa(int(node.Mode)))
	}

	if node.Nowait {
		ctx.WriteString("nowait")
		ctx.WriteString(strconv.FormatBool(node.Nowait))
	}

	if len(node.Relations.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Relations.Fingerprint(&subCtx, node, "Relations")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("relations")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
