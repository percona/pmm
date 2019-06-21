// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * A_Const - a literal constant
 */
type A_Const struct {
	Val      Node `json:"val"`      /* value (includes type info, see value.h) */
	Location int  `json:"location"` /* token location, or -1 if unknown */
}

func (node A_Const) MarshalJSON() ([]byte, error) {
	type A_ConstMarshalAlias A_Const
	return json.Marshal(map[string]interface{}{
		"A_Const": (*A_ConstMarshalAlias)(&node),
	})
}

func (node *A_Const) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["val"] != nil {
		node.Val, err = UnmarshalNodeJSON(fields["val"])
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
