// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node TableSampleClause) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("TableSampleClause")

	if len(node.Args.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Args.Fingerprint(&subCtx, node, "Args")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("args")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Repeatable != nil {
		subCtx := FingerprintSubContext{}
		node.Repeatable.Fingerprint(&subCtx, node, "Repeatable")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("repeatable")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Tsmhandler != 0 {
		ctx.WriteString("tsmhandler")
		ctx.WriteString(strconv.Itoa(int(node.Tsmhandler)))
	}
}
