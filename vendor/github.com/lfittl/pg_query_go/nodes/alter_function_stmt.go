// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create Function Statement
 * ----------------------
 */
type AlterFunctionStmt struct {
	Func    *ObjectWithArgs `json:"func"`    /* name and args of function */
	Actions List            `json:"actions"` /* list of DefElem */
}

func (node AlterFunctionStmt) MarshalJSON() ([]byte, error) {
	type AlterFunctionStmtMarshalAlias AlterFunctionStmt
	return json.Marshal(map[string]interface{}{
		"AlterFunctionStmt": (*AlterFunctionStmtMarshalAlias)(&node),
	})
}

func (node *AlterFunctionStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["func"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["func"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(ObjectWithArgs)
			node.Func = &val
		}
	}

	if fields["actions"] != nil {
		node.Actions.Items, err = UnmarshalNodeArrayJSON(fields["actions"])
		if err != nil {
			return
		}
	}

	return
}
