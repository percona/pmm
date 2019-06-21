// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * Query -
 *	  Parse analysis turns all statements into a Query tree
 *	  for further processing by the rewriter and planner.
 *
 *	  Utility statements (i.e. non-optimizable statements) have the
 *	  utilityStmt field set, and the rest of the Query is mostly dummy.
 *
 *	  Planning converts a Query tree into a Plan tree headed by a PlannedStmt
 *	  node --- the Query structure is not used by the executor.
 */
type Query struct {
	CommandType CmdType `json:"commandType"` /* select|insert|update|delete|utility */

	QuerySource QuerySource `json:"querySource"` /* where did I come from? */

	QueryId uint32 `json:"queryId"` /* query identifier (can be set by plugins) */

	CanSetTag bool `json:"canSetTag"` /* do I set the command result tag? */

	UtilityStmt Node `json:"utilityStmt"` /* non-null if commandType == CMD_UTILITY */

	ResultRelation int `json:"resultRelation"` /* rtable index of target relation for
	 * INSERT/UPDATE/DELETE; 0 for SELECT */

	HasAggs         bool `json:"hasAggs"`         /* has aggregates in tlist or havingQual */
	HasWindowFuncs  bool `json:"hasWindowFuncs"`  /* has window functions in tlist */
	HasTargetSrfs   bool `json:"hasTargetSRFs"`   /* has set-returning functions in tlist */
	HasSubLinks     bool `json:"hasSubLinks"`     /* has subquery SubLink */
	HasDistinctOn   bool `json:"hasDistinctOn"`   /* distinctClause is from DISTINCT ON */
	HasRecursive    bool `json:"hasRecursive"`    /* WITH RECURSIVE was specified */
	HasModifyingCte bool `json:"hasModifyingCTE"` /* has INSERT/UPDATE/DELETE in WITH */
	HasForUpdate    bool `json:"hasForUpdate"`    /* FOR [KEY] UPDATE/SHARE was specified */
	HasRowSecurity  bool `json:"hasRowSecurity"`  /* rewriter has applied some RLS policy */

	CteList List `json:"cteList"` /* WITH list (of CommonTableExpr's) */

	Rtable   List      `json:"rtable"`   /* list of range table entries */
	Jointree *FromExpr `json:"jointree"` /* table join tree (FROM and WHERE clauses) */

	TargetList List `json:"targetList"` /* target list (of TargetEntry) */

	Override OverridingKind `json:"override"` /* OVERRIDING clause */

	OnConflict *OnConflictExpr `json:"onConflict"` /* ON CONFLICT DO [NOTHING | UPDATE] */

	ReturningList List `json:"returningList"` /* return-values list (of TargetEntry) */

	GroupClause List `json:"groupClause"` /* a list of SortGroupClause's */

	GroupingSets List `json:"groupingSets"` /* a list of GroupingSet's if present */

	HavingQual Node `json:"havingQual"` /* qualifications applied to groups */

	WindowClause List `json:"windowClause"` /* a list of WindowClause's */

	DistinctClause List `json:"distinctClause"` /* a list of SortGroupClause's */

	SortClause List `json:"sortClause"` /* a list of SortGroupClause's */

	LimitOffset Node `json:"limitOffset"` /* # of result tuples to skip (int8 expr) */
	LimitCount  Node `json:"limitCount"`  /* # of result tuples to return (int8 expr) */

	RowMarks List `json:"rowMarks"` /* a list of RowMarkClause's */

	SetOperations Node `json:"setOperations"` /* set-operation tree if this is top level of
	 * a UNION/INTERSECT/EXCEPT query */

	ConstraintDeps List `json:"constraintDeps"` /* a list of pg_constraint OIDs that the query
	 * depends on to be semantically valid */

	WithCheckOptions List `json:"withCheckOptions"` /* a list of WithCheckOption's, which are
	 * only added during rewrite and therefore
	 * are not written out as part of Query. */

	/*
	 * The following two fields identify the portion of the source text string
	 * containing this query.  They are typically only populated in top-level
	 * Queries, not in sub-queries.  When not set, they might both be zero, or
	 * both be -1 meaning "unknown".
	 */
	StmtLocation int `json:"stmt_location"` /* start location, or -1 if unknown */
	StmtLen      int `json:"stmt_len"`      /* length in bytes; 0 means "rest of string" */
}

func (node Query) MarshalJSON() ([]byte, error) {
	type QueryMarshalAlias Query
	return json.Marshal(map[string]interface{}{
		"Query": (*QueryMarshalAlias)(&node),
	})
}

