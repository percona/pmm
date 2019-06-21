// Auto-generated from postgres/src/include/nodes/value.h - DO NOT EDIT

package pg_query

import "encoding/json"

type BitString struct {
	Str string `json:"str"`
}

func (node BitString) MarshalJSON() ([]byte, error) {
	type BitStringMarshalAlias BitString
	return json.Marshal(map[string]interface{}{
		"BitString": (*BitStringMarshalAlias)(&node),
	})
}

func (node *BitString) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["str"] != nil {
		err = json.Unmarshal(fields["str"], &node.Str)
		if err != nil {
			return
		}
	}

	return
}
