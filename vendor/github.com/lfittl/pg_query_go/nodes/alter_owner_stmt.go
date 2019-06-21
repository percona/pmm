// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Alter Object Owner Statement
 * ----------------------
 */
type AlterOwnerStmt struct {
	ObjectType ObjectType `json:"objectType"` /* OBJECT_TABLE, OBJECT_TYPE, etc */
	Relation   *RangeVar  `json:"relation"`   /* in case it's a table */
	Object     Node       `json:"object"`     /* in case it's some other object */
	Newowner   *RoleSpec  `json:"newowner"`   /* the new owner */
}

func (node AlterOwnerStmt) MarshalJSON() ([]byte, error) {
	type AlterOwnerStmtMarshalAlias AlterOwnerStmt
	return json.Marshal(map[string]interface{}{
		"AlterOwnerStmt": (*AlterOwnerStmtMarshalAlias)(&node),
	})
}

func (node *AlterOwnerStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["objectType"] != nil {
		err = json.Unmarshal(fields["objectType"], &node.ObjectType)
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

	return
}
