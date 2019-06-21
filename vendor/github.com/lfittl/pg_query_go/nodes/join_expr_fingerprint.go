// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node JoinExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("JoinExpr")

	if node.Alias != nil {
		subCtx := FingerprintSubContext{}
		node.Alias.Fingerprint(&subCtx, node, "Alias")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("alias")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.IsNatural {
		ctx.WriteString("isNatural")
		ctx.WriteString(strconv.FormatBool(node.IsNatural))
	}

	if int(node.Jointype) != 0 {
		ctx.WriteString("jointype")
		ctx.WriteString(strconv.Itoa(int(node.Jointype)))
	}

	if node.Larg != nil {
		subCtx := FingerprintSubContext{}
		node.Larg.Fingerprint(&subCtx, node, "Larg")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("larg")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Quals != nil {
		subCtx := FingerprintSubContext{}
		node.Quals.Fingerprint(&subCtx, node, "Quals")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("quals")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Rarg != nil {
		subCtx := FingerprintSubContext{}
		node.Rarg.Fingerprint(&subCtx, node, "Rarg")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("rarg")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Rtindex != 0 {
		ctx.WriteString("rtindex")
		ctx.WriteString(strconv.Itoa(int(node.Rtindex)))
	}

	if len(node.UsingClause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.UsingClause.Fingerprint(&subCtx, node, "UsingClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("usingClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
