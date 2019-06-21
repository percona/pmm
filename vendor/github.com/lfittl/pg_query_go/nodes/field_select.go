// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------
 * FieldSelect
 *
 * FieldSelect represents the operation of extracting one field from a tuple
 * value.  At runtime, the input expression is expected to yield a rowtype
 * Datum.  The specified field number is extracted and returned as a Datum.
 * ----------------
 */
type FieldSelect struct {
	Xpr        Node       `json:"xpr"`
	Arg        Node       `json:"arg"`        /* input expression */
	Fieldnum   AttrNumber `json:"fieldnum"`   /* attribute number of field to extract */
	Resulttype Oid        `json:"resulttype"` /* type of the field (result type of this
	 * node) */

	Resulttypmod int32 `json:"resulttypmod"` /* output typmod (usually -1) */
	Resultcollid Oid   `json:"resultcollid"` /* OID of collation of the field */
}

func (node FieldSelect) MarshalJSON() ([]byte, error) {
	type FieldSelectMarshalAlias FieldSelect
	return json.Marshal(map[string]interface{}{
		"FieldSelect": (*FieldSelectMarshalAlias)(&node),
	})
}

func (node *FieldSelect) UnmarshalJSON(input []byte) (err error) {
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

	if fields["fieldnum"] != nil {
		err = json.Unmarshal(fields["fieldnum"], &node.Fieldnum)
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

	return
}
