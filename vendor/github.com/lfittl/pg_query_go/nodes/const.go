// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * Const
 *
 * Note: for varlena data types, we make a rule that a Const node's value
 * must be in non-extended form (4-byte header, no compression or external
 * references).  This ensures that the Const node is self-contained and makes
 * it more likely that equal() will see logically identical values as equal.
 */
type Const struct {
	Xpr         Node  `json:"xpr"`
	Consttype   Oid   `json:"consttype"`   /* pg_type OID of the constant's datatype */
	Consttypmod int32 `json:"consttypmod"` /* typmod value, if any */
	Constcollid Oid   `json:"constcollid"` /* OID of collation, or InvalidOid if none */
	Constlen    int   `json:"constlen"`    /* typlen of the constant's datatype */
	Constvalue  Datum `json:"constvalue"`  /* the constant's value */
	Constisnull bool  `json:"constisnull"` /* whether the constant is null (if true,
	 * constvalue is undefined) */

	Constbyval bool `json:"constbyval"` /* whether this datatype is passed by value.
	 * If true, then all the information is stored
	 * in the Datum. If false, then the Datum
	 * contains a pointer to the information. */

	Location int `json:"location"` /* token location, or -1 if unknown */
}

func (node Const) MarshalJSON() ([]byte, error) {
	type ConstMarshalAlias Const
	return json.Marshal(map[string]interface{}{
		"Const": (*ConstMarshalAlias)(&node),
	})
}

func (node *Const) UnmarshalJSON(input []byte) (err error) {
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

	if fields["consttype"] != nil {
		err = json.Unmarshal(fields["consttype"], &node.Consttype)
		if err != nil {
			return
		}
	}

	if fields["consttypmod"] != nil {
		err = json.Unmarshal(fields["consttypmod"], &node.Consttypmod)
		if err != nil {
			return
		}
	}

	if fields["constcollid"] != nil {
		err = json.Unmarshal(fields["constcollid"], &node.Constcollid)
		if err != nil {
			return
		}
	}

	if fields["constlen"] != nil {
		err = json.Unmarshal(fields["constlen"], &node.Constlen)
		if err != nil {
			return
		}
	}

	if fields["constvalue"] != nil {
		err = json.Unmarshal(fields["constvalue"], &node.Constvalue)
		if err != nil {
			return
		}
	}

	if fields["constisnull"] != nil {
		err = json.Unmarshal(fields["constisnull"], &node.Constisnull)
		if err != nil {
			return
		}
	}

	if fields["constbyval"] != nil {
		err = json.Unmarshal(fields["constbyval"], &node.Constbyval)
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
