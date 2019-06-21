// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * A_Star - '*' representing all columns of a table or compound field
 *
 * This can appear within ColumnRef.fields, A_Indirection.indirection, and
 * ResTarget.indirection lists.
 */
type A_Star struct {
}

func (node A_Star) MarshalJSON() ([]byte, error) {
	type A_StarMarshalAlias A_Star
	return json.Marshal(map[string]interface{}{
		"A_Star": (*A_StarMarshalAlias)(&node),
	})
}

func (node *A_Star) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	return
}
