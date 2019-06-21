// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*----------
 * CollateExpr - COLLATE
 *
 * The planner replaces CollateExpr with RelabelType during expression
 * preprocessing, so execution never sees a CollateExpr.
 *----------
 */
type CollateExpr struct {
	Xpr      Node `json:"xpr"`
	Arg      Node `json:"arg"`      /* input expression */
	CollOid  Oid  `json:"collOid"`  /* collation's OID */
	Location int  `json:"location"` /* token location, or -1 if unknown */
}

func (node CollateExpr) MarshalJSON() ([]byte, error) {
	type CollateExprMarshalAlias CollateExpr
	return json.Marshal(map[string]interface{}{
		"CollateExpr": (*CollateExprMarshalAlias)(&node),
	})
}

func (node *CollateExpr) UnmarshalJSON(input []byte) (err error) {
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

	if fields["arg"] != nil {
		node.Arg, err = UnmarshalNodeJSON(fields["arg"])
		if err != nil {
			return
		}
	}

	if fields["collOid"] != nil {
		err = json.Unmarshal(fields["collOid"], &node.CollOid)
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
