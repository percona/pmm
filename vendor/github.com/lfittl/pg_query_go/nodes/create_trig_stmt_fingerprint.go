// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node CreateTrigStmt) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("CreateTrigStmt")

	if len(node.Args.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Args.Fingerprint(&subCtx, node, "Args")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("args")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Columns.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Columns.Fingerprint(&subCtx, node, "Columns")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("columns")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Constrrel != nil {
		subCtx := FingerprintSubContext{}
		node.Constrrel.Fingerprint(&subCtx, node, "Constrrel")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("constrrel")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Deferrable {
		ctx.WriteString("deferrable")
		ctx.WriteString(strconv.FormatBool(node.Deferrable))
	}

	if node.Events != 0 {
		ctx.WriteString("events")
		ctx.WriteString(strconv.Itoa(int(node.Events)))
	}

	if len(node.Funcname.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Funcname.Fingerprint(&subCtx, node, "Funcname")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("funcname")
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

	if node.Row {
		ctx.WriteString("row")
		ctx.WriteString(strconv.FormatBool(node.Row))
	}

	if node.Timing != 0 {
		ctx.WriteString("timing")
		ctx.WriteString(strconv.Itoa(int(node.Timing)))
	}

	if len(node.TransitionRels.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.TransitionRels.Fingerprint(&subCtx, node, "TransitionRels")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("transitionRels")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Trigname != nil {
		ctx.WriteString("trigname")
		ctx.WriteString(*node.Trigname)
	}

	if node.WhenClause != nil {
		subCtx := FingerprintSubContext{}
		node.WhenClause.Fingerprint(&subCtx, node, "WhenClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("whenClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
