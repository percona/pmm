// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node PartitionRangeDatum) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("PartitionRangeDatum")

	if int(node.Kind) != 0 {
		ctx.WriteString("kind")
		ctx.WriteString(strconv.Itoa(int(node.Kind)))
	}

	// Intentionally ignoring node.Location for fingerprinting

	if node.Value != nil {
		subCtx := FingerprintSubContext{}
		node.Value.Fingerprint(&subCtx, node, "Value")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("value")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
