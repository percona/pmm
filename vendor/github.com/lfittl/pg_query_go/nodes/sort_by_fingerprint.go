// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node SortBy) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("SortBy")
	// Intentionally ignoring node.Location for fingerprinting

	if node.Node != nil {
		subCtx := FingerprintSubContext{}
		node.Node.Fingerprint(&subCtx, node, "Node")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("node")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.SortbyDir) != 0 {
		ctx.WriteString("sortby_dir")
		ctx.WriteString(strconv.Itoa(int(node.SortbyDir)))
	}

	if int(node.SortbyNulls) != 0 {
		ctx.WriteString("sortby_nulls")
		ctx.WriteString(strconv.Itoa(int(node.SortbyNulls)))
	}

	if len(node.UseOp.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.UseOp.Fingerprint(&subCtx, node, "UseOp")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("useOp")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
