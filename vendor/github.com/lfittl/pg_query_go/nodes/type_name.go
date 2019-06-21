// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * TypeName - specifies a type in definitions
 *
 * For TypeName structures generated internally, it is often easier to
 * specify the type by OID than by name.  If "names" is NIL then the
 * actual type OID is given by typeOid, otherwise typeOid is unused.
 * Similarly, if "typmods" is NIL then the actual typmod is expected to
 * be prespecified in typemod, otherwise typemod is unused.
 *
 * If pct_type is TRUE, then names is actually a field name and we look up
 * the type of that field.  Otherwise (the normal case), names is a type
 * name possibly qualified with schema and database name.
 */
type TypeName struct {
	Names       List  `json:"names"`       /* qualified name (list of Value strings) */
	TypeOid     Oid   `json:"typeOid"`     /* type identified by OID */
	Setof       bool  `json:"setof"`       /* is a set? */
	PctType     bool  `json:"pct_type"`    /* %TYPE specified? */
	Typmods     List  `json:"typmods"`     /* type modifier expression(s) */
	Typemod     int32 `json:"typemod"`     /* prespecified type modifier */
	ArrayBounds List  `json:"arrayBounds"` /* array bounds */
	Location    int   `json:"location"`    /* token location, or -1 if unknown */
}

func (node TypeName) MarshalJSON() ([]byte, error) {
	type TypeNameMarshalAlias TypeName
	return json.Marshal(map[string]interface{}{
		"TypeName": (*TypeNameMarshalAlias)(&node),
	})
}

func (node *TypeName) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["names"] != nil {
		node.Names.Items, err = UnmarshalNodeArrayJSON(fields["names"])
		if err != nil {
			return
		}
	}

	if fields["typeOid"] != nil {
		err = json.Unmarshal(fields["typeOid"], &node.TypeOid)
		if err != nil {
			return
		}
	}

	if fields["setof"] != nil {
		err = json.Unmarshal(fields["setof"], &node.Setof)
		if err != nil {
			return
		}
	}

	if fields["pct_type"] != nil {
		err = json.Unmarshal(fields["pct_type"], &node.PctType)
		if err != nil {
			return
		}
	}

	if fields["typmods"] != nil {
		node.Typmods.Items, err = UnmarshalNodeArrayJSON(fields["typmods"])
		if err != nil {
			return
		}
	}

	if fields["typemod"] != nil {
		err = json.Unmarshal(fields["typemod"], &node.Typemod)
		if err != nil {
			return
		}
	}

	if fields["arrayBounds"] != nil {
		node.ArrayBounds.Items, err = UnmarshalNodeArrayJSON(fields["arrayBounds"])
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
