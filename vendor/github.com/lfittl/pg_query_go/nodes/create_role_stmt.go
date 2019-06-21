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
type CreateRoleStmt struct {
	StmtType RoleStmtType `json:"stmt_type"` /* ROLE/USER/GROUP */
	Role     *string      `json:"role"`      /* role name */
	Options  List         `json:"options"`   /* List of DefElem nodes */
}

func (node CreateRoleStmt) MarshalJSON() ([]byte, error) {
	type CreateRoleStmtMarshalAlias CreateRoleStmt
	return json.Marshal(map[string]interface{}{
		"CreateRoleStmt": (*CreateRoleStmtMarshalAlias)(&node),
	})
}

func (node *CreateRoleStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["stmt_type"] != nil {
		err = json.Unmarshal(fields["stmt_type"], &node.StmtType)
		if err != nil {
			return
		}
	}

	if fields["role"] != nil {
		err = json.Unmarshal(fields["role"], &node.Role)
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
