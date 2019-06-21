// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node ViewStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("ViewStmt")

	if len(node.Aliases.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Aliases.Fingerprint(&subCtx, node, "Aliases")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("aliases")
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

	if node.Replace {
		ctx.WriteString("replace")
		ctx.WriteString(strconv.FormatBool(node.Replace))
	}

	if node.View != nil {
		subCtx := FingerprintSubContext{}
		node.View.Fingerprint(&subCtx, node, "View")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("view")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.WithCheckOption) != 0 {
		ctx.WriteString("withCheckOption")
		ctx.WriteString(strconv.Itoa(int(node.WithCheckOption)))
	}
}
