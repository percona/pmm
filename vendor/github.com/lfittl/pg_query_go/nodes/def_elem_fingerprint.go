// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node DefElem) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("DefElem")

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

	if int(node.Defaction) != 0 {
		ctx.WriteString("defaction")
		ctx.WriteString(strconv.Itoa(int(node.Defaction)))
	}

	if node.Defname != nil {
		ctx.WriteString("defname")
		ctx.WriteString(*node.Defname)
	}

	if node.Defnamespace != nil {
		ctx.WriteString("defnamespace")
		ctx.WriteString(*node.Defnamespace)
	}

	// Intentionally ignoring node.Location for fingerprinting
}
