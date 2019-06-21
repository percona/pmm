// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*----------
 * CaseExpr - a CASE expression
 *
 * We support two distinct forms of CASE expression:
 *		CASE WHEN boolexpr THEN expr [ WHEN boolexpr THEN expr ... ]
 *		CASE testexpr WHEN compexpr THEN expr [ WHEN compexpr THEN expr ... ]
 * These are distinguishable by the "arg" field being NULL in the first case
 * and the testexpr in the second case.
 *
 * In the raw grammar output for the second form, the condition expressions
 * of the WHEN clauses are just the comparison values.  Parse analysis
 * converts these to valid boolean expressions of the form
 *		CaseTestExpr '=' compexpr
 * where the CaseTestExpr node is a placeholder that emits the correct
 * value at runtime.  This structure is used so that the testexpr need be
 * evaluated only once.  Note that after parse analysis, the condition
 * expressions always yield boolean.
 *
 * Note: we can test whether a CaseExpr has been through parse analysis
 * yet by checking whether casetype is InvalidOid or not.
 *----------
 */
type CaseExpr struct {
	Xpr        Node `json:"xpr"`
	Casetype   Oid  `json:"casetype"`   /* type of expression result */
	Casecollid Oid  `json:"casecollid"` /* OID of collation, or InvalidOid if none */
	Arg        Node `json:"arg"`        /* implicit equality comparison argument */
	Args       List `json:"args"`       /* the arguments (list of WHEN clauses) */
	Defresult  Node `json:"defresult"`  /* the default result (ELSE clause) */
	Location   int  `json:"location"`   /* token location, or -1 if unknown */
}

func (node CaseExpr) MarshalJSON() ([]byte, error) {
	type CaseExprMarshalAlias CaseExpr
	return json.Marshal(map[string]interface{}{
		"CaseExpr": (*CaseExprMarshalAlias)(&node),
	})
}

func (node *CaseExpr) UnmarshalJSON(input []byte) (err error) {
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

	if fields["casetype"] != nil {
		err = json.Unmarshal(fields["casetype"], &node.Casetype)
		if err != nil {
			return
		}
	}

	if fields["casecollid"] != nil {
		err = json.Unmarshal(fields["casecollid"], &node.Casecollid)
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

	if fields["args"] != nil {
		node.Args.Items, err = UnmarshalNodeArrayJSON(fields["args"])
		if err != nil {
			return
		}
	}

	if fields["defresult"] != nil {
		node.Defresult, err = UnmarshalNodeJSON(fields["defresult"])
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
