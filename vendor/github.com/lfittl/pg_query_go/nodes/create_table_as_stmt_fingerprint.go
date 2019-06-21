// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CreateTableAsStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateTableAsStmt")

	if node.IfNotExists {
		ctx.WriteString("if_not_exists")
		ctx.WriteString(strconv.FormatBool(node.IfNotExists))
	}

	if node.Into != nil {
		subCtx := FingerprintSubContext{}
		node.Into.Fingerprint(&subCtx, node, "Into")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("into")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.IsSelectInto {
		ctx.WriteString("is_select_into")
		ctx.WriteString(strconv.FormatBool(node.IsSelectInto))
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

	if int(node.Relkind) != 0 {
		ctx.WriteString("relkind")
		ctx.WriteString(strconv.Itoa(int(node.Relkind)))
	}
}
