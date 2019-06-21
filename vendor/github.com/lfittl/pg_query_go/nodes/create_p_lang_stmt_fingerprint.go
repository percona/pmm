// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CreatePLangStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreatePLangStmt")

	if len(node.Plhandler.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Plhandler.Fingerprint(&subCtx, node, "Plhandler")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("plhandler")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Plinline.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Plinline.Fingerprint(&subCtx, node, "Plinline")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("plinline")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Plname != nil {
		ctx.WriteString("plname")
		ctx.WriteString(*node.Plname)
	}

	if node.Pltrusted {
		ctx.WriteString("pltrusted")
		ctx.WriteString(strconv.FormatBool(node.Pltrusted))
	}

	if len(node.Plvalidator.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Plvalidator.Fingerprint(&subCtx, node, "Plvalidator")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("plvalidator")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Replace {
		ctx.WriteString("replace")
		ctx.WriteString(strconv.FormatBool(node.Replace))
	}
}
