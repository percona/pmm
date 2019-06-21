// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

type SQLValueFunction struct {
	Xpr      Node               `json:"xpr"`
	Op       SQLValueFunctionOp `json:"op"`   /* which function this is */
	Type     Oid                `json:"type"` /* result type/typmod */
	Typmod   int32              `json:"typmod"`
	Location int                `json:"location"` /* token location, or -1 if unknown */
}

func (node SQLValueFunction) MarshalJSON() ([]byte, error) {
	type SQLValueFunctionMarshalAlias SQLValueFunction
	return json.Marshal(map[string]interface{}{
		"SQLValueFunction": (*SQLValueFunctionMarshalAlias)(&node),
	})
}

func (node *SQLValueFunction) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["xpr"] != nil {
		node.Xpr, err = UnmarshalNodeJSON(fields["xpr"])
		if err != nil {
			return
		}
	}

	if fields["op"] != nil {
		err = json.Unmarshal(fields["op"], &node.Op)
		if err != nil {
			return
		}
	}

	if fields["type"] != nil {
		err = json.Unmarshal(fields["type"], &node.Type)
		if err != nil {
			return
		}
	}

	if fields["typmod"] != nil {
		err = json.Unmarshal(fields["typmod"], &node.Typmod)
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
