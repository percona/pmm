// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Grant/Revoke Role Statement
 *
 * Note: because of the parsing ambiguity with the GRANT <privileges>
 * statement, granted_roles is a list of AccessPriv; the execution code
 * should complain if any column lists appear.  grantee_roles is a list
 * of role names, as Value strings.
 * ----------------------
 */
type GrantRoleStmt struct {
	GrantedRoles List         `json:"granted_roles"` /* list of roles to be granted/revoked */
	GranteeRoles List         `json:"grantee_roles"` /* list of member roles to add/delete */
	IsGrant      bool         `json:"is_grant"`      /* true = GRANT, false = REVOKE */
	AdminOpt     bool         `json:"admin_opt"`     /* with admin option */
	Grantor      *RoleSpec    `json:"grantor"`       /* set grantor to other than current role */
	Behavior     DropBehavior `json:"behavior"`      /* drop behavior (for REVOKE) */
}

func (node GrantRoleStmt) MarshalJSON() ([]byte, error) {
	type GrantRoleStmtMarshalAlias GrantRoleStmt
	return json.Marshal(map[string]interface{}{
		"GrantRoleStmt": (*GrantRoleStmtMarshalAlias)(&node),
	})
}

func (node *GrantRoleStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["granted_roles"] != nil {
		node.GrantedRoles.Items, err = UnmarshalNodeArrayJSON(fields["granted_roles"])
		if err != nil {
			return
		}
	}

	if fields["grantee_roles"] != nil {
		node.GranteeRoles.Items, err = UnmarshalNodeArrayJSON(fields["grantee_roles"])
		if err != nil {
			return
		}
	}

	if fields["is_grant"] != nil {
		err = json.Unmarshal(fields["is_grant"], &node.IsGrant)
		if err != nil {
			return
		}
	}

	if fields["admin_opt"] != nil {
		err = json.Unmarshal(fields["admin_opt"], &node.AdminOpt)
		if err != nil {
			return
		}
	}

	if fields["grantor"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["grantor"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RoleSpec)
			node.Grantor = &val
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
