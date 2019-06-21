// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *	Alter Table
 * ----------------------
 */
type ReplicaIdentityStmt struct {
	IdentityType byte    `json:"identity_type"`
	Name         *string `json:"name"`
}

func (node ReplicaIdentityStmt) MarshalJSON() ([]byte, error) {
	type ReplicaIdentityStmtMarshalAlias ReplicaIdentityStmt
	return json.Marshal(map[string]interface{}{
		"ReplicaIdentityStmt": (*ReplicaIdentityStmtMarshalAlias)(&node),
	})
}

func (node *ReplicaIdentityStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["identity_type"] != nil {
		var strVal string
		err = json.Unmarshal(fields["identity_type"], &strVal)
		node.IdentityType = strVal[0]
		if err != nil {
			return
		}
	}

	if fields["name"] != nil {
		err = json.Unmarshal(fields["name"], &node.Name)
		if err != nil {
			return
		}
	}

	return
}
