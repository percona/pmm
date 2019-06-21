// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* ----------------
 * RelabelType
 *
 * RelabelType represents a "dummy" type coercion between two binary-
 * compatible datatypes, such as reinterpreting the result of an OID
 * expression as an int4.  It is a no-op at runtime; we only need it
 * to provide a place to store the correct type to be attributed to
 * the expression result during type resolution.  (We can't get away
 * with just overwriting the type field of the input expression node,
 * so we need a separate node to show the coercion's result type.)
 * ----------------
 */
type RelabelType struct {
	Xpr           Node         `json:"xpr"`
	Arg           Node         `json:"arg"`           /* input expression */
	Resulttype    Oid          `json:"resulttype"`    /* output type of coercion expression */
	Resulttypmod  int32        `json:"resulttypmod"`  /* output typmod (usually -1) */
	Resultcollid  Oid          `json:"resultcollid"`  /* OID of collation, or InvalidOid if none */
	Relabelformat CoercionForm `json:"relabelformat"` /* how to display this node */
	Location      int          `json:"location"`      /* token location, or -1 if unknown */
}

func (node RelabelType) MarshalJSON() ([]byte, error) {
	type RelabelTypeMarshalAlias RelabelType
	return json.Marshal(map[string]interface{}{
		"RelabelType": (*RelabelTypeMarshalAlias)(&node),
	})
}

func (node *RelabelType) UnmarshalJSON(input []byte) (err error) {
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

	if fields["relabelformat"] != nil {
		err = json.Unmarshal(fields["relabelformat"], &node.Relabelformat)
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
