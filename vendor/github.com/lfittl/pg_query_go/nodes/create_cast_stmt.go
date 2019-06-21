// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *	CREATE CAST Statement
 * ----------------------
 */
type CreateCastStmt struct {
	Sourcetype *TypeName       `json:"sourcetype"`
	Targettype *TypeName       `json:"targettype"`
	Func       *ObjectWithArgs `json:"func"`
	Context    CoercionContext `json:"context"`
	Inout      bool            `json:"inout"`
}

func (node CreateCastStmt) MarshalJSON() ([]byte, error) {
	type CreateCastStmtMarshalAlias CreateCastStmt
	return json.Marshal(map[string]interface{}{
		"CreateCastStmt": (*CreateCastStmtMarshalAlias)(&node),
	})
}

func (node *CreateCastStmt) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["sourcetype"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["sourcetype"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(TypeName)
			node.Sourcetype = &val
		}
	}

	if fields["targettype"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["targettype"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(TypeName)
			node.Targettype = &val
		}
	}

	if fields["func"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["func"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(ObjectWithArgs)
			node.Func = &val
		}
	}

	if fields["context"] != nil {
		err = json.Unmarshal(fields["context"], &node.Context)
		if err != nil {
			return
		}
	}

	if fields["inout"] != nil {
		err = json.Unmarshal(fields["inout"], &node.Inout)
		if err != nil {
			return
		}
	}

	return
}
