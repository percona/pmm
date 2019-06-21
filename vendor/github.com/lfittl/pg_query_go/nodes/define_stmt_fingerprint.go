// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node DefineStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("DefineStmt")

	if len(node.Args.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Args.Fingerprint(&subCtx, node, "Args")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("args")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Definition.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Definition.Fingerprint(&subCtx, node, "Definition")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("definition")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

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

	if node.IfNotExists {
		ctx.WriteString("if_not_exists")
		ctx.WriteString(strconv.FormatBool(node.IfNotExists))
	}

	if int(node.Kind) != 0 {
		ctx.WriteString("kind")
		ctx.WriteString(strconv.Itoa(int(node.Kind)))
	}

	if node.Oldstyle {
		ctx.WriteString("oldstyle")
		ctx.WriteString(strconv.FormatBool(node.Oldstyle))
	}
}
