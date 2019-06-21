// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------
 * ConvertRowtypeExpr
 *
 * ConvertRowtypeExpr represents a type coercion from one composite type
 * to another, where the source type is guaranteed to contain all the columns
 * needed for the destination type plus possibly others; the columns need not
 * be in the same positions, but are matched up by name.  This is primarily
 * used to convert a whole-row value of an inheritance child table into a
 * valid whole-row value of its parent table's rowtype.
 * ----------------
 */
type ConvertRowtypeExpr struct {
	Xpr        Node `json:"xpr"`
	Arg        Node `json:"arg"`        /* input expression */
	Resulttype Oid  `json:"resulttype"` /* output type (always a composite type) */

	/* Like RowExpr, we deliberately omit a typmod and collation here */
	Convertformat CoercionForm `json:"convertformat"` /* how to display this node */
	Location      int          `json:"location"`      /* token location, or -1 if unknown */
}

func (node ConvertRowtypeExpr) MarshalJSON() ([]byte, error) {
	type ConvertRowtypeExprMarshalAlias ConvertRowtypeExpr
	return json.Marshal(map[string]interface{}{
		"ConvertRowtypeExpr": (*ConvertRowtypeExprMarshalAlias)(&node),
	})
}

func (node *ConvertRowtypeExpr) UnmarshalJSON(input []byte) (err error) {
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

	if fields["resulttype"] != nil {
		err = json.Unmarshal(fields["resulttype"], &node.Resulttype)
		if err != nil {
			return
		}
	}

	if fields["convertformat"] != nil {
		err = json.Unmarshal(fields["convertformat"], &node.Convertformat)
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
