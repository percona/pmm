// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * RangeFunction - function call appearing in a FROM clause
 *
 * functions is a List because we use this to represent the construct
 * ROWS FROM(func1(...), func2(...), ...).  Each element of this list is a
 * two-element sublist, the first element being the untransformed function
 * call tree, and the second element being a possibly-empty list of ColumnDef
 * nodes representing any columndef list attached to that function within the
 * ROWS FROM() syntax.
 *
 * alias and coldeflist represent any alias and/or columndef list attached
 * at the top level.  (We disallow coldeflist appearing both here and
 * per-function, but that's checked in parse analysis, not by the grammar.)
 */
type RangeFunction struct {
	Lateral    bool   `json:"lateral"`     /* does it have LATERAL prefix? */
	Ordinality bool   `json:"ordinality"`  /* does it have WITH ORDINALITY suffix? */
	IsRowsfrom bool   `json:"is_rowsfrom"` /* is result of ROWS FROM() syntax? */
	Functions  List   `json:"functions"`   /* per-function information, see above */
	Alias      *Alias `json:"alias"`       /* table alias & optional column aliases */
	Coldeflist List   `json:"coldeflist"`  /* list of ColumnDef nodes to describe result
	 * of function returning RECORD */
}

func (node RangeFunction) MarshalJSON() ([]byte, error) {
	type RangeFunctionMarshalAlias RangeFunction
	return json.Marshal(map[string]interface{}{
		"RangeFunction": (*RangeFunctionMarshalAlias)(&node),
	})
}

func (node *RangeFunction) UnmarshalJSON(input []byte) (err error) {
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

	if fields["ordinality"] != nil {
		err = json.Unmarshal(fields["ordinality"], &node.Ordinality)
		if err != nil {
			return
		}
	}

	if fields["is_rowsfrom"] != nil {
		err = json.Unmarshal(fields["is_rowsfrom"], &node.IsRowsfrom)
		if err != nil {
			return
		}
	}

	if fields["functions"] != nil {
		node.Functions.Items, err = UnmarshalNodeArrayJSON(fields["functions"])
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

	if fields["coldeflist"] != nil {
		node.Coldeflist.Items, err = UnmarshalNodeArrayJSON(fields["coldeflist"])
		if err != nil {
			return
		}
	}

	return
}
