// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create/Drop USER MAPPING Statements
 * ----------------------
 */
type AlterUserMappingStmt struct {
	User       *RoleSpec `json:"user"`       /* user role */
	Servername *string   `json:"servername"` /* server name */
	Options    List      `json:"options"`    /* generic options to server */
}

func (node AlterUserMappingStmt) MarshalJSON() ([]byte, error) {
	type AlterUserMappingStmtMarshalAlias AlterUserMappingStmt
	return json.Marshal(map[string]interface{}{
		"AlterUserMappingStmt": (*AlterUserMappingStmtMarshalAlias)(&node),
	})
}

func (node *AlterUserMappingStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["user"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["user"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RoleSpec)
			node.User = &val
		}
	}

	if fields["servername"] != nil {
		err = json.Unmarshal(fields["servername"], &node.Servername)
		if err != nil {
			return
		}
	}

	if fields["options"] != nil {
		node.Options.Items, err = UnmarshalNodeArrayJSON(fields["options"])
		if err != nil {
			return
		}
	}

	return
}
