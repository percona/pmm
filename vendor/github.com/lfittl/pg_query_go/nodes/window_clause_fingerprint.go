// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node WindowClause) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("WindowClause")

	if node.CopiedOrder {
		ctx.WriteString("copiedOrder")
		ctx.WriteString(strconv.FormatBool(node.CopiedOrder))
	}

	if node.EndOffset != nil {
		subCtx := FingerprintSubContext{}
		node.EndOffset.Fingerprint(&subCtx, node, "EndOffset")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("endOffset")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.FrameOptions != 0 {
		ctx.WriteString("frameOptions")
		ctx.WriteString(strconv.Itoa(int(node.FrameOptions)))
	}

	if node.Name != nil {
		ctx.WriteString("name")
		ctx.WriteString(*node.Name)
	}

	if len(node.OrderClause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.OrderClause.Fingerprint(&subCtx, node, "OrderClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("orderClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.PartitionClause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.PartitionClause.Fingerprint(&subCtx, node, "PartitionClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("partitionClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Refname != nil {
		ctx.WriteString("refname")
		ctx.WriteString(*node.Refname)
	}

	if node.StartOffset != nil {
		subCtx := FingerprintSubContext{}
		node.StartOffset.Fingerprint(&subCtx, node, "StartOffset")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("startOffset")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Winref != 0 {
		ctx.WriteString("winref")
		ctx.WriteString(strconv.Itoa(int(node.Winref)))
	}
}
