// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Alter System Statement
 * ----------------------
 */
type AlterSystemStmt struct {
	Setstmt *VariableSetStmt `json:"setstmt"` /* SET subcommand */
}

func (node AlterSystemStmt) MarshalJSON() ([]byte, error) {
	type AlterSystemStmtMarshalAlias AlterSystemStmt
	return json.Marshal(map[string]interface{}{
		"AlterSystemStmt": (*AlterSystemStmtMarshalAlias)(&node),
	})
}

func (node *AlterSystemStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["setstmt"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["setstmt"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(VariableSetStmt)
			node.Setstmt = &val
		}
	}

	return
}
