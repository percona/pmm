// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * A_Indirection - select a field and/or array element from an expression
 *
 * The indirection list can contain A_Indices nodes (representing
 * subscripting), string Value nodes (representing field selection --- the
 * string value is the name of the field to select), and A_Star nodes
 * (representing selection of all fields of a composite type).
 * For example, a complex selection operation like
 *				(foo).field1[42][7].field2
 * would be represented with a single A_Indirection node having a 4-element
 * indirection list.
 *
 * Currently, A_Star must appear only as the last list element --- the grammar
 * is responsible for enforcing this!
 */
type A_Indirection struct {
	Arg         Node `json:"arg"`         /* the thing being selected from */
	Indirection List `json:"indirection"` /* subscripts and/or field names and/or * */
}

func (node A_Indirection) MarshalJSON() ([]byte, error) {
	type A_IndirectionMarshalAlias A_Indirection
	return json.Marshal(map[string]interface{}{
		"A_Indirection": (*A_IndirectionMarshalAlias)(&node),
	})
}

func (node *A_Indirection) UnmarshalJSON(input []byte) (err error) {
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

	if fields["indirection"] != nil {
		node.Indirection.Items, err = UnmarshalNodeArrayJSON(fields["indirection"])
		if err != nil {
			return
		}
	}

	return
}
