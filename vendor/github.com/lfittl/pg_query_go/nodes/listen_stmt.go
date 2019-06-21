// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Listen Statement
 * ----------------------
 */
type ListenStmt struct {
	Conditionname *string `json:"conditionname"` /* condition name to listen on */
}

func (node ListenStmt) MarshalJSON() ([]byte, error) {
	type ListenStmtMarshalAlias ListenStmt
	return json.Marshal(map[string]interface{}{
		"ListenStmt": (*ListenStmtMarshalAlias)(&node),
	})
}

func (node *ListenStmt) UnmarshalJSON(input []byte) (err error) {
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
