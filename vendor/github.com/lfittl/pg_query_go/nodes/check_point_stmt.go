// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 * Checkpoint Statement
 * ----------------------
 */
type CheckPointStmt struct {
}

func (node CheckPointStmt) MarshalJSON() ([]byte, error) {
	type CheckPointStmtMarshalAlias CheckPointStmt
	return json.Marshal(map[string]interface{}{
		"CheckPointStmt": (*CheckPointStmtMarshalAlias)(&node),
	})
}

func (node *CheckPointStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	return
}
