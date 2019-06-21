// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *	Alter Default Privileges Statement
 * ----------------------
 */
type AlterDefaultPrivilegesStmt struct {
	Options List       `json:"options"` /* list of DefElem */
	Action  *GrantStmt `json:"action"`  /* GRANT/REVOKE action (with objects=NIL) */
}

func (node AlterDefaultPrivilegesStmt) MarshalJSON() ([]byte, error) {
	type AlterDefaultPrivilegesStmtMarshalAlias AlterDefaultPrivilegesStmt
	return json.Marshal(map[string]interface{}{
		"AlterDefaultPrivilegesStmt": (*AlterDefaultPrivilegesStmtMarshalAlias)(&node),
	})
}

func (node *AlterDefaultPrivilegesStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["options"] != nil {
		node.Options.Items, err = UnmarshalNodeArrayJSON(fields["options"])
		if err != nil {
			return
		}
	}

	if fields["action"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["action"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(GrantStmt)
			node.Action = &val
		}
	}

	return
}
