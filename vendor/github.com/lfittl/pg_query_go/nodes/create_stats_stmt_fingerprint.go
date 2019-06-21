// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CreateStatsStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateStatsStmt")

	if len(node.Defnames.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Defnames.Fingerprint(&subCtx, node, "Defnames")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("defnames")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Exprs.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Exprs.Fingerprint(&subCtx, node, "Exprs")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("exprs")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.IfNotExists {
		ctx.WriteString("if_not_exists")
		ctx.WriteString(strconv.FormatBool(node.IfNotExists))
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

	if len(node.StatTypes.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.StatTypes.Fingerprint(&subCtx, node, "StatTypes")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("stat_types")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
