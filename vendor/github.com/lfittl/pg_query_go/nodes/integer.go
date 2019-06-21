// Auto-generated from postgres/src/include/nodes/value.h - DO NOT EDIT

package pg_query

import "encoding/json"

type Integer struct {
	Ival int64 `json:"ival"`
}

func (node Integer) MarshalJSON() ([]byte, error) {
	type IntegerMarshalAlias Integer
	return json.Marshal(map[string]interface{}{
		"Integer": (*IntegerMarshalAlias)(&node),
	})
}

func (node *Integer) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["ival"] != nil {
		err = json.Unmarshal(fields["ival"], &node.Ival)
		if err != nil {
			return
		}
	}

	return
}
