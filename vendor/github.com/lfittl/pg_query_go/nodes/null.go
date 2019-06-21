// Auto-generated from postgres/src/include/nodes/value.h - DO NOT EDIT

package pg_query

import "encoding/json"

type Null struct {
}

func (node Null) MarshalJSON() ([]byte, error) {
	type NullMarshalAlias Null
	return json.Marshal(map[string]interface{}{
		"Null": (*NullMarshalAlias)(&node),
	})
}

func (node *Null) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	return
}
