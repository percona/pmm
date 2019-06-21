// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * SubPlan - executable expression node for a subplan (sub-SELECT)
 *
 * The planner replaces SubLink nodes in expression trees with SubPlan
 * nodes after it has finished planning the subquery.  SubPlan references
 * a sub-plantree stored in the subplans list of the toplevel PlannedStmt.
 * (We avoid a direct link to make it easier to copy expression trees
 * without causing multiple processing of the subplan.)
 *
 * In an ordinary subplan, testexpr points to an executable expression
 * (OpExpr, an AND/OR tree of OpExprs, or RowCompareExpr) for the combining
 * operator(s); the left-hand arguments are the original lefthand expressions,
 * and the right-hand arguments are PARAM_EXEC Param nodes representing the
 * outputs of the sub-select.  (NOTE: runtime coercion functions may be
 * inserted as well.)  This is just the same expression tree as testexpr in
 * the original SubLink node, but the PARAM_SUBLINK nodes are replaced by
 * suitably numbered PARAM_EXEC nodes.
 *
 * If the sub-select becomes an initplan rather than a subplan, the executable
 * expression is part of the outer plan's expression tree (and the SubPlan
 * node itself is not, but rather is found in the outer plan's initPlan
 * list).  In this case testexpr is NULL to avoid duplication.
 *
 * The planner also derives lists of the values that need to be passed into
 * and out of the subplan.  Input values are represented as a list "args" of
 * expressions to be evaluated in the outer-query context (currently these
 * args are always just Vars, but in principle they could be any expression).
 * The values are assigned to the global PARAM_EXEC params indexed by parParam
 * (the parParam and args lists must have the same ordering).  setParam is a
 * list of the PARAM_EXEC params that are computed by the sub-select, if it
 * is an initplan; they are listed in order by sub-select output column
 * position.  (parParam and setParam are integer Lists, not Bitmapsets,
 * because their ordering is significant.)
 *
 * Also, the planner computes startup and per-call costs for use of the
 * SubPlan.  Note that these include the cost of the subquery proper,
 * evaluation of the testexpr if any, and any hashtable management overhead.
 */
type SubPlan struct {
	Xpr Node `json:"xpr"`

	/* Fields copied from original SubLink: */
	SubLinkType SubLinkType `json:"subLinkType"` /* see above */

	/* The combining operators, transformed to an executable expression: */
	Testexpr Node `json:"testexpr"` /* OpExpr or RowCompareExpr expression tree */
	ParamIds List `json:"paramIds"` /* IDs of Params embedded in the above */

	/* Identification of the Plan tree to use: */
	PlanId int `json:"plan_id"` /* Index (from 1) in PlannedStmt.subplans */

	/* Identification of the SubPlan for EXPLAIN and debugging purposes: */
	PlanName *string `json:"plan_name"` /* A name assigned during planning */

	/* Extra data useful for determining subplan's output type: */
	FirstColType      Oid   `json:"firstColType"`      /* Type of first column of subplan result */
	FirstColTypmod    int32 `json:"firstColTypmod"`    /* Typmod of first column of subplan result */
	FirstColCollation Oid   `json:"firstColCollation"` /* Collation of first column of subplan
	 * result */

	/* Information about execution strategy: */
	UseHashTable bool `json:"useHashTable"` /* TRUE to store subselect output in a hash
	 * table (implies we are doing "IN") */

	UnknownEqFalse bool `json:"unknownEqFalse"` /* TRUE if it's okay to return FALSE when the
	 * spec result is UNKNOWN; this allows much
	 * simpler handling of null values */

	ParallelSafe bool `json:"parallel_safe"` /* is the subplan parallel-safe? */

	/* Note: parallel_safe does not consider contents of testexpr or args */

	/* Information for passing params into and out of the subselect: */

	/* setParam and parParam are lists of integers (param IDs) */
	SetParam List `json:"setParam"` /* initplan subqueries have to set these
	 * Params for parent plan */

	ParParam List `json:"parParam"` /* indices of input Params from parent plan */
	Args     List `json:"args"`     /* exprs to pass as parParam values */

	/* Estimated execution costs: */
	StartupCost Cost `json:"startup_cost"`  /* one-time setup cost */
	PerCallCost Cost `json:"per_call_cost"` /* cost for each subplan evaluation */
}

func (node SubPlan) MarshalJSON() ([]byte, error) {
	type SubPlanMarshalAlias SubPlan
	return json.Marshal(map[string]interface{}{
		"SubPlan": (*SubPlanMarshalAlias)(&node),
	})
}

func (node *SubPlan) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["xpr"] != nil {
		node.Xpr, err = UnmarshalNodeJSON(fields["xpr"])
		if err != nil {
			return
		}
	}

	if fields["subLinkType"] != nil {
		err = json.Unmarshal(fields["subLinkType"], &node.SubLinkType)
		if err != nil {
			return
		}
	}

	if fields["testexpr"] != nil {
		node.Testexpr, err = UnmarshalNodeJSON(fields["testexpr"])
		if err != nil {
			return
		}
	}

	if fields["paramIds"] != nil {
		node.ParamIds.Items, err = UnmarshalNodeArrayJSON(fields["paramIds"])
		if err != nil {
			return
		}
	}

	if fields["plan_id"] != nil {
		err = json.Unmarshal(fields["plan_id"], &node.PlanId)
		if err != nil {
			return
		}
	}

	if fields["plan_name"] != nil {
		err = json.Unmarshal(fields["plan_name"], &node.PlanName)
		if err != nil {
			return
		}
	}

	if fields["firstColType"] != nil {
		err = json.Unmarshal(fields["firstColType"], &node.FirstColType)
		if err != nil {
			return
		}
	}

	if fields["firstColTypmod"] != nil {
		err = json.Unmarshal(fields["firstColTypmod"], &node.FirstColTypmod)
		if err != nil {
			return
		}
	}

	if fields["firstColCollation"] != nil {
		err = json.Unmarshal(fields["firstColCollation"], &node.FirstColCollation)
		if err != nil {
			return
		}
	}

	if fields["useHashTable"] != nil {
		err = json.Unmarshal(fields["useHashTable"], &node.UseHashTable)
		if err != nil {
			return
		}
	}

	if fields["unknownEqFalse"] != nil {
		err = json.Unmarshal(fields["unknownEqFalse"], &node.UnknownEqFalse)
		if err != nil {
			return
		}
	}

	if fields["parallel_safe"] != nil {
		err = json.Unmarshal(fields["parallel_safe"], &node.ParallelSafe)
		if err != nil {
			return
		}
	}

	if fields["setParam"] != nil {
		node.SetParam.Items, err = UnmarshalNodeArrayJSON(fields["setParam"])
		if err != nil {
			return
		}
	}

	if fields["parParam"] != nil {
		node.ParParam.Items, err = UnmarshalNodeArrayJSON(fields["parParam"])
		if err != nil {
			return
		}
	}

	if fields["args"] != nil {
		node.Args.Items, err = UnmarshalNodeArrayJSON(fields["args"])
		if err != nil {
			return
		}
	}

	if fields["startup_cost"] != nil {
		err = json.Unmarshal(fields["startup_cost"], &node.StartupCost)
		if err != nil {
			return
		}
	}

	if fields["per_call_cost"] != nil {
		err = json.Unmarshal(fields["per_call_cost"], &node.PerCallCost)
		if err != nil {
			return
		}
	}

	return
}
