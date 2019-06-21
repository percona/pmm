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
type AlterRoleSetStmt struct {
	Role     *RoleSpec        `json:"role"`     /* role */
	Database *string          `json:"database"` /* database name, or NULL */
	Setstmt  *VariableSetStmt `json:"setstmt"`  /* SET or RESET subcommand */
}

func (node AlterRoleSetStmt) MarshalJSON() ([]byte, error) {
	type AlterRoleSetStmtMarshalAlias AlterRoleSetStmt
	return json.Marshal(map[string]interface{}{
		"AlterRoleSetStmt": (*AlterRoleSetStmtMarshalAlias)(&node),
	})
}

func (node *AlterRoleSetStmt) UnmarshalJSON(input []byte) (err error) {
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

	if fields["database"] != nil {
		err = json.Unmarshal(fields["database"], &node.Database)
		if err != nil {
			return
		}
	}

	if fields["setstmt"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["setstmt"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(VariableSetStmt)
			node.Setstmt = &val
		}
	}

	return
}
