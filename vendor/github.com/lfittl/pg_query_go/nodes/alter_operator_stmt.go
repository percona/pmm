// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Alter Operator Set Restrict, Join
 * ----------------------
 */
type AlterOperatorStmt struct {
	Opername *ObjectWithArgs `json:"opername"` /* operator name and argument types */
	Options  List            `json:"options"`  /* List of DefElem nodes */
}

func (node AlterOperatorStmt) MarshalJSON() ([]byte, error) {
	type AlterOperatorStmtMarshalAlias AlterOperatorStmt
	return json.Marshal(map[string]interface{}{
		"AlterOperatorStmt": (*AlterOperatorStmtMarshalAlias)(&node),
	})
}

func (node *AlterOperatorStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["opername"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["opername"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(ObjectWithArgs)
			node.Opername = &val
		}
	}

	if fields["options"] != nil {
		node.Options.Items, err = UnmarshalNodeArrayJSON(fields["options"])
		if err != nil {
			return
		}
	}

	return
}
