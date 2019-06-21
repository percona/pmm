// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * Placeholder node for the test value to be processed by a CASE expression.
 * This is effectively like a Param, but can be implemented more simply
 * since we need only one replacement value at a time.
 *
 * We also use this in nested UPDATE expressions.
 * See transformAssignmentIndirection().
 */
type CaseTestExpr struct {
	Xpr       Node  `json:"xpr"`
	TypeId    Oid   `json:"typeId"`    /* type for substituted value */
	TypeMod   int32 `json:"typeMod"`   /* typemod for substituted value */
	Collation Oid   `json:"collation"` /* collation for the substituted value */
}

func (node CaseTestExpr) MarshalJSON() ([]byte, error) {
	type CaseTestExprMarshalAlias CaseTestExpr
	return json.Marshal(map[string]interface{}{
		"CaseTestExpr": (*CaseTestExprMarshalAlias)(&node),
	})
}

func (node *CaseTestExpr) UnmarshalJSON(input []byte) (err error) {
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

	if fields["typeId"] != nil {
		err = json.Unmarshal(fields["typeId"], &node.TypeId)
		if err != nil {
			return
		}
	}

	if fields["typeMod"] != nil {
		err = json.Unmarshal(fields["typeMod"], &node.TypeMod)
		if err != nil {
			return
		}
	}

	if fields["collation"] != nil {
		err = json.Unmarshal(fields["collation"], &node.Collation)
		if err != nil {
			return
		}
	}

	return
}
