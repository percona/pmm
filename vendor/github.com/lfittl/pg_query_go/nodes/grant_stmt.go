// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Grant|Revoke Statement
 * ----------------------
 */
type GrantStmt struct {
	IsGrant  bool            `json:"is_grant"` /* true = GRANT, false = REVOKE */
	Targtype GrantTargetType `json:"targtype"` /* type of the grant target */
	Objtype  GrantObjectType `json:"objtype"`  /* kind of object being operated on */
	Objects  List            `json:"objects"`  /* list of RangeVar nodes, ObjectWithArgs
	 * nodes, or plain names (as Value strings) */

	Privileges List `json:"privileges"` /* list of AccessPriv nodes */

	/* privileges == NIL denotes ALL PRIVILEGES */
	Grantees    List         `json:"grantees"`     /* list of RoleSpec nodes */
	GrantOption bool         `json:"grant_option"` /* grant or revoke grant option */
	Behavior    DropBehavior `json:"behavior"`     /* drop behavior (for REVOKE) */
}

func (node GrantStmt) MarshalJSON() ([]byte, error) {
	type GrantStmtMarshalAlias GrantStmt
	return json.Marshal(map[string]interface{}{
		"GrantStmt": (*GrantStmtMarshalAlias)(&node),
	})
}

func (node *GrantStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["is_grant"] != nil {
		err = json.Unmarshal(fields["is_grant"], &node.IsGrant)
		if err != nil {
			return
		}
	}

	if fields["targtype"] != nil {
		err = json.Unmarshal(fields["targtype"], &node.Targtype)
		if err != nil {
			return
		}
	}

	if fields["objtype"] != nil {
		err = json.Unmarshal(fields["objtype"], &node.Objtype)
		if err != nil {
			return
		}
	}

	if fields["objects"] != nil {
		node.Objects.Items, err = UnmarshalNodeArrayJSON(fields["objects"])
		if err != nil {
			return
		}
	}

	if fields["privileges"] != nil {
		node.Privileges.Items, err = UnmarshalNodeArrayJSON(fields["privileges"])
		if err != nil {
			return
		}
	}

	if fields["grantees"] != nil {
		node.Grantees.Items, err = UnmarshalNodeArrayJSON(fields["grantees"])
		if err != nil {
			return
		}
	}

	if fields["grant_option"] != nil {
		err = json.Unmarshal(fields["grant_option"], &node.GrantOption)
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
