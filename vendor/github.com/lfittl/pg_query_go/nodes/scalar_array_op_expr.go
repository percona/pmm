// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * ScalarArrayOpExpr - expression node for "scalar op ANY/ALL (array)"
 *
 * The operator must yield boolean.  It is applied to the left operand
 * and each element of the righthand array, and the results are combined
 * with OR or AND (for ANY or ALL respectively).  The node representation
 * is almost the same as for the underlying operator, but we need a useOr
 * flag to remember whether it's ANY or ALL, and we don't have to store
 * the result type (or the collation) because it must be boolean.
 */
type ScalarArrayOpExpr struct {
	Xpr         Node `json:"xpr"`
	Opno        Oid  `json:"opno"`        /* PG_OPERATOR OID of the operator */
	Opfuncid    Oid  `json:"opfuncid"`    /* PG_PROC OID of underlying function */
	UseOr       bool `json:"useOr"`       /* true for ANY, false for ALL */
	Inputcollid Oid  `json:"inputcollid"` /* OID of collation that operator should use */
	Args        List `json:"args"`        /* the scalar and array operands */
	Location    int  `json:"location"`    /* token location, or -1 if unknown */
}

func (node ScalarArrayOpExpr) MarshalJSON() ([]byte, error) {
	type ScalarArrayOpExprMarshalAlias ScalarArrayOpExpr
	return json.Marshal(map[string]interface{}{
		"ScalarArrayOpExpr": (*ScalarArrayOpExprMarshalAlias)(&node),
	})
}

func (node *ScalarArrayOpExpr) UnmarshalJSON(input []byte) (err error) {
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

	if fields["opno"] != nil {
		err = json.Unmarshal(fields["opno"], &node.Opno)
		if err != nil {
			return
		}
	}

	if fields["opfuncid"] != nil {
		err = json.Unmarshal(fields["opfuncid"], &node.Opfuncid)
		if err != nil {
			return
		}
	}

	if fields["useOr"] != nil {
		err = json.Unmarshal(fields["useOr"], &node.UseOr)
		if err != nil {
			return
		}
	}

	if fields["inputcollid"] != nil {
		err = json.Unmarshal(fields["inputcollid"], &node.Inputcollid)
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
