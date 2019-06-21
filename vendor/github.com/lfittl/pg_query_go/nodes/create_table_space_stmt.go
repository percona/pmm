// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create/Drop Table Space Statements
 * ----------------------
 */
type CreateTableSpaceStmt struct {
	Tablespacename *string   `json:"tablespacename"`
	Owner          *RoleSpec `json:"owner"`
	Location       *string   `json:"location"`
	Options        List      `json:"options"`
}

func (node CreateTableSpaceStmt) MarshalJSON() ([]byte, error) {
	type CreateTableSpaceStmtMarshalAlias CreateTableSpaceStmt
	return json.Marshal(map[string]interface{}{
		"CreateTableSpaceStmt": (*CreateTableSpaceStmtMarshalAlias)(&node),
	})
}

func (node *CreateTableSpaceStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["tablespacename"] != nil {
		err = json.Unmarshal(fields["tablespacename"], &node.Tablespacename)
		if err != nil {
			return
		}
	}

	if fields["owner"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["owner"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RoleSpec)
			node.Owner = &val
		}
	}

	if fields["location"] != nil {
		err = json.Unmarshal(fields["location"], &node.Location)
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
