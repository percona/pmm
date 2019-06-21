// Auto-generated from postgres/src/include/nodes/pg_list.h - DO NOT EDIT

package pg_query

import "encoding/json"

type List struct {
	Items []Node `json:"items"`
}

func (node List) MarshalJSON() ([]byte, error) {
	type ListMarshalAlias List
	return json.Marshal(map[string]interface{}{
		"List": (*ListMarshalAlias)(&node),
	})
}

func (node *List) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["items"] != nil {
		node.Items, err = UnmarshalNodeArrayJSON(fields["items"])
		if err != nil {
			return
		}
	}

	return
}
