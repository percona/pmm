// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Delete Statement
 * ----------------------
 */
type DeleteStmt struct {
	Relation      *RangeVar   `json:"relation"`      /* relation to delete from */
	UsingClause   List        `json:"usingClause"`   /* optional using clause for more tables */
	WhereClause   Node        `json:"whereClause"`   /* qualifications */
	ReturningList List        `json:"returningList"` /* list of expressions to return */
	WithClause    *WithClause `json:"withClause"`    /* WITH clause */
}

func (node DeleteStmt) MarshalJSON() ([]byte, error) {
	type DeleteStmtMarshalAlias DeleteStmt
	return json.Marshal(map[string]interface{}{
		"DeleteStmt": (*DeleteStmtMarshalAlias)(&node),
	})
}

func (node *DeleteStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["usingClause"] != nil {
		node.UsingClause.Items, err = UnmarshalNodeArrayJSON(fields["usingClause"])
		if err != nil {
			return
		}
	}

	if fields["whereClause"] != nil {
		node.WhereClause, err = UnmarshalNodeJSON(fields["whereClause"])
		if err != nil {
			return
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

	return
}
