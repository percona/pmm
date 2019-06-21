// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		EXECUTE Statement
 * ----------------------
 */
type ExecuteStmt struct {
	Name   *string `json:"name"`   /* The name of the plan to execute */
	Params List    `json:"params"` /* Values to assign to parameters */
}

func (node ExecuteStmt) MarshalJSON() ([]byte, error) {
	type ExecuteStmtMarshalAlias ExecuteStmt
	return json.Marshal(map[string]interface{}{
		"ExecuteStmt": (*ExecuteStmtMarshalAlias)(&node),
	})
}

func (node *ExecuteStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["params"] != nil {
		node.Params.Items, err = UnmarshalNodeArrayJSON(fields["params"])
		if err != nil {
			return
		}
	}

	return
}
