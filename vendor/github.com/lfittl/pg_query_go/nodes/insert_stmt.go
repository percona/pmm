// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Insert Statement
 *
 * The source expression is represented by SelectStmt for both the
 * SELECT and VALUES cases.  If selectStmt is NULL, then the query
 * is INSERT ... DEFAULT VALUES.
 * ----------------------
 */
type InsertStmt struct {
	Relation         *RangeVar         `json:"relation"`         /* relation to insert into */
	Cols             List              `json:"cols"`             /* optional: names of the target columns */
	SelectStmt       Node              `json:"selectStmt"`       /* the source SELECT/VALUES, or NULL */
	OnConflictClause *OnConflictClause `json:"onConflictClause"` /* ON CONFLICT clause */
	ReturningList    List              `json:"returningList"`    /* list of expressions to return */
	WithClause       *WithClause       `json:"withClause"`       /* WITH clause */
	Override         OverridingKind    `json:"override"`         /* OVERRIDING clause */
}

func (node InsertStmt) MarshalJSON() ([]byte, error) {
	type InsertStmtMarshalAlias InsertStmt
	return json.Marshal(map[string]interface{}{
		"InsertStmt": (*InsertStmtMarshalAlias)(&node),
	})
}

func (node *InsertStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["relation"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["relation"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RangeVar)
			node.Relation = &val
		}
	}

	if fields["cols"] != nil {
		node.Cols.Items, err = UnmarshalNodeArrayJSON(fields["cols"])
		if err != nil {
			return
		}
	}

	if fields["selectStmt"] != nil {
		node.SelectStmt, err = UnmarshalNodeJSON(fields["selectStmt"])
		if err != nil {
			return
		}
	}

	if fields["onConflictClause"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["onConflictClause"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(OnConflictClause)
			node.OnConflictClause = &val
		}
	}

	if fields["returningList"] != nil {
		node.ReturningList.Items, err = UnmarshalNodeArrayJSON(fields["returningList"])
		if err != nil {
			return
		}
	}

	if fields["withClause"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["withClause"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(WithClause)
			node.WithClause = &val
		}
	}

	if fields["override"] != nil {
		err = json.Unmarshal(fields["override"], &node.Override)
		if err != nil {
			return
		}
	}

	return
}
