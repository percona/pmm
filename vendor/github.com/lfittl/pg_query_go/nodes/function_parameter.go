// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------------
 *		Create Function Statement
 * ----------------------
 */
type FunctionParameter struct {
	Name    *string               `json:"name"`    /* parameter name, or NULL if not given */
	ArgType *TypeName             `json:"argType"` /* TypeName for parameter type */
	Mode    FunctionParameterMode `json:"mode"`    /* IN/OUT/etc */
	Defexpr Node                  `json:"defexpr"` /* raw default expr, or NULL if not given */
}

func (node FunctionParameter) MarshalJSON() ([]byte, error) {
	type FunctionParameterMarshalAlias FunctionParameter
	return json.Marshal(map[string]interface{}{
		"FunctionParameter": (*FunctionParameterMarshalAlias)(&node),
	})
}

func (node *FunctionParameter) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["name"] != nil {
		err = json.Unmarshal(fields["name"], &node.Name)
		if err != nil {
			return
		}
	}

	if fields["argType"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["argType"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(TypeName)
			node.ArgType = &val
		}
	}

	if fields["mode"] != nil {
		err = json.Unmarshal(fields["mode"], &node.Mode)
		if err != nil {
			return
		}
	}

	if fields["defexpr"] != nil {
		node.Defexpr, err = UnmarshalNodeJSON(fields["defexpr"])
		if err != nil {
			return
		}
	}

	return
}
