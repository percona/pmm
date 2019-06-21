// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CreateFunctionStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateFunctionStmt")

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

	if len(node.Parameters.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Parameters.Fingerprint(&subCtx, node, "Parameters")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("parameters")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Replace {
		ctx.WriteString("replace")
		ctx.WriteString(strconv.FormatBool(node.Replace))
	}

	if node.ReturnType != nil {
		subCtx := FingerprintSubContext{}
		node.ReturnType.Fingerprint(&subCtx, node, "ReturnType")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("returnType")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.WithClause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.WithClause.Fingerprint(&subCtx, node, "WithClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("withClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
