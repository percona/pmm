// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * CaseWhen - one arm of a CASE expression
 */
type CaseWhen struct {
	Xpr      Node `json:"xpr"`
	Expr     Node `json:"expr"`     /* condition expression */
	Result   Node `json:"result"`   /* substitution result */
	Location int  `json:"location"` /* token location, or -1 if unknown */
}

func (node CaseWhen) MarshalJSON() ([]byte, error) {
	type CaseWhenMarshalAlias CaseWhen
	return json.Marshal(map[string]interface{}{
		"CaseWhen": (*CaseWhenMarshalAlias)(&node),
	})
}

func (node *CaseWhen) UnmarshalJSON(input []byte) (err error) {
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

	if fields["expr"] != nil {
		node.Expr, err = UnmarshalNodeJSON(fields["expr"])
		if err != nil {
			return
		}
	}

	if fields["result"] != nil {
		node.Result, err = UnmarshalNodeJSON(fields["result"])
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
