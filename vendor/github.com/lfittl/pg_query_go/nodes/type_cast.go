// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * TypeCast - a CAST expression
 */
type TypeCast struct {
	Arg      Node      `json:"arg"`      /* the expression being casted */
	TypeName *TypeName `json:"typeName"` /* the target type */
	Location int       `json:"location"` /* token location, or -1 if unknown */
}

func (node TypeCast) MarshalJSON() ([]byte, error) {
	type TypeCastMarshalAlias TypeCast
	return json.Marshal(map[string]interface{}{
		"TypeCast": (*TypeCastMarshalAlias)(&node),
	})
}

func (node *TypeCast) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["arg"] != nil {
		node.Arg, err = UnmarshalNodeJSON(fields["arg"])
		if err != nil {
			return
		}
	}

	if fields["typeName"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["typeName"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(TypeName)
			node.TypeName = &val
		}
	}

	if fields["location"] != nil {
		err = json.Unmarshal(fields["location"], &node.Location)
		if err != nil {
			return
		}
	}

	return
}
