// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *	CREATE TRANSFORM Statement
 * ----------------------
 */
type CreateTransformStmt struct {
	Replace  bool            `json:"replace"`
	TypeName *TypeName       `json:"type_name"`
	Lang     *string         `json:"lang"`
	Fromsql  *ObjectWithArgs `json:"fromsql"`
	Tosql    *ObjectWithArgs `json:"tosql"`
}

func (node CreateTransformStmt) MarshalJSON() ([]byte, error) {
	type CreateTransformStmtMarshalAlias CreateTransformStmt
	return json.Marshal(map[string]interface{}{
		"CreateTransformStmt": (*CreateTransformStmtMarshalAlias)(&node),
	})
}

func (node *CreateTransformStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["replace"] != nil {
		err = json.Unmarshal(fields["replace"], &node.Replace)
		if err != nil {
			return
		}
	}

	if fields["type_name"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["type_name"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(TypeName)
			node.TypeName = &val
		}
	}

	if fields["lang"] != nil {
		err = json.Unmarshal(fields["lang"], &node.Lang)
		if err != nil {
			return
		}
	}

	if fields["fromsql"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["fromsql"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(ObjectWithArgs)
			node.Fromsql = &val
		}
	}

	if fields["tosql"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["tosql"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(ObjectWithArgs)
			node.Tosql = &val
		}
	}

	return
}
