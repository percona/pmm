// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node GroupingSet) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("GroupingSet")

	if len(node.Content.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Content.Fingerprint(&subCtx, node, "Content")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("content")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Kind) != 0 {
		ctx.WriteString("kind")
		ctx.WriteString(strconv.Itoa(int(node.Kind)))
	}

	// Intentionally ignoring node.Location for fingerprinting
}
