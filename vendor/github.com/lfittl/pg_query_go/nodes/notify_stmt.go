// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Notify Statement
 * ----------------------
 */
type NotifyStmt struct {
	Conditionname *string `json:"conditionname"` /* condition name to notify */
	Payload       *string `json:"payload"`       /* the payload string, or NULL if none */
}

func (node NotifyStmt) MarshalJSON() ([]byte, error) {
	type NotifyStmtMarshalAlias NotifyStmt
	return json.Marshal(map[string]interface{}{
		"NotifyStmt": (*NotifyStmtMarshalAlias)(&node),
	})
}

func (node *NotifyStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["payload"] != nil {
		err = json.Unmarshal(fields["payload"], &node.Payload)
		if err != nil {
			return
		}
	}

	return
}
