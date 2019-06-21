// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

type AlterTableMoveAllStmt struct {
	OrigTablespacename *string    `json:"orig_tablespacename"`
	Objtype            ObjectType `json:"objtype"` /* Object type to move */
	Roles              List       `json:"roles"`   /* List of roles to move objects of */
	NewTablespacename  *string    `json:"new_tablespacename"`
	Nowait             bool       `json:"nowait"`
}

func (node AlterTableMoveAllStmt) MarshalJSON() ([]byte, error) {
	type AlterTableMoveAllStmtMarshalAlias AlterTableMoveAllStmt
	return json.Marshal(map[string]interface{}{
		"AlterTableMoveAllStmt": (*AlterTableMoveAllStmtMarshalAlias)(&node),
	})
}

func (node *AlterTableMoveAllStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["orig_tablespacename"] != nil {
		err = json.Unmarshal(fields["orig_tablespacename"], &node.OrigTablespacename)
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

	if fields["roles"] != nil {
		node.Roles.Items, err = UnmarshalNodeArrayJSON(fields["roles"])
		if err != nil {
			return
		}
	}

	if fields["new_tablespacename"] != nil {
		err = json.Unmarshal(fields["new_tablespacename"], &node.NewTablespacename)
		if err != nil {
			return
		}
	}

	if fields["nowait"] != nil {
		err = json.Unmarshal(fields["nowait"], &node.Nowait)
		if err != nil {
			return
		}
	}

	return
}
