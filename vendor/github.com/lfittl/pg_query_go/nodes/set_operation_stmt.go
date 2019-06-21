// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Set Operation node for post-analysis query trees
 *
 * After parse analysis, a SELECT with set operations is represented by a
 * top-level Query node containing the leaf SELECTs as subqueries in its
 * range table.  Its setOperations field shows the tree of set operations,
 * with leaf SelectStmt nodes replaced by RangeTblRef nodes, and internal
 * nodes replaced by SetOperationStmt nodes.  Information about the output
 * column types is added, too.  (Note that the child nodes do not necessarily
 * produce these types directly, but we've checked that their output types
 * can be coerced to the output column type.)  Also, if it's not UNION ALL,
 * information about the types' sort/group semantics is provided in the form
 * of a SortGroupClause list (same representation as, eg, DISTINCT).
 * The resolved common column collations are provided too; but note that if
 * it's not UNION ALL, it's okay for a column to not have a common collation,
 * so a member of the colCollations list could be InvalidOid even though the
 * column has a collatable type.
 * ----------------------
 */
type SetOperationStmt struct {
	Op   SetOperation `json:"op"`   /* type of set op */
	All  bool         `json:"all"`  /* ALL specified? */
	Larg Node         `json:"larg"` /* left child */
	Rarg Node         `json:"rarg"` /* right child */

	/* Eventually add fields for CORRESPONDING spec here */

	/* Fields derived during parse analysis: */
	ColTypes      List `json:"colTypes"`      /* OID list of output column type OIDs */
	ColTypmods    List `json:"colTypmods"`    /* integer list of output column typmods */
	ColCollations List `json:"colCollations"` /* OID list of output column collation OIDs */
	GroupClauses  List `json:"groupClauses"`  /* a list of SortGroupClause's */

	/* groupClauses is NIL if UNION ALL, but must be set otherwise */
}

func (node SetOperationStmt) MarshalJSON() ([]byte, error) {
	type SetOperationStmtMarshalAlias SetOperationStmt
	return json.Marshal(map[string]interface{}{
		"SetOperationStmt": (*SetOperationStmtMarshalAlias)(&node),
	})
}

func (node *SetOperationStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["op"] != nil {
		err = json.Unmarshal(fields["op"], &node.Op)
		if err != nil {
			return
		}
	}

	if fields["all"] != nil {
		err = json.Unmarshal(fields["all"], &node.All)
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

	if fields["colTypes"] != nil {
		node.ColTypes.Items, err = UnmarshalNodeArrayJSON(fields["colTypes"])
		if err != nil {
			return
		}
	}

	if fields["colTypmods"] != nil {
		node.ColTypmods.Items, err = UnmarshalNodeArrayJSON(fields["colTypmods"])
		if err != nil {
			return
		}
	}

	if fields["colCollations"] != nil {
		node.ColCollations.Items, err = UnmarshalNodeArrayJSON(fields["colCollations"])
		if err != nil {
			return
		}
	}

	if fields["groupClauses"] != nil {
		node.GroupClauses.Items, err = UnmarshalNodeArrayJSON(fields["groupClauses"])
		if err != nil {
			return
		}
	}

	return
}