func (node *Query) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["commandType"] != nil {
		err = json.Unmarshal(fields["commandType"], &node.CommandType)
		if err != nil {
			return
		}
	}

	if fields["querySource"] != nil {
		err = json.Unmarshal(fields["querySource"], &node.QuerySource)
		if err != nil {
			return
		}
	}

	if fields["queryId"] != nil {
		err = json.Unmarshal(fields["queryId"], &node.QueryId)
		if err != nil {
			return
		}
	}

	if fields["canSetTag"] != nil {
		err = json.Unmarshal(fields["canSetTag"], &node.CanSetTag)
		if err != nil {
			return
		}
	}

	if fields["utilityStmt"] != nil {
		node.UtilityStmt, err = UnmarshalNodeJSON(fields["utilityStmt"])
		if err != nil {
			return
		}
	}

	if fields["resultRelation"] != nil {
		err = json.Unmarshal(fields["resultRelation"], &node.ResultRelation)
		if err != nil {
			return
		}
	}

	if fields["hasAggs"] != nil {
		err = json.Unmarshal(fields["hasAggs"], &node.HasAggs)
		if err != nil {
			return
		}
	}

	if fields["hasWindowFuncs"] != nil {
		err = json.Unmarshal(fields["hasWindowFuncs"], &node.HasWindowFuncs)
		if err != nil {
			return
		}
	}

	if fields["hasTargetSRFs"] != nil {
		err = json.Unmarshal(fields["hasTargetSRFs"], &node.HasTargetSrfs)
		if err != nil {
			return
		}
	}

	if fields["hasSubLinks"] != nil {
		err = json.Unmarshal(fields["hasSubLinks"], &node.HasSubLinks)
		if err != nil {
			return
		}
	}

	if fields["hasDistinctOn"] != nil {
		err = json.Unmarshal(fields["hasDistinctOn"], &node.HasDistinctOn)
		if err != nil {
			return
		}
	}

	if fields["hasRecursive"] != nil {
		err = json.Unmarshal(fields["hasRecursive"], &node.HasRecursive)
		if err != nil {
			return
		}
	}

	if fields["hasModifyingCTE"] != nil {
		err = json.Unmarshal(fields["hasModifyingCTE"], &node.HasModifyingCte)
		if err != nil {
			return
		}
	}

	if fields["hasForUpdate"] != nil {
		err = json.Unmarshal(fields["hasForUpdate"], &node.HasForUpdate)
		if err != nil {
			return
		}
	}

	if fields["hasRowSecurity"] != nil {
		err = json.Unmarshal(fields["hasRowSecurity"], &node.HasRowSecurity)
		if err != nil {
			return
		}
	}

	if fields["cteList"] != nil {
		node.CteList.Items, err = UnmarshalNodeArrayJSON(fields["cteList"])
		if err != nil {
			return
		}
	}

	if fields["rtable"] != nil {
		node.Rtable.Items, err = UnmarshalNodeArrayJSON(fields["rtable"])
		if err != nil {
			return
		}
	}

	if fields["jointree"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["jointree"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(FromExpr)
			node.Jointree = &val
		}
	}

	if fields["targetList"] != nil {
		node.TargetList.Items, err = UnmarshalNodeArrayJSON(fields["targetList"])
		if err != nil {
			return
		}
	}

	if fields["override"] != nil {
		err = json.Unmarshal(fields["override"], &node.Override)
		if err != nil {
			return
		}
	}

	if fields["onConflict"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["onConflict"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(OnConflictExpr)
			node.OnConflict = &val
		}
	}

	if fields["returningList"] != nil {
		node.ReturningList.Items, err = UnmarshalNodeArrayJSON(fields["returningList"])
		if err != nil {
			return
		}
	}

	if fields["groupClause"] != nil {
		node.GroupClause.Items, err = UnmarshalNodeArrayJSON(fields["groupClause"])
		if err != nil {
			return
		}
	}

	if fields["groupingSets"] != nil {
		node.GroupingSets.Items, err = UnmarshalNodeArrayJSON(fields["groupingSets"])
		if err != nil {
			return
		}
	}

	if fields["havingQual"] != nil {
		node.HavingQual, err = UnmarshalNodeJSON(fields["havingQual"])
		if err != nil {
			return
		}
	}

	if fields["windowClause"] != nil {
		node.WindowClause.Items, err = UnmarshalNodeArrayJSON(fields["windowClause"])
		if err != nil {
			return
		}
	}

	if fields["distinctClause"] != nil {
		node.DistinctClause.Items, err = UnmarshalNodeArrayJSON(fields["distinctClause"])
		if err != nil {
			return
		}
	}

	if fields["sortClause"] != nil {
		node.SortClause.Items, err = UnmarshalNodeArrayJSON(fields["sortClause"])
		if err != nil {
			return
		}
	}

	if fields["limitOffset"] != nil {
		node.LimitOffset, err = UnmarshalNodeJSON(fields["limitOffset"])
		if err != nil {
			return
		}
	}

	if fields["limitCount"] != nil {
		node.LimitCount, err = UnmarshalNodeJSON(fields["limitCount"])
		if err != nil {
			return
		}
	}

	if fields["rowMarks"] != nil {
		node.RowMarks.Items, err = UnmarshalNodeArrayJSON(fields["rowMarks"])
		if err != nil {
			return
		}
	}

	if fields["setOperations"] != nil {
		node.SetOperations, err = UnmarshalNodeJSON(fields["setOperations"])
		if err != nil {
			return
		}
	}

	if fields["constraintDeps"] != nil {
		node.ConstraintDeps.Items, err = UnmarshalNodeArrayJSON(fields["constraintDeps"])
		if err != nil {
			return
		}
	}

	if fields["withCheckOptions"] != nil {
		node.WithCheckOptions.Items, err = UnmarshalNodeArrayJSON(fields["withCheckOptions"])
		if err != nil {
			return
		}
	}

	if fields["stmt_location"] != nil {
		err = json.Unmarshal(fields["stmt_location"], &node.StmtLocation)
		if err != nil {
			return
		}
	}

	if fields["stmt_len"] != nil {
		err = json.Unmarshal(fields["stmt_len"], &node.StmtLen)
		if err != nil {
			return
		}
	}

	return
}
