// Auto-generated - DO NOT EDIT

package pg_query

func (node CreateEventTrigStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateEventTrigStmt")

	if node.Eventname != nil {
		ctx.WriteString("eventname")
		ctx.WriteString(*node.Eventname)
	}

	if len(node.Funcname.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Funcname.Fingerprint(&subCtx, node, "Funcname")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("funcname")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Trigname != nil {
		ctx.WriteString("trigname")
		ctx.WriteString(*node.Trigname)
	}

	if len(node.Whenclause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Whenclause.Fingerprint(&subCtx, node, "Whenclause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("whenclause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
