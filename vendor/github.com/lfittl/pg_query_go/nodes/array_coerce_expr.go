// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------
 * ArrayCoerceExpr
 *
 * ArrayCoerceExpr represents a type coercion from one array type to another,
 * which is implemented by applying the indicated element-type coercion
 * function to each element of the source array.  If elemfuncid is InvalidOid
 * then the element types are binary-compatible, but the coercion still
 * requires some effort (we have to fix the element type ID stored in the
 * array header).
 * ----------------
 */
type ArrayCoerceExpr struct {
	Xpr          Node         `json:"xpr"`
	Arg          Node         `json:"arg"`          /* input expression (yields an array) */
	Elemfuncid   Oid          `json:"elemfuncid"`   /* OID of element coercion function, or 0 */
	Resulttype   Oid          `json:"resulttype"`   /* output type of coercion (an array type) */
	Resulttypmod int32        `json:"resulttypmod"` /* output typmod (also element typmod) */
	Resultcollid Oid          `json:"resultcollid"` /* OID of collation, or InvalidOid if none */
	IsExplicit   bool         `json:"isExplicit"`   /* conversion semantics flag to pass to func */
	Coerceformat CoercionForm `json:"coerceformat"` /* how to display this node */
	Location     int          `json:"location"`     /* token location, or -1 if unknown */
}

func (node ArrayCoerceExpr) MarshalJSON() ([]byte, error) {
	type ArrayCoerceExprMarshalAlias ArrayCoerceExpr
	return json.Marshal(map[string]interface{}{
		"ArrayCoerceExpr": (*ArrayCoerceExprMarshalAlias)(&node),
	})
}

func (node *ArrayCoerceExpr) UnmarshalJSON(input []byte) (err error) {
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

	if fields["elemfuncid"] != nil {
		err = json.Unmarshal(fields["elemfuncid"], &node.Elemfuncid)
		if err != nil {
			return
		}
	}

	if fields["resulttype"] != nil {
		err = json.Unmarshal(fields["resulttype"], &node.Resulttype)
		if err != nil {
			return
		}
	}

	if fields["resulttypmod"] != nil {
		err = json.Unmarshal(fields["resulttypmod"], &node.Resulttypmod)
		if err != nil {
			return
		}
	}

	if fields["resultcollid"] != nil {
		err = json.Unmarshal(fields["resultcollid"], &node.Resultcollid)
		if err != nil {
			return
		}
	}

	if fields["isExplicit"] != nil {
		err = json.Unmarshal(fields["isExplicit"], &node.IsExplicit)
		if err != nil {
			return
		}
	}

	if fields["coerceformat"] != nil {
		err = json.Unmarshal(fields["coerceformat"], &node.Coerceformat)
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
