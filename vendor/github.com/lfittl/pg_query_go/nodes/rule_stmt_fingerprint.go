// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node RuleStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("RuleStmt")

	if len(node.Actions.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Actions.Fingerprint(&subCtx, node, "Actions")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("actions")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Event) != 0 {
		ctx.WriteString("event")
		ctx.WriteString(strconv.Itoa(int(node.Event)))
	}

	if node.Instead {
		ctx.WriteString("instead")
		ctx.WriteString(strconv.FormatBool(node.Instead))
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

	if node.Replace {
		ctx.WriteString("replace")
		ctx.WriteString(strconv.FormatBool(node.Replace))
	}

	if node.Rulename != nil {
		ctx.WriteString("rulename")
		ctx.WriteString(*node.Rulename)
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
