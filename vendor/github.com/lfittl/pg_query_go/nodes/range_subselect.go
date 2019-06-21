// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * RangeSubselect - subquery appearing in a FROM clause
 */
type RangeSubselect struct {
	Lateral  bool   `json:"lateral"`  /* does it have LATERAL prefix? */
	Subquery Node   `json:"subquery"` /* the untransformed sub-select clause */
	Alias    *Alias `json:"alias"`    /* table alias & optional column aliases */
}

func (node RangeSubselect) MarshalJSON() ([]byte, error) {
	type RangeSubselectMarshalAlias RangeSubselect
	return json.Marshal(map[string]interface{}{
		"RangeSubselect": (*RangeSubselectMarshalAlias)(&node),
	})
}

func (node *RangeSubselect) UnmarshalJSON(input []byte) (err error) {
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

	if fields["subquery"] != nil {
		node.Subquery, err = UnmarshalNodeJSON(fields["subquery"])
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

	return
}
