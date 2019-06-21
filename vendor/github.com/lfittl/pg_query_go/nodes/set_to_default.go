// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * Placeholder node for a DEFAULT marker in an INSERT or UPDATE command.
 *
 * This is not an executable expression: it must be replaced by the actual
 * column default expression during rewriting.  But it is convenient to
 * treat it as an expression node during parsing and rewriting.
 */
type SetToDefault struct {
	Xpr       Node  `json:"xpr"`
	TypeId    Oid   `json:"typeId"`    /* type for substituted value */
	TypeMod   int32 `json:"typeMod"`   /* typemod for substituted value */
	Collation Oid   `json:"collation"` /* collation for the substituted value */
	Location  int   `json:"location"`  /* token location, or -1 if unknown */
}

func (node SetToDefault) MarshalJSON() ([]byte, error) {
	type SetToDefaultMarshalAlias SetToDefault
	return json.Marshal(map[string]interface{}{
		"SetToDefault": (*SetToDefaultMarshalAlias)(&node),
	})
}

func (node *SetToDefault) UnmarshalJSON(input []byte) (err error) {
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

	if fields["typeId"] != nil {
		err = json.Unmarshal(fields["typeId"], &node.TypeId)
		if err != nil {
			return
		}
	}

	if fields["typeMod"] != nil {
		err = json.Unmarshal(fields["typeMod"], &node.TypeMod)
		if err != nil {
			return
		}
	}

	if fields["collation"] != nil {
		err = json.Unmarshal(fields["collation"], &node.Collation)
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
