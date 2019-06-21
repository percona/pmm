// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * ParamRef - specifies a $n parameter reference
 */
type ParamRef struct {
	Number   int `json:"number"`   /* the number of the parameter */
	Location int `json:"location"` /* token location, or -1 if unknown */
}

func (node ParamRef) MarshalJSON() ([]byte, error) {
	type ParamRefMarshalAlias ParamRef
	return json.Marshal(map[string]interface{}{
		"ParamRef": (*ParamRefMarshalAlias)(&node),
	})
}

func (node *ParamRef) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["number"] != nil {
		err = json.Unmarshal(fields["number"], &node.Number)
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
