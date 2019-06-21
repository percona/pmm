// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *	Create/Alter/Drop Role Statements
 *
 * Note: these node types are also used for the backwards-compatible
 * Create/Alter/Drop User/Group statements.  In the ALTER and DROP cases
 * there's really no need to distinguish what the original spelling was,
 * but for CREATE we mark the type because the defaults vary.
 * ----------------------
 */
type AlterRoleStmt struct {
	Role    *RoleSpec `json:"role"`    /* role */
	Options List      `json:"options"` /* List of DefElem nodes */
	Action  int       `json:"action"`  /* +1 = add members, -1 = drop members */
}

func (node AlterRoleStmt) MarshalJSON() ([]byte, error) {
	type AlterRoleStmtMarshalAlias AlterRoleStmt
	return json.Marshal(map[string]interface{}{
		"AlterRoleStmt": (*AlterRoleStmtMarshalAlias)(&node),
	})
}

func (node *AlterRoleStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["role"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["role"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RoleSpec)
			node.Role = &val
		}
	}

	if fields["options"] != nil {
		node.Options.Items, err = UnmarshalNodeArrayJSON(fields["options"])
		if err != nil {
			return
		}
	}

	if fields["action"] != nil {
		err = json.Unmarshal(fields["action"], &node.Action)
		if err != nil {
			return
		}
	}

	return
}
