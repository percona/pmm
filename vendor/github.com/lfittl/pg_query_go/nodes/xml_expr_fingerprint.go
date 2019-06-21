// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node XmlExpr) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("XmlExpr")

	if len(node.ArgNames.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.ArgNames.Fingerprint(&subCtx, node, "ArgNames")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("arg_names")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

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
	// Intentionally ignoring node.Location for fingerprinting

	if node.Name != nil {
		ctx.WriteString("name")
		ctx.WriteString(*node.Name)
	}

	if len(node.NamedArgs.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.NamedArgs.Fingerprint(&subCtx, node, "NamedArgs")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("named_args")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Op) != 0 {
		ctx.WriteString("op")
		ctx.WriteString(strconv.Itoa(int(node.Op)))
	}

	if node.Type != 0 {
		ctx.WriteString("type")
		ctx.WriteString(strconv.Itoa(int(node.Type)))
	}

	if node.Typmod != 0 {
		ctx.WriteString("typmod")
		ctx.WriteString(strconv.Itoa(int(node.Typmod)))
	}

	if int(node.Xmloption) != 0 {
		ctx.WriteString("xmloption")
		ctx.WriteString(strconv.Itoa(int(node.Xmloption)))
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
