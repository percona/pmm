// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node AlterSubscriptionStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterSubscriptionStmt")

	if node.Conninfo != nil {
		ctx.WriteString("conninfo")
		ctx.WriteString(*node.Conninfo)
	}

	if int(node.Kind) != 0 {
		ctx.WriteString("kind")
		ctx.WriteString(strconv.Itoa(int(node.Kind)))
	}

	if len(node.Options.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Options.Fingerprint(&subCtx, node, "Options")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("options")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Publication.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Publication.Fingerprint(&subCtx, node, "Publication")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("publication")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Subname != nil {
		ctx.WriteString("subname")
		ctx.WriteString(*node.Subname)
	}
}
