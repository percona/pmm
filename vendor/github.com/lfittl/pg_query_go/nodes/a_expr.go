// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * A_Expr - infix, prefix, and postfix expressions
 */
type A_Expr struct {
	Kind     A_Expr_Kind `json:"kind"`     /* see above */
	Name     List        `json:"name"`     /* possibly-qualified name of operator */
	Lexpr    Node        `json:"lexpr"`    /* left argument, or NULL if none */
	Rexpr    Node        `json:"rexpr"`    /* right argument, or NULL if none */
	Location int         `json:"location"` /* token location, or -1 if unknown */
}

func (node A_Expr) MarshalJSON() ([]byte, error) {
	type A_ExprMarshalAlias A_Expr
	return json.Marshal(map[string]interface{}{
		"A_Expr": (*A_ExprMarshalAlias)(&node),
	})
}

func (node *A_Expr) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["kind"] != nil {
		err = json.Unmarshal(fields["kind"], &node.Kind)
		if err != nil {
			return
		}
	}

	if fields["name"] != nil {
		node.Name.Items, err = UnmarshalNodeArrayJSON(fields["name"])
		if err != nil {
			return
		}
	}

	if fields["lexpr"] != nil {
		node.Lexpr, err = UnmarshalNodeJSON(fields["lexpr"])
		if err != nil {
			return
		}
	}

	if fields["rexpr"] != nil {
		node.Rexpr, err = UnmarshalNodeJSON(fields["rexpr"])
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
