// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Unlisten Statement
 * ----------------------
 */
type UnlistenStmt struct {
	Conditionname *string `json:"conditionname"` /* name to unlisten on, or NULL for all */
}

func (node UnlistenStmt) MarshalJSON() ([]byte, error) {
	type UnlistenStmtMarshalAlias UnlistenStmt
	return json.Marshal(map[string]interface{}{
		"UnlistenStmt": (*UnlistenStmtMarshalAlias)(&node),
	})
}

func (node *UnlistenStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["conditionname"] != nil {
		err = json.Unmarshal(fields["conditionname"], &node.Conditionname)
		if err != nil {
			return
		}
	}

	return
}
