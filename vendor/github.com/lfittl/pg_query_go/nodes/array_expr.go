// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * ArrayExpr - an ARRAY[] expression
 *
 * Note: if multidims is false, the constituent expressions all yield the
 * scalar type identified by element_typeid.  If multidims is true, the
 * constituent expressions all yield arrays of element_typeid (ie, the same
 * type as array_typeid); at runtime we must check for compatible subscripts.
 */
type ArrayExpr struct {
	Xpr           Node `json:"xpr"`
	ArrayTypeid   Oid  `json:"array_typeid"`   /* type of expression result */
	ArrayCollid   Oid  `json:"array_collid"`   /* OID of collation, or InvalidOid if none */
	ElementTypeid Oid  `json:"element_typeid"` /* common type of array elements */
	Elements      List `json:"elements"`       /* the array elements or sub-arrays */
	Multidims     bool `json:"multidims"`      /* true if elements are sub-arrays */
	Location      int  `json:"location"`       /* token location, or -1 if unknown */
}

func (node ArrayExpr) MarshalJSON() ([]byte, error) {
	type ArrayExprMarshalAlias ArrayExpr
	return json.Marshal(map[string]interface{}{
		"ArrayExpr": (*ArrayExprMarshalAlias)(&node),
	})
}

func (node *ArrayExpr) UnmarshalJSON(input []byte) (err error) {
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

	if fields["array_typeid"] != nil {
		err = json.Unmarshal(fields["array_typeid"], &node.ArrayTypeid)
		if err != nil {
			return
		}
	}

	if fields["array_collid"] != nil {
		err = json.Unmarshal(fields["array_collid"], &node.ArrayCollid)
		if err != nil {
			return
		}
	}

	if fields["element_typeid"] != nil {
		err = json.Unmarshal(fields["element_typeid"], &node.ElementTypeid)
		if err != nil {
			return
		}
	}

	if fields["elements"] != nil {
		node.Elements.Items, err = UnmarshalNodeArrayJSON(fields["elements"])
		if err != nil {
			return
		}
	}

	if fields["multidims"] != nil {
		err = json.Unmarshal(fields["multidims"], &node.Multidims)
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
