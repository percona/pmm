// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*----------
 * JoinExpr - for SQL JOIN expressions
 *
 * isNatural, usingClause, and quals are interdependent.  The user can write
 * only one of NATURAL, USING(), or ON() (this is enforced by the grammar).
 * If he writes NATURAL then parse analysis generates the equivalent USING()
 * list, and from that fills in "quals" with the right equality comparisons.
 * If he writes USING() then "quals" is filled with equality comparisons.
 * If he writes ON() then only "quals" is set.  Note that NATURAL/USING
 * are not equivalent to ON() since they also affect the output column list.
 *
 * alias is an Alias node representing the AS alias-clause attached to the
 * join expression, or NULL if no clause.  NB: presence or absence of the
 * alias has a critical impact on semantics, because a join with an alias
 * restricts visibility of the tables/columns inside it.
 *
 * During parse analysis, an RTE is created for the Join, and its index
 * is filled into rtindex.  This RTE is present mainly so that Vars can
 * be created that refer to the outputs of the join.  The planner sometimes
 * generates JoinExprs internally; these can have rtindex = 0 if there are
 * no join alias variables referencing such joins.
 *----------
 */
type JoinExpr struct {
	Jointype    JoinType `json:"jointype"`    /* type of join */
	IsNatural   bool     `json:"isNatural"`   /* Natural join? Will need to shape table */
	Larg        Node     `json:"larg"`        /* left subtree */
	Rarg        Node     `json:"rarg"`        /* right subtree */
	UsingClause List     `json:"usingClause"` /* USING clause, if any (list of String) */
	Quals       Node     `json:"quals"`       /* qualifiers on join, if any */
	Alias       *Alias   `json:"alias"`       /* user-written alias clause, if any */
	Rtindex     int      `json:"rtindex"`     /* RT index assigned for join, or 0 */
}

func (node JoinExpr) MarshalJSON() ([]byte, error) {
	type JoinExprMarshalAlias JoinExpr
	return json.Marshal(map[string]interface{}{
		"JoinExpr": (*JoinExprMarshalAlias)(&node),
	})
}

func (node *JoinExpr) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["jointype"] != nil {
		err = json.Unmarshal(fields["jointype"], &node.Jointype)
		if err != nil {
			return
		}
	}

	if fields["isNatural"] != nil {
		err = json.Unmarshal(fields["isNatural"], &node.IsNatural)
		if err != nil {
			return
		}
	}

	if fields["larg"] != nil {
		node.Larg, err = UnmarshalNodeJSON(fields["larg"])
		if err != nil {
			return
		}
	}

	if fields["rarg"] != nil {
		node.Rarg, err = UnmarshalNodeJSON(fields["rarg"])
		if err != nil {
			return
		}
	}

	if fields["usingClause"] != nil {
		node.UsingClause.Items, err = UnmarshalNodeArrayJSON(fields["usingClause"])
		if err != nil {
			return
		}
	}

	if fields["quals"] != nil {
		node.Quals, err = UnmarshalNodeJSON(fields["quals"])
		if err != nil {
			return
		}
	}

	if fields["alias"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["alias"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(Alias)
			node.Alias = &val
		}
	}

	if fields["rtindex"] != nil {
		err = json.Unmarshal(fields["rtindex"], &node.Rtindex)
		if err != nil {
			return
		}
	}

	return
}
