// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/* Symbols for the indexes of the special RTE entries in rules */
type Var struct {
	Xpr   Node  `json:"xpr"`
	Varno Index `json:"varno"` /* index of this var's relation in the range
	 * table, or INNER_VAR/OUTER_VAR/INDEX_VAR */

	Varattno AttrNumber `json:"varattno"` /* attribute number of this var, or zero for
	 * all */

	Vartype     Oid   `json:"vartype"`     /* pg_type OID for the type of this var */
	Vartypmod   int32 `json:"vartypmod"`   /* pg_attribute typmod value */
	Varcollid   Oid   `json:"varcollid"`   /* OID of collation, or InvalidOid if none */
	Varlevelsup Index `json:"varlevelsup"` /* for subquery variables referencing outer
	 * relations; 0 in a normal var, >0 means N
	 * levels up */

	Varnoold  Index      `json:"varnoold"`  /* original value of varno, for debugging */
	Varoattno AttrNumber `json:"varoattno"` /* original value of varattno */
	Location  int        `json:"location"`  /* token location, or -1 if unknown */
}

func (node Var) MarshalJSON() ([]byte, error) {
	type VarMarshalAlias Var
	return json.Marshal(map[string]interface{}{
		"Var": (*VarMarshalAlias)(&node),
	})
}

func (node *Var) UnmarshalJSON(input []byte) (err error) {
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

	if fields["varno"] != nil {
		err = json.Unmarshal(fields["varno"], &node.Varno)
		if err != nil {
			return
		}
	}

	if fields["varattno"] != nil {
		err = json.Unmarshal(fields["varattno"], &node.Varattno)
		if err != nil {
			return
		}
	}

	if fields["vartype"] != nil {
		err = json.Unmarshal(fields["vartype"], &node.Vartype)
		if err != nil {
			return
		}
	}

	if fields["vartypmod"] != nil {
		err = json.Unmarshal(fields["vartypmod"], &node.Vartypmod)
		if err != nil {
			return
		}
	}

	if fields["varcollid"] != nil {
		err = json.Unmarshal(fields["varcollid"], &node.Varcollid)
		if err != nil {
			return
		}
	}

	if fields["varlevelsup"] != nil {
		err = json.Unmarshal(fields["varlevelsup"], &node.Varlevelsup)
		if err != nil {
			return
		}
	}

	if fields["varnoold"] != nil {
		err = json.Unmarshal(fields["varnoold"], &node.Varnoold)
		if err != nil {
			return
		}
	}

	if fields["varoattno"] != nil {
		err = json.Unmarshal(fields["varoattno"], &node.Varoattno)
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
