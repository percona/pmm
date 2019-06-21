// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * CoalesceExpr - a COALESCE expression
 */
type CoalesceExpr struct {
	Xpr            Node `json:"xpr"`
	Coalescetype   Oid  `json:"coalescetype"`   /* type of expression result */
	Coalescecollid Oid  `json:"coalescecollid"` /* OID of collation, or InvalidOid if none */
	Args           List `json:"args"`           /* the arguments */
	Location       int  `json:"location"`       /* token location, or -1 if unknown */
}

func (node CoalesceExpr) MarshalJSON() ([]byte, error) {
	type CoalesceExprMarshalAlias CoalesceExpr
	return json.Marshal(map[string]interface{}{
		"CoalesceExpr": (*CoalesceExprMarshalAlias)(&node),
	})
}

func (node *CoalesceExpr) UnmarshalJSON(input []byte) (err error) {
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

	if fields["coalescetype"] != nil {
		err = json.Unmarshal(fields["coalescetype"], &node.Coalescetype)
		if err != nil {
			return
		}
	}

	if fields["coalescecollid"] != nil {
		err = json.Unmarshal(fields["coalescecollid"], &node.Coalescecollid)
		if err != nil {
			return
		}
	}

	if fields["args"] != nil {
		node.Args.Items, err = UnmarshalNodeArrayJSON(fields["args"])
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
