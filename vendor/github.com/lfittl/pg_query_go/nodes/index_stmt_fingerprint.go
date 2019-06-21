// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node IndexStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("IndexStmt")

	if node.AccessMethod != nil {
		ctx.WriteString("accessMethod")
		ctx.WriteString(*node.AccessMethod)
	}

	if node.Concurrent {
		ctx.WriteString("concurrent")
		ctx.WriteString(strconv.FormatBool(node.Concurrent))
	}

	if node.Deferrable {
		ctx.WriteString("deferrable")
		ctx.WriteString(strconv.FormatBool(node.Deferrable))
	}

	if len(node.ExcludeOpNames.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.ExcludeOpNames.Fingerprint(&subCtx, node, "ExcludeOpNames")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("excludeOpNames")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Idxcomment != nil {
		ctx.WriteString("idxcomment")
		ctx.WriteString(*node.Idxcomment)
	}

	if node.Idxname != nil {
		ctx.WriteString("idxname")
		ctx.WriteString(*node.Idxname)
	}

	if node.IfNotExists {
		ctx.WriteString("if_not_exists")
		ctx.WriteString(strconv.FormatBool(node.IfNotExists))
	}

	if node.IndexOid != 0 {
		ctx.WriteString("indexOid")
		ctx.WriteString(strconv.Itoa(int(node.IndexOid)))
	}

	if len(node.IndexParams.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.IndexParams.Fingerprint(&subCtx, node, "IndexParams")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("indexParams")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Initdeferred {
		ctx.WriteString("initdeferred")
		ctx.WriteString(strconv.FormatBool(node.Initdeferred))
	}

	if node.Isconstraint {
		ctx.WriteString("isconstraint")
		ctx.WriteString(strconv.FormatBool(node.Isconstraint))
	}

	if node.OldNode != 0 {
		ctx.WriteString("oldNode")
		ctx.WriteString(strconv.Itoa(int(node.OldNode)))
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

	if node.Primary {
		ctx.WriteString("primary")
		ctx.WriteString(strconv.FormatBool(node.Primary))
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

	if node.TableSpace != nil {
		ctx.WriteString("tableSpace")
		ctx.WriteString(*node.TableSpace)
	}

	if node.Transformed {
		ctx.WriteString("transformed")
		ctx.WriteString(strconv.FormatBool(node.Transformed))
	}

	if node.Unique {
		ctx.WriteString("unique")
		ctx.WriteString(strconv.FormatBool(node.Unique))
	}

	if node.WhereClause != nil {
		subCtx := FingerprintSubContext{}
		node.WhereClause.Fingerprint(&subCtx, node, "WhereClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("whereClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
