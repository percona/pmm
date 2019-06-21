// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CreateTransformStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateTransformStmt")

	if node.Fromsql != nil {
		subCtx := FingerprintSubContext{}
		node.Fromsql.Fingerprint(&subCtx, node, "Fromsql")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("fromsql")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Lang != nil {
		ctx.WriteString("lang")
		ctx.WriteString(*node.Lang)
	}

	if node.Replace {
		ctx.WriteString("replace")
		ctx.WriteString(strconv.FormatBool(node.Replace))
	}

	if node.Tosql != nil {
		subCtx := FingerprintSubContext{}
		node.Tosql.Fingerprint(&subCtx, node, "Tosql")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("tosql")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.TypeName != nil {
		subCtx := FingerprintSubContext{}
		node.TypeName.Fingerprint(&subCtx, node, "TypeName")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("type_name")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
