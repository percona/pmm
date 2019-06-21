// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Alter Object Rename Statement
 * ----------------------
 */
type RenameStmt struct {
	RenameType   ObjectType `json:"renameType"`   /* OBJECT_TABLE, OBJECT_COLUMN, etc */
	RelationType ObjectType `json:"relationType"` /* if column name, associated relation type */
	Relation     *RangeVar  `json:"relation"`     /* in case it's a table */
	Object       Node       `json:"object"`       /* in case it's some other object */
	Subname      *string    `json:"subname"`      /* name of contained object (column, rule,
	 * trigger, etc) */

	Newname   *string      `json:"newname"`    /* the new name */
	Behavior  DropBehavior `json:"behavior"`   /* RESTRICT or CASCADE behavior */
	MissingOk bool         `json:"missing_ok"` /* skip error if missing? */
}

func (node RenameStmt) MarshalJSON() ([]byte, error) {
	type RenameStmtMarshalAlias RenameStmt
	return json.Marshal(map[string]interface{}{
		"RenameStmt": (*RenameStmtMarshalAlias)(&node),
	})
}

func (node *RenameStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["renameType"] != nil {
		err = json.Unmarshal(fields["renameType"], &node.RenameType)
		if err != nil {
			return
		}
	}

	if fields["relationType"] != nil {
		err = json.Unmarshal(fields["relationType"], &node.RelationType)
		if err != nil {
			return
		}
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

	if fields["object"] != nil {
		node.Object, err = UnmarshalNodeJSON(fields["object"])
		if err != nil {
			return
		}
	}

	if fields["subname"] != nil {
		err = json.Unmarshal(fields["subname"], &node.Subname)
		if err != nil {
			return
		}
	}

	if fields["newname"] != nil {
		err = json.Unmarshal(fields["newname"], &node.Newname)
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

	if fields["missing_ok"] != nil {
		err = json.Unmarshal(fields["missing_ok"], &node.MissingOk)
		if err != nil {
			return
		}
	}

	return
}
