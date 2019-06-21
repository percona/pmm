// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------
 * CoerceViaIO
 *
 * CoerceViaIO represents a type coercion between two types whose textual
 * representations are compatible, implemented by invoking the source type's
 * typoutput function then the destination type's typinput function.
 * ----------------
 */
type CoerceViaIO struct {
	Xpr        Node `json:"xpr"`
	Arg        Node `json:"arg"`        /* input expression */
	Resulttype Oid  `json:"resulttype"` /* output type of coercion */

	/* output typmod is not stored, but is presumed -1 */
	Resultcollid Oid          `json:"resultcollid"` /* OID of collation, or InvalidOid if none */
	Coerceformat CoercionForm `json:"coerceformat"` /* how to display this node */
	Location     int          `json:"location"`     /* token location, or -1 if unknown */
}

func (node CoerceViaIO) MarshalJSON() ([]byte, error) {
	type CoerceViaIOMarshalAlias CoerceViaIO
	return json.Marshal(map[string]interface{}{
		"CoerceViaIO": (*CoerceViaIOMarshalAlias)(&node),
	})
}

func (node *CoerceViaIO) UnmarshalJSON(input []byte) (err error) {
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

	if fields["resultcollid"] != nil {
		err = json.Unmarshal(fields["resultcollid"], &node.Resultcollid)
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
