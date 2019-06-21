// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CreateConversionStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateConversionStmt")

	if len(node.ConversionName.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.ConversionName.Fingerprint(&subCtx, node, "ConversionName")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("conversion_name")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Def {
		ctx.WriteString("def")
		ctx.WriteString(strconv.FormatBool(node.Def))
	}

	if node.ForEncodingName != nil {
		ctx.WriteString("for_encoding_name")
		ctx.WriteString(*node.ForEncodingName)
	}

	if len(node.FuncName.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.FuncName.Fingerprint(&subCtx, node, "FuncName")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("func_name")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.ToEncodingName != nil {
		ctx.WriteString("to_encoding_name")
		ctx.WriteString(*node.ToEncodingName)
	}
}
