// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*----------------------
 *		Create POLICY Statement
 *----------------------
 */
type CreatePolicyStmt struct {
	PolicyName *string   `json:"policy_name"` /* Policy's name */
	Table      *RangeVar `json:"table"`       /* the table name the policy applies to */
	CmdName    *string   `json:"cmd_name"`    /* the command name the policy applies to */
	Permissive bool      `json:"permissive"`  /* restrictive or permissive policy */
	Roles      List      `json:"roles"`       /* the roles associated with the policy */
	Qual       Node      `json:"qual"`        /* the policy's condition */
	WithCheck  Node      `json:"with_check"`  /* the policy's WITH CHECK condition. */
}

func (node CreatePolicyStmt) MarshalJSON() ([]byte, error) {
	type CreatePolicyStmtMarshalAlias CreatePolicyStmt
	return json.Marshal(map[string]interface{}{
		"CreatePolicyStmt": (*CreatePolicyStmtMarshalAlias)(&node),
	})
}

func (node *CreatePolicyStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["policy_name"] != nil {
		err = json.Unmarshal(fields["policy_name"], &node.PolicyName)
		if err != nil {
			return
		}
	}

	if fields["table"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["table"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RangeVar)
			node.Table = &val
		}
	}

	if fields["cmd_name"] != nil {
		err = json.Unmarshal(fields["cmd_name"], &node.CmdName)
		if err != nil {
			return
		}
	}

	if fields["permissive"] != nil {
		err = json.Unmarshal(fields["permissive"], &node.Permissive)
		if err != nil {
			return
		}
	}

	if fields["roles"] != nil {
		node.Roles.Items, err = UnmarshalNodeArrayJSON(fields["roles"])
		if err != nil {
			return
		}
	}

	if fields["qual"] != nil {
		node.Qual, err = UnmarshalNodeJSON(fields["qual"])
		if err != nil {
			return
		}
	}

	if fields["with_check"] != nil {
		node.WithCheck, err = UnmarshalNodeJSON(fields["with_check"])
		if err != nil {
			return
		}
	}

	return
}
