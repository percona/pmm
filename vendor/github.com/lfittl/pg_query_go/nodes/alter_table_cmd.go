// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *	Alter Table
 * ----------------------
 */
type AlterTableCmd struct {
	Subtype AlterTableType `json:"subtype"` /* Type of table alteration to apply */
	Name    *string        `json:"name"`    /* column, constraint, or trigger to act on,
	 * or tablespace */

	Newowner *RoleSpec `json:"newowner"`
	Def      Node      `json:"def"` /* definition of new column, index,
	 * constraint, or parent table */

	Behavior  DropBehavior `json:"behavior"`   /* RESTRICT or CASCADE for DROP cases */
	MissingOk bool         `json:"missing_ok"` /* skip error if missing? */
}

func (node AlterTableCmd) MarshalJSON() ([]byte, error) {
	type AlterTableCmdMarshalAlias AlterTableCmd
	return json.Marshal(map[string]interface{}{
		"AlterTableCmd": (*AlterTableCmdMarshalAlias)(&node),
	})
}

func (node *AlterTableCmd) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["subtype"] != nil {
		err = json.Unmarshal(fields["subtype"], &node.Subtype)
		if err != nil {
			return
		}
	}

	if fields["name"] != nil {
		err = json.Unmarshal(fields["name"], &node.Name)
		if err != nil {
			return
		}
	}

	if fields["newowner"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["newowner"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(RoleSpec)
			node.Newowner = &val
		}
	}

	if fields["def"] != nil {
		node.Def, err = UnmarshalNodeJSON(fields["def"])
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
