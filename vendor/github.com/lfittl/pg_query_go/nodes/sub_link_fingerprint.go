// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node SubLink) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("SubLink")
	// Intentionally ignoring node.Location for fingerprinting

	if len(node.OperName.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.OperName.Fingerprint(&subCtx, node, "OperName")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("operName")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.SubLinkId != 0 {
		ctx.WriteString("subLinkId")
		ctx.WriteString(strconv.Itoa(int(node.SubLinkId)))
	}

	if int(node.SubLinkType) != 0 {
		ctx.WriteString("subLinkType")
		ctx.WriteString(strconv.Itoa(int(node.SubLinkType)))
	}

	if node.Subselect != nil {
		subCtx := FingerprintSubContext{}
		node.Subselect.Fingerprint(&subCtx, node, "Subselect")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("subselect")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Testexpr != nil {
		subCtx := FingerprintSubContext{}
		node.Testexpr.Fingerprint(&subCtx, node, "Testexpr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("testexpr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
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
