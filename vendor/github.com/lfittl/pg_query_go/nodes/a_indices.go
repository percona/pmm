// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * A_Indices - array subscript or slice bounds ([idx] or [lidx:uidx])
 *
 * In slice case, either or both of lidx and uidx can be NULL (omitted).
 * In non-slice case, uidx holds the single subscript and lidx is always NULL.
 */
type A_Indices struct {
	IsSlice bool `json:"is_slice"` /* true if slice (i.e., colon present) */
	Lidx    Node `json:"lidx"`     /* slice lower bound, if any */
	Uidx    Node `json:"uidx"`     /* subscript, or slice upper bound if any */
}

func (node A_Indices) MarshalJSON() ([]byte, error) {
	type A_IndicesMarshalAlias A_Indices
	return json.Marshal(map[string]interface{}{
		"A_Indices": (*A_IndicesMarshalAlias)(&node),
	})
}

func (node *A_Indices) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["is_slice"] != nil {
		err = json.Unmarshal(fields["is_slice"], &node.IsSlice)
		if err != nil {
			return
		}
	}

	if fields["lidx"] != nil {
		node.Lidx, err = UnmarshalNodeJSON(fields["lidx"])
		if err != nil {
			return
		}
	}

	if fields["uidx"] != nil {
		node.Uidx, err = UnmarshalNodeJSON(fields["uidx"])
		if err != nil {
			return
		}
	}

	return
}
