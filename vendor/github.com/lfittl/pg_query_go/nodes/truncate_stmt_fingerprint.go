// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node TruncateStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("TruncateStmt")

	if int(node.Behavior) != 0 {
		ctx.WriteString("behavior")
		ctx.WriteString(strconv.Itoa(int(node.Behavior)))
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

	if node.RestartSeqs {
		ctx.WriteString("restart_seqs")
		ctx.WriteString(strconv.FormatBool(node.RestartSeqs))
	}
}
