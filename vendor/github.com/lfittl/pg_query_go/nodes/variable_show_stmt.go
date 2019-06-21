// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 * Show Statement
 * ----------------------
 */
type VariableShowStmt struct {
	Name *string `json:"name"`
}

func (node VariableShowStmt) MarshalJSON() ([]byte, error) {
	type VariableShowStmtMarshalAlias VariableShowStmt
	return json.Marshal(map[string]interface{}{
		"VariableShowStmt": (*VariableShowStmtMarshalAlias)(&node),
	})
}

func (node *VariableShowStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["name"] != nil {
		err = json.Unmarshal(fields["name"], &node.Name)
		if err != nil {
			return
		}
	}

	return
}
