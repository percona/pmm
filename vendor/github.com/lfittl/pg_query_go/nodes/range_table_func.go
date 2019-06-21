// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * RangeTableFunc - raw form of "table functions" such as XMLTABLE
 */
type RangeTableFunc struct {
	Lateral    bool   `json:"lateral"`    /* does it have LATERAL prefix? */
	Docexpr    Node   `json:"docexpr"`    /* document expression */
	Rowexpr    Node   `json:"rowexpr"`    /* row generator expression */
	Namespaces List   `json:"namespaces"` /* list of namespaces as ResTarget */
	Columns    List   `json:"columns"`    /* list of RangeTableFuncCol */
	Alias      *Alias `json:"alias"`      /* table alias & optional column aliases */
	Location   int    `json:"location"`   /* token location, or -1 if unknown */
}

func (node RangeTableFunc) MarshalJSON() ([]byte, error) {
	type RangeTableFuncMarshalAlias RangeTableFunc
	return json.Marshal(map[string]interface{}{
		"RangeTableFunc": (*RangeTableFuncMarshalAlias)(&node),
	})
}

func (node *RangeTableFunc) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["lateral"] != nil {
		err = json.Unmarshal(fields["lateral"], &node.Lateral)
		if err != nil {
			return
		}
	}

	if fields["docexpr"] != nil {
		node.Docexpr, err = UnmarshalNodeJSON(fields["docexpr"])
		if err != nil {
			return
		}
	}

	if fields["rowexpr"] != nil {
		node.Rowexpr, err = UnmarshalNodeJSON(fields["rowexpr"])
		if err != nil {
			return
		}
	}

	if fields["namespaces"] != nil {
		node.Namespaces.Items, err = UnmarshalNodeArrayJSON(fields["namespaces"])
		if err != nil {
			return
		}
	}

	if fields["columns"] != nil {
		node.Columns.Items, err = UnmarshalNodeArrayJSON(fields["columns"])
		if err != nil {
			return
		}
	}

	if fields["alias"] != nil {
		var nodePtr *Node
		nodePtr, err = UnmarshalNodePtrJSON(fields["alias"])
		if err != nil {
			return
		}
		if nodePtr != nil && *nodePtr != nil {
			val := (*nodePtr).(Alias)
			node.Alias = &val
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
