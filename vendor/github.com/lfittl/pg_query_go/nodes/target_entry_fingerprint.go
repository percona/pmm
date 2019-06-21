// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node TargetEntry) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("TargetEntry")

	if node.Expr != nil {
		subCtx := FingerprintSubContext{}
		node.Expr.Fingerprint(&subCtx, node, "Expr")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("expr")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Resjunk {
		ctx.WriteString("resjunk")
		ctx.WriteString(strconv.FormatBool(node.Resjunk))
	}

	if node.Resname != nil {
		ctx.WriteString("resname")
		ctx.WriteString(*node.Resname)
	}

	if node.Resno != 0 {
		ctx.WriteString("resno")
		ctx.WriteString(strconv.Itoa(int(node.Resno)))
	}

	if node.Resorigcol != 0 {
		ctx.WriteString("resorigcol")
		ctx.WriteString(strconv.Itoa(int(node.Resorigcol)))
	}

	if node.Resorigtbl != 0 {
		ctx.WriteString("resorigtbl")
		ctx.WriteString(strconv.Itoa(int(node.Resorigtbl)))
	}

	if node.Ressortgroupref != 0 {
		ctx.WriteString("ressortgroupref")
		ctx.WriteString(strconv.Itoa(int(node.Ressortgroupref)))
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
