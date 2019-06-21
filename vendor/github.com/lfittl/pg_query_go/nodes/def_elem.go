// Auto-generated from postgres/src/include/nodes/parsenodes.h - DO NOT EDIT

package pg_query

import "encoding/json"

/*
 * DefElem - a generic "name = value" option definition
 *
 * In some contexts the name can be qualified.  Also, certain SQL commands
 * allow a SET/ADD/DROP action to be attached to option settings, so it's
 * convenient to carry a field for that too.  (Note: currently, it is our
 * practice that the grammar allows namespace and action only in statements
 * where they are relevant; C code can just ignore those fields in other
 * statements.)
 */
type DefElem struct {
	Defnamespace *string       `json:"defnamespace"` /* NULL if unqualified name */
	Defname      *string       `json:"defname"`
	Arg          Node          `json:"arg"`       /* a (Value *) or a (TypeName *) */
	Defaction    DefElemAction `json:"defaction"` /* unspecified action, or SET/ADD/DROP */
	Location     int           `json:"location"`  /* token location, or -1 if unknown */
}

func (node DefElem) MarshalJSON() ([]byte, error) {
	type DefElemMarshalAlias DefElem
	return json.Marshal(map[string]interface{}{
		"DefElem": (*DefElemMarshalAlias)(&node),
	})
}

func (node *DefElem) UnmarshalJSON(input []byte) (err error) {
	var fields map[string]json.RawMessage

	err = json.Unmarshal(input, &fields)
	if err != nil {
		return
	}

	if fields["defnamespace"] != nil {
		err = json.Unmarshal(fields["defnamespace"], &node.Defnamespace)
		if err != nil {
			return
		}
	}

	if fields["defname"] != nil {
		err = json.Unmarshal(fields["defname"], &node.Defname)
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

	if fields["defaction"] != nil {
		err = json.Unmarshal(fields["defaction"], &node.Defaction)
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
