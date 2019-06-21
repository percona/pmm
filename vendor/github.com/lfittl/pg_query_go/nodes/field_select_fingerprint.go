// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node FieldSelect) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("FieldSelect")

	if node.Arg != nil {
		subCtx := FingerprintSubContext{}
		node.Arg.Fingerprint(&subCtx, node, "Arg")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("arg")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Fieldnum != 0 {
		ctx.WriteString("fieldnum")
		ctx.WriteString(strconv.Itoa(int(node.Fieldnum)))
	}

	if node.Resultcollid != 0 {
		ctx.WriteString("resultcollid")
		ctx.WriteString(strconv.Itoa(int(node.Resultcollid)))
	}

	if node.Resulttype != 0 {
		ctx.WriteString("resulttype")
		ctx.WriteString(strconv.Itoa(int(node.Resulttype)))
	}

	if node.Resulttypmod != 0 {
		ctx.WriteString("resulttypmod")
		ctx.WriteString(strconv.Itoa(int(node.Resulttypmod)))
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
