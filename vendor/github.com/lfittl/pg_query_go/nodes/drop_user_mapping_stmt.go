// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create/Drop USER MAPPING Statements
 * ----------------------
 */
type DropUserMappingStmt struct {
	User       *RoleSpec `json:"user"`       /* user role */
	Servername *string   `json:"servername"` /* server name */
	MissingOk  bool      `json:"missing_ok"` /* ignore missing mappings */
}

func (node DropUserMappingStmt) MarshalJSON() ([]byte, error) {
	type DropUserMappingStmtMarshalAlias DropUserMappingStmt
	return json.Marshal(map[string]interface{}{
		"DropUserMappingStmt": (*DropUserMappingStmtMarshalAlias)(&node),
	})
}

func (node *DropUserMappingStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["missing_ok"] != nil {
		err = json.Unmarshal(fields["missing_ok"], &node.MissingOk)
		if err != nil {
			return
		}
	}

	return
}
