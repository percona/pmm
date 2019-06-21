// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create/Drop USER MAPPING Statements
 * ----------------------
 */
type CreateUserMappingStmt struct {
	User        *RoleSpec `json:"user"`          /* user role */
	Servername  *string   `json:"servername"`    /* server name */
	IfNotExists bool      `json:"if_not_exists"` /* just do nothing if it already exists? */
	Options     List      `json:"options"`       /* generic options to server */
}

func (node CreateUserMappingStmt) MarshalJSON() ([]byte, error) {
	type CreateUserMappingStmtMarshalAlias CreateUserMappingStmt
	return json.Marshal(map[string]interface{}{
		"CreateUserMappingStmt": (*CreateUserMappingStmtMarshalAlias)(&node),
	})
}

func (node *CreateUserMappingStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["if_not_exists"] != nil {
		err = json.Unmarshal(fields["if_not_exists"], &node.IfNotExists)
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
