// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * RangeTableFuncCol - one column in a RangeTableFunc->columns
 *
 * If for_ordinality is true (FOR ORDINALITY), then the column is an int4
 * column and the rest of the fields are ignored.
 */
type RangeTableFuncCol struct {
	Colname       *string   `json:"colname"`        /* name of generated column */
	TypeName      *TypeName `json:"typeName"`       /* type of generated column */
	ForOrdinality bool      `json:"for_ordinality"` /* does it have FOR ORDINALITY? */
	IsNotNull     bool      `json:"is_not_null"`    /* does it have NOT NULL? */
	Colexpr       Node      `json:"colexpr"`        /* column filter expression */
	Coldefexpr    Node      `json:"coldefexpr"`     /* column default value expression */
	Location      int       `json:"location"`       /* token location, or -1 if unknown */
}

func (node RangeTableFuncCol) MarshalJSON() ([]byte, error) {
	type RangeTableFuncColMarshalAlias RangeTableFuncCol
	return json.Marshal(map[string]interface{}{
		"RangeTableFuncCol": (*RangeTableFuncColMarshalAlias)(&node),
	})
}

func (node *RangeTableFuncCol) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["colname"] != nil {
		err = json.Unmarshal(fields["colname"], &node.Colname)
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

	if fields["for_ordinality"] != nil {
		err = json.Unmarshal(fields["for_ordinality"], &node.ForOrdinality)
		if err != nil {
			return
		}
	}

	if fields["is_not_null"] != nil {
		err = json.Unmarshal(fields["is_not_null"], &node.IsNotNull)
		if err != nil {
			return
		}
	}

	if fields["colexpr"] != nil {
		node.Colexpr, err = UnmarshalNodeJSON(fields["colexpr"])
		if err != nil {
			return
		}
	}

	if fields["coldefexpr"] != nil {
		node.Coldefexpr, err = UnmarshalNodeJSON(fields["coldefexpr"])
		if err != nil {
			return
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
