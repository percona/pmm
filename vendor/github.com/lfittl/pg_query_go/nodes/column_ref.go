// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * ColumnRef - specifies a reference to a column, or possibly a whole tuple
 *
 * The "fields" list must be nonempty.  It can contain string Value nodes
 * (representing names) and A_Star nodes (representing occurrence of a '*').
 * Currently, A_Star must appear only as the last list element --- the grammar
 * is responsible for enforcing this!
 *
 * Note: any array subscripting or selection of fields from composite columns
 * is represented by an A_Indirection node above the ColumnRef.  However,
 * for simplicity in the normal case, initial field selection from a table
 * name is represented within ColumnRef and not by adding A_Indirection.
 */
type ColumnRef struct {
	Fields   List `json:"fields"`   /* field names (Value strings) or A_Star */
	Location int  `json:"location"` /* token location, or -1 if unknown */
}

func (node ColumnRef) MarshalJSON() ([]byte, error) {
	type ColumnRefMarshalAlias ColumnRef
	return json.Marshal(map[string]interface{}{
		"ColumnRef": (*ColumnRefMarshalAlias)(&node),
	})
}

func (node *ColumnRef) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["fields"] != nil {
		node.Fields.Items, err = UnmarshalNodeArrayJSON(fields["fields"])
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
