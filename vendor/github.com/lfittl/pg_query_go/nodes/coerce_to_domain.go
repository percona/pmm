// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * CoerceToDomain
 *
 * CoerceToDomain represents the operation of coercing a value to a domain
 * type.  At runtime (and not before) the precise set of constraints to be
 * checked will be determined.  If the value passes, it is returned as the
 * result; if not, an error is raised.  Note that this is equivalent to
 * RelabelType in the scenario where no constraints are applied.
 */
type CoerceToDomain struct {
	Xpr            Node         `json:"xpr"`
	Arg            Node         `json:"arg"`            /* input expression */
	Resulttype     Oid          `json:"resulttype"`     /* domain type ID (result type) */
	Resulttypmod   int32        `json:"resulttypmod"`   /* output typmod (currently always -1) */
	Resultcollid   Oid          `json:"resultcollid"`   /* OID of collation, or InvalidOid if none */
	Coercionformat CoercionForm `json:"coercionformat"` /* how to display this node */
	Location       int          `json:"location"`       /* token location, or -1 if unknown */
}

func (node CoerceToDomain) MarshalJSON() ([]byte, error) {
	type CoerceToDomainMarshalAlias CoerceToDomain
	return json.Marshal(map[string]interface{}{
		"CoerceToDomain": (*CoerceToDomainMarshalAlias)(&node),
	})
}

func (node *CoerceToDomain) UnmarshalJSON(input []byte) (err error) {
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

	if fields["coercionformat"] != nil {
		err = json.Unmarshal(fields["coercionformat"], &node.Coercionformat)
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
