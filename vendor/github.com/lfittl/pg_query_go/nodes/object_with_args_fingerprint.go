// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node ObjectWithArgs) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("ObjectWithArgs")

	if node.ArgsUnspecified {
		ctx.WriteString("args_unspecified")
		ctx.WriteString(strconv.FormatBool(node.ArgsUnspecified))
	}

	if len(node.Objargs.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Objargs.Fingerprint(&subCtx, node, "Objargs")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("objargs")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Objname.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Objname.Fingerprint(&subCtx, node, "Objname")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("objname")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
