// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node WithCheckOption) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("WithCheckOption")

	if node.Cascaded {
		ctx.WriteString("cascaded")
		ctx.WriteString(strconv.FormatBool(node.Cascaded))
	}

	if int(node.Kind) != 0 {
		ctx.WriteString("kind")
		ctx.WriteString(strconv.Itoa(int(node.Kind)))
	}

	if node.Polname != nil {
		ctx.WriteString("polname")
		ctx.WriteString(*node.Polname)
	}

	if node.Qual != nil {
		subCtx := FingerprintSubContext{}
		node.Qual.Fingerprint(&subCtx, node, "Qual")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("qual")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Relname != nil {
		ctx.WriteString("relname")
		ctx.WriteString(*node.Relname)
	}
}
