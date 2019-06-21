// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CopyStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CopyStmt")

	if len(node.Attlist.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Attlist.Fingerprint(&subCtx, node, "Attlist")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("attlist")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Filename != nil {
		ctx.WriteString("filename")
		ctx.WriteString(*node.Filename)
	}

	if node.IsFrom {
		ctx.WriteString("is_from")
		ctx.WriteString(strconv.FormatBool(node.IsFrom))
	}

	if node.IsProgram {
		ctx.WriteString("is_program")
		ctx.WriteString(strconv.FormatBool(node.IsProgram))
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

	if node.Query != nil {
		subCtx := FingerprintSubContext{}
		node.Query.Fingerprint(&subCtx, node, "Query")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("query")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Relation != nil {
		subCtx := FingerprintSubContext{}
		node.Relation.Fingerprint(&subCtx, node, "Relation")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("relation")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
