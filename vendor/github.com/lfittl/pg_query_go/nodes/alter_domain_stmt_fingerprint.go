// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node AlterDomainStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("AlterDomainStmt")

	if int(node.Behavior) != 0 {
		ctx.WriteString("behavior")
		ctx.WriteString(strconv.Itoa(int(node.Behavior)))
	}

	if node.Def != nil {
		subCtx := FingerprintSubContext{}
		node.Def.Fingerprint(&subCtx, node, "Def")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("def")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.MissingOk {
		ctx.WriteString("missing_ok")
		ctx.WriteString(strconv.FormatBool(node.MissingOk))
	}

	if node.Name != nil {
		ctx.WriteString("name")
		ctx.WriteString(*node.Name)
	}

	if node.Subtype != 0 {
		ctx.WriteString("subtype")
		ctx.WriteString(string(node.Subtype))

	}

	if len(node.TypeName.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.TypeName.Fingerprint(&subCtx, node, "TypeName")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("typeName")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
