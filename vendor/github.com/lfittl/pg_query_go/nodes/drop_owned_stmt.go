// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 *		DROP OWNED statement
 */
type DropOwnedStmt struct {
	Roles    List         `json:"roles"`
	Behavior DropBehavior `json:"behavior"`
}

func (node DropOwnedStmt) MarshalJSON() ([]byte, error) {
	type DropOwnedStmtMarshalAlias DropOwnedStmt
	return json.Marshal(map[string]interface{}{
		"DropOwnedStmt": (*DropOwnedStmtMarshalAlias)(&node),
	})
}

func (node *DropOwnedStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["roles"] != nil {
		node.Roles.Items, err = UnmarshalNodeArrayJSON(fields["roles"])
		if err != nil {
			return
		}
	}

	if fields["behavior"] != nil {
		err = json.Unmarshal(fields["behavior"], &node.Behavior)
		if err != nil {
			return
		}
	}

	return
}
