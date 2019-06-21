// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *	Alter Table
 * ----------------------
 */
type AlterTableStmt struct {
	Relation  *RangeVar  `json:"relation"`   /* table to work on */
	Cmds      List       `json:"cmds"`       /* list of subcommands */
	Relkind   ObjectType `json:"relkind"`    /* type of object */
	MissingOk bool       `json:"missing_ok"` /* skip error if table missing */
}

func (node AlterTableStmt) MarshalJSON() ([]byte, error) {
	type AlterTableStmtMarshalAlias AlterTableStmt
	return json.Marshal(map[string]interface{}{
		"AlterTableStmt": (*AlterTableStmtMarshalAlias)(&node),
	})
}

func (node *AlterTableStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["relation"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["relation"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RangeVar)
			node.Relation = &val
		}
	}

	if fields["cmds"] != nil {
		node.Cmds.Items, err = UnmarshalNodeArrayJSON(fields["cmds"])
		if err != nil {
			return
		}
	}

	if fields["relkind"] != nil {
		err = json.Unmarshal(fields["relkind"], &node.Relkind)
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
