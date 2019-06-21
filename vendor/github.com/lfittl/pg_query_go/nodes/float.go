// Auto-generated from postgres/src/include/nodes/value.h - DO NOT EDIT

package pg_query

import "encoding/json"

type Float struct {
	Str string `json:"str"`
}

func (node Float) MarshalJSON() ([]byte, error) {
	type FloatMarshalAlias Float
	return json.Marshal(map[string]interface{}{
		"Float": (*FloatMarshalAlias)(&node),
	})
}

func (node *Float) UnmarshalJSON(input []byte) (err error) {
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
