// Auto-generated - DO NOT EDIT

package pg_query

import "strconv"

func (node Query) Fingerprint(ctx FingerprintContext, parentNode Node, parentFieldName string) {
	ctx.WriteString("Query")

	if node.CanSetTag {
		ctx.WriteString("canSetTag")
		ctx.WriteString(strconv.FormatBool(node.CanSetTag))
	}

	if int(node.CommandType) != 0 {
		ctx.WriteString("commandType")
		ctx.WriteString(strconv.Itoa(int(node.CommandType)))
	}

	if len(node.ConstraintDeps.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.ConstraintDeps.Fingerprint(&subCtx, node, "ConstraintDeps")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("constraintDeps")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.CteList.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.CteList.Fingerprint(&subCtx, node, "CteList")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("cteList")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.DistinctClause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.DistinctClause.Fingerprint(&subCtx, node, "DistinctClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("distinctClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.GroupClause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.GroupClause.Fingerprint(&subCtx, node, "GroupClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("groupClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.GroupingSets.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.GroupingSets.Fingerprint(&subCtx, node, "GroupingSets")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("groupingSets")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.HasAggs {
		ctx.WriteString("hasAggs")
		ctx.WriteString(strconv.FormatBool(node.HasAggs))
	}

	if node.HasDistinctOn {
		ctx.WriteString("hasDistinctOn")
		ctx.WriteString(strconv.FormatBool(node.HasDistinctOn))
	}

	if node.HasForUpdate {
		ctx.WriteString("hasForUpdate")
		ctx.WriteString(strconv.FormatBool(node.HasForUpdate))
	}

	if node.HasModifyingCte {
		ctx.WriteString("hasModifyingCTE")
		ctx.WriteString(strconv.FormatBool(node.HasModifyingCte))
	}

	if node.HasRecursive {
		ctx.WriteString("hasRecursive")
		ctx.WriteString(strconv.FormatBool(node.HasRecursive))
	}

	if node.HasRowSecurity {
		ctx.WriteString("hasRowSecurity")
		ctx.WriteString(strconv.FormatBool(node.HasRowSecurity))
	}

	if node.HasSubLinks {
		ctx.WriteString("hasSubLinks")
		ctx.WriteString(strconv.FormatBool(node.HasSubLinks))
	}

	if node.HasTargetSrfs {
		ctx.WriteString("hasTargetSRFs")
		ctx.WriteString(strconv.FormatBool(node.HasTargetSrfs))
	}

	if node.HasWindowFuncs {
		ctx.WriteString("hasWindowFuncs")
		ctx.WriteString(strconv.FormatBool(node.HasWindowFuncs))
	}

	if node.HavingQual != nil {
		subCtx := FingerprintSubContext{}
		node.HavingQual.Fingerprint(&subCtx, node, "HavingQual")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("havingQual")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.Jointree != nil {
		subCtx := FingerprintSubContext{}
		node.Jointree.Fingerprint(&subCtx, node, "Jointree")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("jointree")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.LimitCount != nil {
		subCtx := FingerprintSubContext{}
		node.LimitCount.Fingerprint(&subCtx, node, "LimitCount")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("limitCount")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.LimitOffset != nil {
		subCtx := FingerprintSubContext{}
		node.LimitOffset.Fingerprint(&subCtx, node, "LimitOffset")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("limitOffset")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.OnConflict != nil {
		subCtx := FingerprintSubContext{}
		node.OnConflict.Fingerprint(&subCtx, node, "OnConflict")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("onConflict")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if int(node.Override) != 0 {
		ctx.WriteString("override")
		ctx.WriteString(strconv.Itoa(int(node.Override)))
	}

	if node.QueryId != 0 {
		ctx.WriteString("queryId")
		ctx.WriteString(strconv.Itoa(int(node.QueryId)))
	}

	if int(node.QuerySource) != 0 {
		ctx.WriteString("querySource")
		ctx.WriteString(strconv.Itoa(int(node.QuerySource)))
	}

	if node.ResultRelation != 0 {
		ctx.WriteString("resultRelation")
		ctx.WriteString(strconv.Itoa(int(node.ResultRelation)))
	}

	if len(node.ReturningList.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.ReturningList.Fingerprint(&subCtx, node, "ReturningList")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("returningList")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.RowMarks.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.RowMarks.Fingerprint(&subCtx, node, "RowMarks")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("rowMarks")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.Rtable.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.Rtable.Fingerprint(&subCtx, node, "Rtable")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("rtable")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.SetOperations != nil {
		subCtx := FingerprintSubContext{}
		node.SetOperations.Fingerprint(&subCtx, node, "SetOperations")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("setOperations")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.SortClause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.SortClause.Fingerprint(&subCtx, node, "SortClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("sortClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.StmtLen != 0 {
		ctx.WriteString("stmt_len")
		ctx.WriteString(strconv.Itoa(int(node.StmtLen)))
	}

	if node.StmtLocation != 0 {
		ctx.WriteString("stmt_location")
		ctx.WriteString(strconv.Itoa(int(node.StmtLocation)))
	}

	if len(node.TargetList.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.TargetList.Fingerprint(&subCtx, node, "TargetList")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("targetList")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if node.UtilityStmt != nil {
		subCtx := FingerprintSubContext{}
		node.UtilityStmt.Fingerprint(&subCtx, node, "UtilityStmt")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("utilityStmt")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.WindowClause.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.WindowClause.Fingerprint(&subCtx, node, "WindowClause")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("windowClause")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}

	if len(node.WithCheckOptions.Items) > 0 {
		subCtx := FingerprintSubContext{}
		node.WithCheckOptions.Fingerprint(&subCtx, node, "WithCheckOptions")

		if len(subCtx.parts) > 0 {
			ctx.WriteString("withCheckOptions")
			for _, part := range subCtx.parts {
				ctx.WriteString(part)
			}
		}
	}
}
