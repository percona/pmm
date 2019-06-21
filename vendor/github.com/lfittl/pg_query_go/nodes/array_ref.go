// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------
 *	ArrayRef: describes an array subscripting operation
 *
 * An ArrayRef can describe fetching a single element from an array,
 * fetching a subarray (array slice), storing a single element into
 * an array, or storing a slice.  The "store" cases work with an
 * initial array value and a source value that is inserted into the
 * appropriate part of the array; the result of the operation is an
 * entire new modified array value.
 *
 * If reflowerindexpr = NIL, then we are fetching or storing a single array
 * element at the subscripts given by refupperindexpr.  Otherwise we are
 * fetching or storing an array slice, that is a rectangular subarray
 * with lower and upper bounds given by the index expressions.
 * reflowerindexpr must be the same length as refupperindexpr when it
 * is not NIL.
 *
 * In the slice case, individual expressions in the subscript lists can be
 * NULL, meaning "substitute the array's current lower or upper bound".
 *
 * Note: the result datatype is the element type when fetching a single
 * element; but it is the array type when doing subarray fetch or either
 * type of store.
 *
 * Note: for the cases where an array is returned, if refexpr yields a R/W
 * expanded array, then the implementation is allowed to modify that object
 * in-place and return the same object.)
 * ----------------
 */
type ArrayRef struct {
	Xpr             Node  `json:"xpr"`
	Refarraytype    Oid   `json:"refarraytype"`    /* type of the array proper */
	Refelemtype     Oid   `json:"refelemtype"`     /* type of the array elements */
	Reftypmod       int32 `json:"reftypmod"`       /* typmod of the array (and elements too) */
	Refcollid       Oid   `json:"refcollid"`       /* OID of collation, or InvalidOid if none */
	Refupperindexpr List  `json:"refupperindexpr"` /* expressions that evaluate to upper
	 * array indexes */

	Reflowerindexpr List `json:"reflowerindexpr"` /* expressions that evaluate to lower
	 * array indexes, or NIL for single array
	 * element */

	Refexpr Node `json:"refexpr"` /* the expression that evaluates to an array
	 * value */

	Refassgnexpr Node `json:"refassgnexpr"` /* expression for the source value, or NULL if
	 * fetch */
}

func (node ArrayRef) MarshalJSON() ([]byte, error) {
	type ArrayRefMarshalAlias ArrayRef
	return json.Marshal(map[string]interface{}{
		"ArrayRef": (*ArrayRefMarshalAlias)(&node),
	})
}

func (node *ArrayRef) UnmarshalJSON(input []byte) (err error) {
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

	if fields["refarraytype"] != nil {
		err = json.Unmarshal(fields["refarraytype"], &node.Refarraytype)
		if err != nil {
			return
		}
	}

	if fields["refelemtype"] != nil {
		err = json.Unmarshal(fields["refelemtype"], &node.Refelemtype)
		if err != nil {
			return
		}
	}

	if fields["reftypmod"] != nil {
		err = json.Unmarshal(fields["reftypmod"], &node.Reftypmod)
		if err != nil {
			return
		}
	}

	if fields["refcollid"] != nil {
		err = json.Unmarshal(fields["refcollid"], &node.Refcollid)
		if err != nil {
			return
		}
	}

	if fields["refupperindexpr"] != nil {
		node.Refupperindexpr.Items, err = UnmarshalNodeArrayJSON(fields["refupperindexpr"])
		if err != nil {
			return
		}
	}

	if fields["reflowerindexpr"] != nil {
		node.Reflowerindexpr.Items, err = UnmarshalNodeArrayJSON(fields["reflowerindexpr"])
		if err != nil {
			return
		}
	}

	if fields["refexpr"] != nil {
		node.Refexpr, err = UnmarshalNodeJSON(fields["refexpr"])
		if err != nil {
			return
		}
	}

	if fields["refassgnexpr"] != nil {
		node.Refassgnexpr, err = UnmarshalNodeJSON(fields["refassgnexpr"])
		if err != nil {
			return
		}
	}

	return
}
