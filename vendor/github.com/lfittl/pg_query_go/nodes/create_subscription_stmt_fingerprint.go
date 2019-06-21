// Auto-generated - DO NOT EDIT

package pg_query

func (node CreateSubscriptionStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateSubscriptionStmt")

	if node.Conninfo != nil {
		ctx.WriteString("conninfo")
		ctx.WriteString(*node.Conninfo)
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
